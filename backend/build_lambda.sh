#!/usr/bin/env bash
set -euo pipefail

# Build the Go binary for Linux (compatible with AWS Lambda)
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o main ./backend

# Create a deployment package (zip) for the Lambda function
zip -j lambda_package.zip main

# Clean up the binary after zipping (optional)
rm -f main
