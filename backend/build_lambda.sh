#!/usr/bin/env bash
set -euo pipefail

# Build the Go binary for Linux (Amazon Linux 2) suitable for Lambda
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main ./backend/lambda_main.go

# Create a deployment package
zip -j lambda.zip main

# Clean up the binary after zipping (optional)
rm -f main

echo "Lambda deployment package created: lambda.zip"
