import json
import os
import subprocess
import sys
from typing import Any, Dict

# NOTE:
# This adapter is a thin wrapper that runs the compiled Go binary (bootstrap) in the same
# Lambda execution environment. The binary is built separately (see the SAM template) and
# placed alongside this file. The handler receives an API‑Gateway proxy event, extracts the
# request body and forwards it to the Go binary via stdin/stdout using a subprocess.
# The Go binary implements the original HTTP server logic (listening on port 3000). In a
# Lambda environment we invoke it in "one‑shot" mode – it reads a single JSON payload from
# stdin, processes it, writes a JSON response to stdout and exits.
#
# If the binary cannot be executed (e.g., missing, permission issues) the handler returns a
# 500 error with a helpful message.

GO_BINARY_PATH = "/var/task/bootstrap"  # SAM places the compiled binary here


def _run_go_binary(payload: Dict[str, Any]) -> Dict[str, Any]:
    """Execute the compiled Go binary with the given JSON payload.

    The binary is expected to read a JSON object from stdin and write a JSON object to
    stdout. Any non‑zero exit code is treated as an error.
    """
    try:
        # Serialize payload to JSON bytes
        input_bytes = json.dumps(payload).encode("utf-8")
        # Run the binary; capture stdout and stderr
        result = subprocess.run(
            [GO_BINARY_PATH],
            input=input_bytes,
            capture_output=True,
            timeout=10,  # seconds – matches the original runTimeout
        )
    except FileNotFoundError:
        raise RuntimeError(f"Go binary not found at {GO_BINARY_PATH}")
    except subprocess.TimeoutExpired:
        raise RuntimeError("Go binary execution timed out")
    except Exception as exc:
        raise RuntimeError(f"Unexpected error invoking Go binary: {exc}")

    if result.returncode != 0:
        # Include stderr for debugging
        raise RuntimeError(
            f"Go binary exited with code {result.returncode}: {result.stderr.decode('utf-8', errors='ignore')}"
        )

    try:
        output = json.loads(result.stdout)
    except json.JSONDecodeError as exc:
        raise RuntimeError(f"Failed to decode JSON from Go binary: {exc}")

    return output


def lambda_handler(event: Dict[str, Any], context: Any) -> Dict[str, Any]:
    """AWS Lambda entry point.

    The function expects an API Gateway HTTP API proxy event. The request body is passed
    directly to the Go binary. The response from the binary is returned to the caller.
    """
    # API Gateway may deliver the body as a string (already JSON) or base64‑encoded.
    body = event.get("body", "")
    if event.get("isBase64Encoded", False):
        import base64
        body = base64.b64decode(body).decode("utf-8")

    try:
        payload = json.loads(body) if body else {}
    except json.JSONDecodeError:
        return {
            "statusCode": 400,
            "body": json.dumps({"error": "Invalid JSON payload"}),
            "headers": {"Content-Type": "application/json"},
        }

    try:
        response_payload = _run_go_binary(payload)
        return {
            "statusCode": 200,
            "body": json.dumps(response_payload),
            "headers": {"Content-Type": "application/json"},
        }
    except RuntimeError as err:
        # Log the error for CloudWatch visibility
        print(f"Error invoking Go binary: {err}", file=sys.stderr)
        return {
            "statusCode": 500,
            "body": json.dumps({"error": str(err)}),
            "headers": {"Content-Type": "application/json"},
        }
