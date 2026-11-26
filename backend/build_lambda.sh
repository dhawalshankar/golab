#!/usr/bin/env bash
# Cross-compile Go binary for AWS Lambda Amazon Linux 2023 (arm64)
set -e
echo 'Downloading AWS Lambda Go dependencies...'
go get github.com/aws/aws-lambda-go/lambda github.com/aws/aws-lambda-go/events github.com/awslabs/aws-lambda-go-api-proxy/httpadapter 2>/dev/null || true
go mod tidy 2>/dev/null || true
echo 'Building bootstrap binary for AWS Lambda Graviton (arm64)...'
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o bootstrap lambda_main.go
echo 'Build complete: bootstrap ready for deployment.'
