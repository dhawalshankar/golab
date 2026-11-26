# EC2 TO LAMBDA MIGRATION REPORT
**Audited Application:** `golab`  
**Audit Tool:** DAROGA Autonomous Infrastructure Inspector & Migration Auditor  
**Generated Date:** 2026-11-26 23:23:50 UTC  
**AWS Account ID:** `196591916069`  
**AWS Region:** `ap-southeast-2`  

---

## 1. Executive Summary
This report presents an application-specific, architectural migration assessment for transitioning `golab` from an Amazon EC2 virtual machine hosting model to **AWS Lambda Serverless Architecture**.

* **Target Service Stack:** Go (Standard net/http (Go))
* **Target Migration Scope:** `backend` backend service
* **Migration Suitability Verdict:** **MIGRATE (WITH DECOUPLING)** (Compatibility Score: **55%**)
* **Baseline EC2 Cost:** **$9.99/month** (Live Instance: GoLab-Runner (i-01f086c5a44746100, t3.micro))
* **Projected Serverless Lambda Cost (Low/Moderate Traffic):** **$0.00 – $6.80/month**
* **Potential Monthly Infrastructure Savings:** **100.0%** ($9.99/month)
* **Blockers Detected:** Dynamic Docker execution (docker run, docker.sock) is unsupported inside AWS Lambda.
* **Required Architectural Conditions:** Decouple dynamic container execution to AWS ECS Fargate task worker

> **Important Operational Guardrail:** This report is prepared by an autonomous auditor. In accordance with safety restrictions, **no production infrastructure was modified and no code was deployed**. The user performs all actual implementation steps manually using the copy-paste instructions in Section 16 & 17.

---

## 2. Existing Application Architecture
* **Target Service Language & Runtime:** Go (Recommended Lambda Runtime: `provided.al2023`)
* **Repo-Wide Languages:** javascript, go, typescript
* **Web Framework:** Standard net/http (Go)
* **Entry Point(s):** backend/main.go
* **Exposed Ports:** 3000, 10001, 10485
* **Continuous Execution Model:** The application currently runs as a persistent listening daemon (managed via systemd, PM2, or Docker) on an EC2 instance. In the serverless model, this persistent socket listener is superseded by an event-driven AWS API Gateway HTTP trigger.

---

## 3. Repository Analysis
* **Total Files Scanned:** 23 files inspected deterministically.
* **Package Managers & Manifests:** go.mod, package.json
* **Container Specifications:** backend/sandbox/compiler.Dockerfile, backend/sandbox/runtime.Dockerfile
* **Web Server / Reverse Proxy Configs:** None (direct port binding)
* **CI/CD Workflows:** None detected
* **Database Integrations:** None detected / Stateless API

---

## 4. Existing EC2 Configuration
* **Instance ID:** `i-01f086c5a44746100`
* **Instance Type:** `t3.micro`
* **State:** `running`
* **AMI ID:** `ami-0eeab0e1473986ffd`
* **Platform Details:** `Linux/UNIX`
* **Attached EBS Volumes:** `vol-072c16e1e280a7ac3`
* **IAM Instance Profile:** `None`

---

## 5. Existing AWS Dependencies
* **Core Compute:** Amazon EC2 Virtual Machine
* **Storage:** Amazon EBS gp3 Elastic Block Store
* **Networking:** Amazon VPC, Internet Gateway, Security Groups
* **Auxiliary Services Inspected:** None required
* **Browser CDP Cross-Check:** CDP not connected: Chrome DevTools Protocol (CDP) is not listening on http://127.0.0.1:9222. To enable live browser console inspection, launch Chrome with: `chrome.exe --remote-debugging-port=9222` (preserving your existing user profile).

---

## 6. Existing Networking Architecture
* **Virtual Private Cloud (VPC):** `vpc-04367ea1807d6e576`
* **Subnet ID:** `subnet-0d28f0a2af40a959b`
* **Security Group Associations:** `sg-0028040e2c1e1e79b`
* **Ingress Traffic Flow:** Currently allows direct inbound traffic on ports `3000, 10001, 10485` to the EC2 network interface.
* **Target Serverless Flow:** Public traffic hits Amazon API Gateway HTTPS endpoints, which invokes AWS Lambda within the private VPC subnets.

---

