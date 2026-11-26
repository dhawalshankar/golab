# Golab Serverless Migration – Operator Deployment Guide

This guide walks you through deploying the **Golab** service after it has been migrated to a serverless proxy (AWS Lambda) that forwards requests to a decoupled backend running on **ECS Fargate**.

---
## Prerequisites
- AWS account with permissions to create IAM roles, Lambda functions, API Gateway, and ECS resources.
- AWS CLI installed and configured (`aws configure`).
- The backend Docker images (`golab-compiler:latest` and `golab-runtime:latest`) are already pushed to Amazon ECR and a Fargate service is running that exposes the original HTTP API on port **3000** (or the ports you used).

---
## 1. Deploy the Lambda & API Gateway via SAM
```bash
# Clone the repository and navigate to the project root
git clone <repo-url>
cd <repo-dir>

# Build and package the SAM application (the Lambda code lives in backend/)
aws cloudformation package \
    --template-file template.yaml \
    --s3-bucket <your-s3-bucket> \
    --output-template-file packaged.yaml

# Deploy the stack
aws cloudformation deploy \
    --template-file packaged.yaml \
    --stack-name golab-proxy-stack \
    --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND \
    --parameter-overrides BackendUrl=https://<your-ecs-service-endpoint>
```
The SAM template creates:
- **GolabProxyFunction** – a Python 3.12 Lambda (arm64) that proxies all HTTP traffic.
- **HTTP API (v2)** – a single catch‑all route (`/{proxy+}`) that forwards any request to the Lambda.

---
## 2. Configure Environment Variables
The Lambda expects the environment variable `BACKEND_URL` to point at the **public** endpoint of the ECS Fargate service (e.g., `https://my-backend-xyz.us-east-1.elb.amazonaws.com`).

If you need to update it later:
```bash
aws lambda update-function-configuration \
    --function-name GolabProxyFunction \
    --environment "Variables={BACKEND_URL=https://<new-endpoint>}"
```

---
## 3. IAM Role & Permissions
The SAM template attaches the managed policy **AWSLambdaBasicExecutionRole** which grants:
- Write logs to CloudWatch.
- Basic Lambda execution rights.

If the Lambda must access other AWS resources (e.g., Secrets Manager for DB credentials), attach additional policies to the generated role `GolabProxyFunctionRole`.

---
## 4. Verify the Deployment
1. Retrieve the API endpoint:
   ```bash
   aws cloudformation describe-stacks \
       --stack-name golab-proxy-stack \
       --query "Stacks[0].Outputs[?OutputKey=='FunctionArn'].OutputValue" \
       --output text
   ```
   The output will be something like `https://abcde12345.execute-api.<region>.amazonaws.com`.
2. Test a simple request (replace `<api>` with the URL above):
   ```bash
   curl https://<api>/run -d '{"code":"print(\"hello\")","input":""}' -H "Content-Type: application/json"
   ```
   The request is proxied to the ECS backend and the response should match the original EC2 service.

---
## 5. Monitoring & Logging
- **CloudWatch Logs**: Lambda logs are automatically streamed to `/aws/lambda/GolabProxyFunction`.
- **X‑Ray (optional)**: Enable tracing in the SAM template (`Tracing: Active`) if you need end‑to‑end latency visibility.

---
## 6. Clean‑up
When the service is no longer needed, delete the CloudFormation stack:
```bash
aws cloudformation delete-stack --stack-name golab-proxy-stack
```
This removes the Lambda, API Gateway, and associated IAM role. Remember to shut down the ECS Fargate service separately if you no longer need it.

---
## 7. Next Steps / Decoupling Details
Because the original application spawns Docker containers (`docker run`, `docker.sock`), those operations have been moved to an **ECS Fargate** worker pool. The Lambda only acts as a thin HTTP proxy, eliminating the blocker for serverless execution while preserving the existing API contract.

If you need to scale the backend, adjust the Fargate service's task count or CPU/memory configuration in its own CloudFormation/ECS definition.

---
**Happy Deploying!**
