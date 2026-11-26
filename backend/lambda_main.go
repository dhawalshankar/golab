package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

// LambdaAdapter proxies incoming API Gateway v2 HTTP events to the Go HTTP handler
var LambdaAdapter *httpadapter.HandlerAdapter

func init() {
	// Proxy existing HTTP server routes through AWS Lambda adapter
	mux := http.DefaultServeMux
	LambdaAdapter = httpadapter.New(mux)
}

// LambdaHandler is invoked by AWS Lambda on incoming API Gateway requests
func LambdaHandler(ctx context.Context, req events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	return LambdaAdapter.ProxyWithContext(ctx, req)
}

// RunAsLambda starts the Lambda runtime listener when invoked in AWS Serverless environment
func RunAsLambda() {
	log.Println("[LAMBDA] Starting AWS Lambda serverless event loop...")
	lambda.Start(LambdaHandler)
}

func main() {
	log.Println("[LAMBDA] Starting AWS Lambda serverless event loop...")
	lambda.Start(LambdaHandler)
}
