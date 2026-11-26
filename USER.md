# AWS Deployment Guide: golab

This guide provides step-by-step instructions for deploying your migrated backend service to **AWS Lambda** and **Amazon API Gateway** using both the AWS SAM CLI and the AWS Management Console.

---

## Option 1: One-Click Deployment via AWS SAM CLI (Recommended)

1. Make sure you have the [AWS SAM CLI](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/install-sam-cli.html) installed and configured with your AWS credentials (`aws configure`).
2. Run the build and deployment commands from the repository root:
   ```bash
   sam build
   sam deploy --guided
   ```
3. During the guided prompt:
   - Stack Name: `golab-stack`
   - AWS Region: `us-east-1` (or your preferred region)
   - Allow SAM CLI to create IAM roles: **Yes**
   - Save arguments to configuration file: **Yes**
4. Once deployed, SAM will output your live public HTTP API endpoint:
   `https://<api-id>.execute-api.<region>.amazonaws.com`

---

## Option 2: Manual Setup via AWS Management Console

### Step 1: Create the Lambda Function
1. Open the [AWS Lambda Console](https://console.aws.amazon.com/lambda/).
2. Click **Create function** -> **Author from scratch**.
3. **Function name:** `golab-backend`
4. **Runtime:** 
   - For Go: Choose **Amazon Linux 2023** (`provided.al2023`).
   - For Python: Choose **Python 3.12**.
   - For Node.js: Choose **Node.js 20.x**.
5. **Architecture:** Select **arm64** (Graviton — faster and 20% cheaper).
6. Click **Create function**.

### Step 2: Configure Environment Variables
1. In your new Lambda function, navigate to the **Configuration** tab.
2. Click **Environment variables** on the left menu -> **Edit** -> **Add environment variable**.
3. Add any necessary database connection strings or configuration variables:
   - `ENVIRONMENT` = `production`
   - `PORT` = `8080`
4. Click **Save**.

### Step 3: Add API Gateway HTTP Trigger
1. In the function overview diagram, click **+ Add trigger**.
2. Select **API Gateway**.
3. Trigger type: **Create a new API** -> Choose **HTTP API**.
4. Security: **Open** (or IAM / JWT if authentication is required).
5. Click **Add**.
6. The public URL will appear under **API endpoint** (e.g. `https://xxxx.execute-api.us-east-1.amazonaws.com`).

### Step 4: Test Your Function
1. Navigate to the **Test** tab in the Lambda Console.
2. Select **Create new event** -> Choose the template **apigateway-aws-proxy**.
3. Click **Test**.
4. Verify you receive `HTTP 200 OK` in the execution output!

---

## Frontend Integration
In your frontend configuration (e.g. `frontend/.env.local` or environment settings):
```env
NEXT_PUBLIC_API_URL=https://<your-api-id>.execute-api.<region>.amazonaws.com
```
