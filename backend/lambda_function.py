import os
import json
import base64
import logging
import urllib.request
from urllib.error import URLError, HTTPError

logger = logging.getLogger()
logger.setLevel(logging.INFO)

# The backend service (ECS Fargate) endpoint should be provided via env var
BACKEND_URL = os.getenv('BACKEND_URL')
if not BACKEND_URL:
    logger.error('Environment variable BACKEND_URL is not set')
    raise RuntimeError('BACKEND_URL not configured')

def lambda_handler(event, context):
    """Proxy HTTP requests from API Gateway to the decoupled backend service.

    The original Go application expects JSON payloads on its HTTP endpoints.
    This handler forwards the request method, path, query string, headers and
    body unchanged to the backend URL and returns the response back to the
    caller.
    """
    try:
        # Build the target URL preserving path and query string
        path = event.get('rawPath', '/')
        query = event.get('rawQueryString')
        target = f"{BACKEND_URL}{path}"
        if query:
            target = f"{target}?{query}"

        method = event.get('requestContext', {}).get('http', {}).get('method', 'GET')
        headers = {k: v for k, v in event.get('headers', {}).items() if k.lower() != 'host'}

        # Decode body if present (API Gateway may base64‑encode binary payloads)
        body = event.get('body')
        if body is not None:
            if event.get('isBase64Encoded'):
                body = base64.b64decode(body)
            else:
                body = body.encode('utf-8')
        else:
            body = None

        req = urllib.request.Request(url=target, data=body, method=method)
        for k, v in headers.items():
            req.add_header(k, v)

        with urllib.request.urlopen(req) as resp:
            resp_body = resp.read()
            resp_status = resp.getcode()
            resp_headers = dict(resp.getheaders())

        # API Gateway expects the body as a string; if binary, base64‑encode it
        is_binary = any(h.lower().startswith('content-type') and 'application/octet-stream' in v.lower()
                        for h, v in resp_headers.items())
        if is_binary:
            resp_body = base64.b64encode(resp_body).decode('utf-8')
            is_base64 = True
        else:
            resp_body = resp_body.decode('utf-8')
            is_base64 = False

        return {
            'statusCode': resp_status,
            'headers': resp_headers,
            'body': resp_body,
            'isBase64Encoded': is_base64
        }
    except HTTPError as e:
        logger.error(f'HTTP error from backend: {e.code} {e.reason}')
        return {
            'statusCode': e.code,
            'body': e.read().decode('utf-8', errors='replace')
        }
    except URLError as e:
        logger.error(f'Connection error to backend: {e.reason}')
        return {
            'statusCode': 502,
            'body': json.dumps({'error': 'Bad Gateway', 'detail': str(e.reason)})
        }
    except Exception as e:
        logger.exception('Unexpected error in lambda_handler')
        return {
            'statusCode': 500,
            'body': json.dumps({'error': 'Internal Server Error'})
        }
