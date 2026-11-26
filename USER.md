# Golab Backend – Serverless Migration Operator Guide

This guide walks you through deploying the **golab** backend (originally a Go HTTP server) as an AWS Lambda function using the **provided.al2023** custom runtime. The Lambda is fronted by an HTTP API Gateway that proxies all requests to the function.

---
## Prerequisites
- An AWS account with permissions to create IAM roles, Lambda functions, and API Gateway resources.
- AWS CLI installed and configured (`aws configure`).
- The source repository with the generated files:
  - `backend/lambda_function.py` – Python adapter that invokes the compiled Go binary.
  - `backend/` also contains the compiled **bootstrap** binary (built by SAM during deployment).
  - `template.yaml` – AWS SAM template.
- **Optional**: Docker installed locally if you want to build the Go binary yourself before SAM deployment.

---
## 1. Build the Go binary (custom runtime bootstrap)
The SAM template expects a Linux‑ARM64 executable named **bootstrap** in the `backend/` folder.
```bash
cd backend
# Ensure Go is installed (>=1.22) and set the correct env vars for Lambda
GOOS=linux GOARCH=arm64 go build -tags lambda.norpc -o bootstrap .
# Verify the binary runs (it will exit after processing a single request)
chmod +x bootstrap
```
> **Note**: The original application spawns Docker containers – this is not supported in Lambda. Those parts must be off‑loaded to an external worker (e.g., ECS Fargate). The binary you just built should contain only the request‑handling logic that does not depend on Docker.

---
## 2. Package and Deploy with AWS SAM
```bash
# From the repository root (where template.yaml lives)
sam build --use-container   # Builds the Lambda package inside a Docker container (optional)
sam deploy --guided
```
During the guided deployment you will be asked for:
- **Stack name** – e.g., `golab-backend-stack`
- **AWS Region** – choose the region where you want the service.
- **Confirm changes before deploy** – `Y`
- **Allow SAM to create IAM roles** – `Y`
- **Save arguments to a configuration file** – `Y` (creates `samconfig.toml` for future deployments)

SAM will:
1. Upload the `backend/` folder (including `bootstrap` and `lambda_function.py`) to an S3 bucket.
2. Create a Lambda function using the **provided.al2023** runtime, arm64 architecture, and a 15‑second timeout.
3. Provision an HTTP API Gateway that forwards all paths (`/{proxy+}`) to the Lambda.
4. Output the API endpoint URL.

---
## 3. Configure Environment Variables (if needed)
If your Go binary expects additional environment variables (e.g., external service URLs, secrets), add them in the SAM template under `Globals.Function.Environment.Variables` and redeploy, or edit the Lambda configuration directly in the AWS Console:
1. Open the **Lambda** service.
2. Select the **GolabFunction**.
3. Choose **Configuration → Environment variables**.
4. Add key/value pairs and click **Save**.

---
## 4. IAM Role & Permissions
The generated SAM template creates an execution role with basic Lambda permissions. If the function needs to access other AWS services (e.g., S3, SQS, Secrets Manager), attach the appropriate policies to the role:
```bash
aws iam attach-role-policy \
  --role-name <GolabFunction‑Role> \
  --policy-arn arn:aws:iam::aws:policy/AmazonS3ReadOnlyAccess
```
Replace the policy ARN with the one you need.

---
## 5. Testing the Deployment
After deployment, SAM prints the **ApiEndpoint** output. Test with `curl`:
```bash
API_URL=$(sam outputs --stack-name golab-backend-stack --query ApiEndpoint --output text)

# Example request payload (adjust to your API contract)
cat <<EOF > payload.json
{
  "code": "print(\"Hello, world!\")",
  "input": ""
}
EOF

curl -X POST "$API_URL/run" \
  -H "Content-Type: application/json" \
  -d @payload.json
```
You should receive a JSON response mirroring the original Go server's `RunResponse` structure.

---
## 6. Monitoring & Logging
- **CloudWatch Logs**: Each Lambda invocation writes logs to `/aws/lambda/<function‑name>`. Use the console or `aws logs` to view them.
- **Metrics**: Lambda provides duration, error count, and throttles automatically. You can create CloudWatch Alarms on these metrics.

---
## 7. Clean‑up
When you no longer need the service:
```bash
sam delete --stack-name golab-backend-stack
```
This removes the Lambda, API Gateway, IAM role, and any generated S3 artifacts.

---
## 8. Next Steps – Decoupling Docker Workloads
Because Lambda cannot run Docker containers, any functionality that relied on `docker run` must be migrated to a separate worker service (e.g., an **ECS Fargate** task queue). The typical pattern:
1. Lambda receives the request and places a job description onto an SQS queue.
2. An ECS Fargate service polls the queue, runs the Docker container, stores the result (e.g., in S3), and optionally notifies the caller via another queue or API.
3. The Lambda can poll for completion or return a job‑ID for asynchronous handling.

Implementing this decoupling is outside the scope of the current migration but is required for full functionality.