## 7. Relevant IAM Configuration
* **Existing EC2 Role:** `None (uses default AWS credentials or instance metadata)`
* **Required Lambda Execution Role:** `golab-LambdaExecutionRole`
* **Trust Relationship:** `lambda.amazonaws.com`
* **Managed Policies to Attach:**
- `arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole`
- `arn:aws:iam::aws:policy/service-role/AWSLambdaVPCAccessExecutionRole`

---

## 8. Lambda Compatibility Assessment

| Rule ID | Category | Assessment Finding | Compatibility Classification | Severity | Recommended Action |
| :--- | :--- | :--- | :--- | :--- | :--- |
| `RULE-RNT-01` | Runtime Compatibility | Golang Custom Runtime on Amazon Linux 2023 (provided.al2023) | Compatible with code modifications | **LOW** | Compile binary for Linux ARM64 (bootstrap) and package using custom runtime 'provided.al2023'. |
| `RULE-NET-01` | Application Execution Model | Continuous TCP Socket Listener Conversion | Compatible with code modifications | **MEDIUM** | Replace continuous `http.ListenAndServe` with event-driven invocation via Amazon API Gateway HTTP API ($default or /{proxy+}). |
| `RULE-CNT-01` | Container Runtime | Docker-in-Lambda is Unsupported (Dynamic Container Execution Detected) | Not directly compatible | **HIGH** | AWS Lambda runs inside lightweight Firecracker microVMs with a read-only root filesystem and cannot run a Docker daemon or execute nested containers. Recommend AWS Lambda for the public API layer plus an asynchronous AWS ECS Fargate task (or AWS Batch) triggered via Amazon SQS or EventBridge for dynamic container execution. |
| `RULE-LIM-01` | Execution Limits | 15-Minute Timeout Limit (Long-Running Tasks Detected) | Not directly compatible | **HIGH** | AWS Lambda enforces a hard 15-minute (900s) maximum execution duration. Long-running compilation or batch tasks must be decoupled to ECS Fargate. |
| `RULE-LIM-02` | Execution Limits | 6 MB Synchronous Payload Limit | Compatible without changes | **INFO** | AWS Lambda enforces a 6 MB maximum payload limit for synchronous API Gateway requests and responses. For large file uploads or outputs, generate Amazon S3 presigned URLs. |
| `RULE-LIM-03` | Execution Limits | Ephemeral Disk Storage (/tmp 512MB-10GB) | Compatible without changes | **INFO** | Ephemeral storage (/tmp) is available from 512 MB up to 10 GB. Data resets between warm container life cycles. |

---

## 9. Migration Blockers
- ⚠️ **Dynamic Docker execution (docker run, docker.sock) is unsupported inside AWS Lambda.**

---

## 10. Proposed Lambda Architecture
The recommended target architecture replaces the 24/7 EC2 server process with an event-driven serverless topology:

```
[Client / Browser / Frontend]
              │
              ▼ HTTPS
[Amazon API Gateway (HTTP API v2)]
              │
              ▼ Event Payload (APIGatewayV2HTTPRequest)
[AWS Lambda Function (arm64 Graviton)]
        │                    │
        ▼                    ▼
[Amazon CloudWatch Logs]  [Existing Database / VPC Resources]
```

---

## 11. Required AWS Resources

| AWS Service | Architectural Role | Replaces EC2 Responsibility | Cost Impact |
| :--- | :--- | :--- | :--- |
| **AWS Lambda (arm64 Graviton)** | Executes Go backend API logic on-demand without persistent EC2 instances. | Replaces 24/7 EC2 virtual machine CPU, memory, and continuous OS host. | Pay-per-millisecond (First 1,000,000 requests and 3,200,000 compute-seconds per month are 100% FREE). |
| **Amazon API Gateway (HTTP API v2)** | Exposes live public HTTPS endpoints and routes client requests directly to the Lambda function. | Replaces EC2 local port listeners, reverse proxies (Nginx/Caddy), and public IP port routing. | $1.00 per 1,000,000 requests after AWS Free Tier. |
| **Amazon CloudWatch Logs** | Aggregates application stdout/stderr logs and execution telemetry. | Replaces local log files (`/var/log/...`), journald, and logrotate on the EC2 EBS volume. | 5 GB ingestion free tier per month; $0.50 per GB ingested thereafter. |
| **AWS Identity and Access Management (IAM)** | Grants Lambda least-privilege permission to emit logs, access VPC networking, and query databases. | Replaces EC2 IAM Instance Profile attached to the virtual machine. | Free of charge (IAM features are always free). |

---

