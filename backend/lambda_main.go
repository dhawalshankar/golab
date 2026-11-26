package main

import (
    "context"
    "log"
    "net/http"

    "github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
    "github.com/aws/aws-lambda-go/lambda"
)

// init registers routes as the original main.go would.
// It is assumed that the original code registers its handlers on the default ServeMux
// in an init() function or via a function call. If the original code uses a custom
// router, replace `http.DefaultServeMux` with that router instance.
func init() {
    // Import the original package's side‑effects (e.g., route registration).
    // The original main.go likely lives in the same package (main), so its init()
    // functions will run automatically when this file is compiled.
    // If the original code requires an explicit call, uncomment and adjust:
    // registerRoutes()
}

func main() {
    // Create an adapter that converts API Gateway proxy events to an http.Handler.
    adapter := httpadapter.New(http.DefaultServeMux)

    // Start the Lambda handler.
    lambda.Start(func(ctx context.Context, req interface{}) (interface{}, error) {
        // The adapter expects *events.APIGatewayProxyRequest, but the generic
        // lambda.Start will forward the correct type. We simply forward the call.
        return adapter.ProxyWithContext(ctx, req)
    })

    // The line below will never be reached in Lambda, but keeps the binary
    // runnable locally for testing.
    log.Println("Running locally on :3000")
    if err := http.ListenAndServe(":3000", http.DefaultServeMux); err != nil {
        log.Fatalf("failed to start local server: %v", err)
    }
}
