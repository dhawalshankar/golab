# AWS Lambda Migration Deployment Guide

This guide walks you through deploying the migrated backend service to AWS using the Serverless Application Model (SAM).

## Prerequisites
- AWS CLI configured with sufficient permissions.
- AWS SAM CLI installed (v1.100+).
- Docker installed (for building the Lambda container if you later need a custom runtime).
- An existing VPC/subnets/security groups if you plan to run the decoupled ECS Fargate task.

## 1. Build the Lambda Package
```bash
cd backend
chmod +x build_lambda.sh
./build_lambda.sh
```
The script produces `lambda_package.zip` containing the compiled Go binary.

## 2. Package the SAM Application
```bash
sam package \
  --template-file template.yaml \
  --s3-bucket <YOUR_S3_BUCKET_FOR_DEPLOYMENTS> \
  --output-template-file packaged.yaml
```
Replace `<YOUR_S3_BUCKET_FOR_DEPLOYMENTS>` with an S3 bucket you control.

## 3. Deploy the Stack
```bash
sam deploy \
  --template-file packaged.yaml \
  --stack-name backend-serverless-stack \
  --capabilities CAPABILITY_IAM CAPABILITY_AUTO_EXPAND \
  --region <YOUR_AWS_REGION>
```
The deployment creates:
- **BackendFunction** – Lambda handling all HTTP routes via API Gateway.
- **DockerTaskDefinition** – ECS Fargate task definition that you can invoke from the Lambda (e.g., via AWS SDK) to run any required Docker workloads.
- **LogGroup** – CloudWatch Logs for the Fargate container.

## 4. Configure Environment Variables (Optional)
If your application needs configuration (e.g., DB connection strings, secrets), add them to the Lambda function after deployment:
```bash
aws lambda update-function-configuration \
  --function-name $(aws cloudformation describe-stack-resource \
    --stack-name backend-serverless-stack \
    --logical-resource-id BackendFunction \
    --query 'StackResourceDetail.PhysicalResourceId' --output text) \
  --environment Variables={KEY1=VALUE1,KEY2=VALUE2}
```
Similarly, you can attach IAM policies to the `ECSTaskExecutionRole` if the Docker task needs extra permissions.

## 5. Test the API
The SAM deployment creates an API Gateway endpoint. Retrieve it:
```bash
aws cloudformation describe-stacks \
  --stack-name backend-serverless-stack \
  --query "Stacks[0].Outputs[?OutputKey=='BackendFunctionArn'].OutputValue" --output text
```
Use the URL (e.g., `https://xxxx.execute-api.<region>.amazonaws.com/Prod/`) to send HTTP requests to your service.

## 6. Invoking the Decoupled Docker Worker
From within your Lambda code, you can start the ECS task using the AWS SDK:
```go
svc := ecs.New(session.Must(session.NewSession()))
_, err := svc.RunTask(&ecs.RunTaskInput{
    Cluster:        aws.String("default"), // or your cluster name
    TaskDefinition: aws.String("<DockerTaskDefinitionArn>"),
    LaunchType:     aws.String("FARGATE"),
    NetworkConfiguration: &ecs.NetworkConfiguration{...},
})
```
Make sure the Lambda execution role has `ecs:RunTask` permission.

## 7. Clean‑up
To delete all resources:
```bash
sam delete --stack-name backend-serverless-stack
```

---
**Note:** The original service exposed ports 3000, 10001, and 10485. In the serverless model, these ports are no longer needed; API Gateway routes all HTTP traffic to the Lambda function. Any internal Docker workloads that required those ports should be handled by the decoupled ECS Fargate task.
