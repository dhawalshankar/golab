package main

import (
    "net/http"
    "github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
    "github.com/aws/aws-lambda-go/lambda"
)

// Assuming the original application registers its routes on the default ServeMux
// (e.g., via init functions or in main()). If the app uses a custom router,
// replace http.DefaultServeMux with that router instance.

func main() {
    // Create an adapter that converts API Gateway proxy events to http.Request
    // and forwards them to the standard net/http handler.
    adapter := httpadapter.New(http.DefaultServeMux)
    // Start the Lambda runtime.
    lambda.Start(adapter.ProxyWithContext)
}