## 12. IAM Execution Role Requirements
Create an execution role for Lambda with least-privilege permissions:
* **Role Name:** `golab-LambdaExecutionRole`
* **Trust Policy:**
```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "lambda.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
```

---

## 13. API Gateway Configuration
* **Type:** HTTP API (API Gateway v2) — provides lowest latency and 70% cost savings compared to REST APIs.
* **Integrations:** AWS Lambda proxy integration (`payloadFormatVersion: "2.0"`).
* **Routes to Configure:**
  - `ANY /` -> Integrates with Lambda
  - `ANY /{proxy+}` -> Integrates with Lambda catch-all
* **CORS Settings:** Allowed Origins: `*` (or your production frontend domain), Allowed Methods: `GET, POST, PUT, DELETE, OPTIONS`.

---

## 14. Networking and Database Configuration
* **VPC Attachment:** If connecting to private RDS or internal VPC caches, attach Lambda to VPC `vpc-04367ea1807d6e576` across subnets `subnet-0d28f0a2af40a959b`.
* **Security Group:** Assign a security group allowing outbound port access (e.g. `5432` for Postgres, `3306` for MySQL) into the database security group.
* **Connection Lifecycle:** In serverless architectures, instantiate database connection pools outside the main request handler (in global initialization scope) to reuse database sockets across warm invocations.

---

## 15. Required Application Code Changes
To run on AWS Lambda, the user must implement the following serverless entrypoint adjustments:

1. **Wrap HTTP router in `backend/lambda_main.go` using `aws-lambda-go-api-proxy` or standard Lambda handler.**
2. **Decouple `docker run` container execution into an asynchronous ECS Fargate or Lambda container task.**

*Example Serverless Entrypoint (Golang):*
```go
package main

import (
    "context"
    "net/http"
    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

var adapter *httpadapter.HandlerAdapter

func init() {
    adapter = httpadapter.New(http.DefaultServeMux)
}

func Handler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
    return adapter.ProxyWithContext(ctx, req)
}

func main() {
    lambda.Start(Handler)
}
```

---

## 16. Step-by-Step AWS Console Instructions

### Step 1: Create IAM Role in AWS Console
1. Open the [AWS IAM Console -> Roles](https://console.aws.amazon.com/iam/home#/roles).
2. Click **Create role** -> Select **AWS service** -> Use case: **Lambda** -> Click **Next**.
3. Under Add permissions, search and select:
   - `AWSLambdaBasicExecutionRole`
   - `AWSLambdaVPCAccessExecutionRole` (if VPC database connectivity is required)
4. Role name: `golab-LambdaExecutionRole` -> Click **Create role**.

### Step 2: Create the Lambda Function in AWS Console
1. Open the [AWS Lambda Console](https://console.aws.amazon.com/lambda/home#/functions).
2. Click **Create function** -> **Author from scratch**.
3. **Function name:** `golab-backend`
4. **Runtime:** Select `provided.al2023`.
5. **Architecture:** Select **arm64** (Graviton2/3).
6. Under Permissions, select **Use an existing role** -> Choose `golab-LambdaExecutionRole`.
7. Click **Create function**.

### Step 3: Configure Function Timeout & Memory
1. In the function details, open the **Configuration** tab -> **General configuration** -> Click **Edit**.
2. Set **Memory:** `512 MB`.
3. Set **Timeout:** `30 sec`.
4. Click **Save**.

### Step 4: Configure API Gateway HTTP Trigger
1. In the Lambda function overview diagram, click **+ Add trigger**.
2. Select **API Gateway**.
3. Intent: **Create a new API** -> API type: **HTTP API**.
4. Security: **Open** (or configure JWT / IAM authorizer).
5. Click **Add**.
6. The public URL will display under **API endpoint** (e.g. `https://xxxx.execute-api.ap-southeast-2.amazonaws.com`).

---

## 17. Copy-Paste-Ready AWS CLI Commands

> **Notice:** All commands below are for manual execution by the user. The agent does not execute modification commands.

### Inspection Commands (Read-Only)
```powershell
# Verify current AWS identity and active region
aws sts get-caller-identity

# Inspect existing EC2 instance configuration
aws ec2 describe-instances --instance-ids i-01f086c5a44746100 --region ap-southeast-2

# Inspect associated security groups
aws ec2 describe-security-groups --group-ids sg-0028040e2c1e1e79b --region ap-southeast-2
```

### Manual Deployment Commands (User-Executed)
```powershell
# 1. Create the Lambda Execution Role
aws iam create-role `
  --role-name golab-LambdaExecutionRole `
  --assume-role-policy-document '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]}'

# 2. Attach Basic Execution Policy
aws iam attach-role-policy `
  --role-name golab-LambdaExecutionRole `
  --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole

# 3. Create the Lambda Function (replace deployment.zip with your compiled zip)
aws lambda create-function `
  --function-name golab-backend `
  --runtime provided.al2023 `
  --role arn:aws:iam::196591916069:role/golab-LambdaExecutionRole `
  --handler bootstrap `
  --architectures arm64 `
  --timeout 30 `
  --memory-size 512 `
  --zip-file fileb://deployment.zip `
  --region ap-southeast-2
```

---

## 18. Cost and Free-Tier Assessment

### Cost Comparison Table

| Service / Resource | Legacy EC2 Virtual Machine | Proposed AWS Lambda Serverless |
| :--- | :--- | :--- |
| **Compute Charges** | `$9.99/month` (24/7 t3.micro) | `$0.00/month` (Within 3.2M free GB-seconds) |
| **Request Charges** | N/A (included in VM) | `$0.00/month` (Within 1M free requests) |
| **API Gateway Ingress** | N/A (direct TCP ingress) | `$0.00/month` (<1M free calls) or ~$1.00/M |
| **Disk Storage** | ~$2.40/month (30 GB gp3 volume) | `$0.00/month` (512MB ephemeral `/tmp` included) |
| **Monthly Total** | **$9.99/month** | **$0.00 – $6.80/month** |

* **AWS Free Tier Allowances Applicable:**
  - 1,000,000 Lambda invocations free per month (Permanent).
  - 3,200,000 GB-seconds of compute time per month (Permanent).
  - 1,000,000 HTTP API calls free per month (First 12 months).
  - 5 GB CloudWatch Logs ingestion free per month (Permanent).

---

## 19. Verification Checklist
- [ ] Lambda function created with runtime `provided.al2023` and architecture `arm64`.
- [ ] IAM execution role `golab-LambdaExecutionRole` attached with basic logging permissions.
- [ ] API Gateway HTTP trigger mapped to `ANY /{proxy+}` routes.
- [ ] Database connection URI and secrets added under Configuration -> Environment variables.
- [ ] Executed test event in Lambda Console (`apigateway-aws-proxy` template) returns HTTP 200.
- [ ] Tested public HTTPS endpoint using curl or Postman.
- [ ] CloudWatch Log stream verified under Monitoring -> View CloudWatch logs.

---

## 20. Rollback Procedure
* **EC2 Instance Preservation:** Keep the existing EC2 instance `i-01f086c5a44746100` in the **Running** or **Stopped** state throughout the initial serverless evaluation period.
* **Instant Traffic Rollback:** Because DNS traffic routing is managed at the domain level (Route 53 or your DNS registrar), if any unforeseen issue arises on AWS Lambda, simply point your DNS A-Record or ALB target back to the EC2 Elastic IP address `54.79.80.188`.
* **Zero Downtime Risk:** No existing EC2 compute or EBS storage is terminated during the migration planning phase.

---

## 21. Final Migration Readiness Assessment
* **Readiness Level:** **READY FOR MANUAL SERVERLESS STAGING**
* **Technical Verdict:** The application's core HTTP routing is well suited for AWS Lambda pay-per-request execution. By containerizing or compiling to an ARM64 binary with a serverless adapter, the workload eliminates 24/7 idle EC2 compute costs while achieving automated scaling.

---

## 22. Inspection Limitations
- **Read-Only Scope:** The auditor strictly abstained from modifying files, creating resources, or touching live production configurations.
- **Dynamic Workload Runtime:** If the application relies on nested Docker daemon calls or background compilations, these subroutines must be asynchronously delegated to ECS Fargate or Lambda Container images as outlined in Section 8.
- **Browser CDP:** Chrome DevTools Protocol (CDP) is not listening on http://127.0.0.1:9222. To enable live browser console inspection, launch Chrome with: `chrome.exe --remote-debugging-port=9222` (preserving your existing user profile).

---

## 23. LLM Usage and Token Consumption Summary
```text
Inspection Summary
------------------
Files scanned: 23
Files sent to LLM: 0
AWS resources inspected: 5
Deterministic findings: 6
AI reasoning calls: 0
Cache hits: 0
Total input tokens: 0
Total output tokens: 0
Estimated API cost: $0.0000
```

---
*Report compiled autonomously by DAROGA Migration Auditor.*
