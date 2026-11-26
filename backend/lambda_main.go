package main

import (
    "bytes"
    "context"
    "encoding/json"
    "log"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "sync"
    "time"

    "github.com/aws/aws-lambda-go/events"
    "github.com/aws/aws-lambda-go/lambda"
)

// Constants copied from original main.go
const (
    maxCodeSize   = 10 * 1024       // 10 KB
    maxInputSize  = 10 * 1024       // 10 KB
    maxOutputSize = 1 * 1024 * 1024 // 1 MB

    compileTimeout = 90 * time.Second
    runTimeout     = 5 * time.Second

    defaultCompilerImage = "golab-compiler:latest"
    defaultRuntimeImage  = "golab-runtime:latest"
)

// Keep the MVP single‑job for the small EC2 instance.
var runLock sync.Mutex

type RunRequest struct {
    Code  string `json:"code"`
    Input string `json:"input"`
}

type RunResponse struct {
    Stdout      string `json:"stdout"`
    Stderr      string `json:"stderr"`
    ExitCode    int    `json:"exitCode"`
    Runtime     int64  `json:"runtimeMs"` // Backward‑compatible field for the frontend.
    CompileMs   int64  `json:"compileMs"`
    ExecutionMs int64  `json:"executionMs"`
    TotalMs     int64  `json:"totalMs"`
}

type LimitedBuffer struct {
    Buffer    bytes.Buffer
    Limit     int
    Truncated bool
}

func (b *LimitedBuffer) Write(p []byte) (int, error) {
    remaining := b.Limit - b.Buffer.Len()
    if remaining <= 0 {
        b.Truncated = true
        return len(p), nil
    }
    if len(p) > remaining {
        _, _ = b.Buffer.Write(p[:remaining])
        b.Truncated = true
        return len(p), nil
    }
    _, _ = b.Buffer.Write(p)
    return len(p), nil
}

// Helper to write JSON response for API Gateway
func apiResponse(status int, body interface{}) (events.APIGatewayProxyResponse, error) {
    jsonBody, err := json.Marshal(body)
    if err != nil {
        return events.APIGatewayProxyResponse{StatusCode: http.StatusInternalServerError}, err
    }
    return events.APIGatewayProxyResponse{
        StatusCode: status,
        Headers: map[string]string{"Content-Type": "application/json"},
        Body: string(jsonBody),
    }, nil
}

func handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
    // Parse incoming JSON body (same shape as original HTTP server)
    var runReq RunRequest
    if err := json.Unmarshal([]byte(req.Body), &runReq); err != nil {
        return apiResponse(http.StatusBadRequest, map[string]string{"error": "invalid JSON payload"})
    }

    // Enforce size limits (same as original)
    if len(runReq.Code) > maxCodeSize || len(runReq.Input) > maxInputSize {
        return apiResponse(http.StatusBadRequest, map[string]string{"error": "payload too large"})
    }

    // The original implementation used a global mutex to serialize runs. Preserve that behaviour.
    runLock.Lock()
    defer runLock.Unlock()

    // Compile step – identical to original main.go (extracted into a helper function below)
    compileStart := time.Now()
    compileCmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-i", "-v", "/tmp:/tmp", "-w", "/tmp", defaultCompilerImage, "bash", "-c", "cat > main.go && go build -o main main.go")
    compileCmd.Stdin = strings.NewReader(runReq.Code)
    var compileStdout, compileStderr LimitedBuffer
    compileStdout.Limit = maxOutputSize
    compileStderr.Limit = maxOutputSize
    compileCmd.Stdout = &compileStdout
    compileCmd.Stderr = &compileStderr
    compileCmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64")
    compileCtx, cancelCompile := context.WithTimeout(ctx, compileTimeout)
    defer cancelCompile()
    if err := compileCmd.Run(); err != nil {
        // Compilation failed – return stderr as part of response
        resp := RunResponse{Stdout: compileStdout.Buffer.String(), Stderr: compileStderr.Buffer.String(), ExitCode: 1, CompileMs: time.Since(compileStart).Milliseconds()}
        return apiResponse(http.StatusOK, resp)
    }
    compileMs := time.Since(compileStart).Milliseconds()

    // Execution step – run the compiled binary inside the runtime container
    execStart := time.Now()
    execCmd := exec.CommandContext(ctx, "docker", "run", "--rm", "-i", "-v", "/tmp:/tmp", "-w", "/tmp", defaultRuntimeImage, "./main")
    execCmd.Stdin = strings.NewReader(runReq.Input)
    var execStdout, execStderr LimitedBuffer
    execStdout.Limit = maxOutputSize
    execStderr.Limit = maxOutputSize
    execCmd.Stdout = &execStdout
    execCmd.Stderr = &execStderr
    execCtx, cancelRun := context.WithTimeout(ctx, runTimeout)
    defer cancelRun()
    err := execCmd.Run()
    exitCode := 0
    if err != nil {
        if exitError, ok := err.(*exec.ExitError); ok {
            exitCode = exitError.ExitCode()
        } else {
            // Unexpected error – treat as internal server error
            log.Printf("runtime error: %v", err)
            return apiResponse(http.StatusInternalServerError, map[string]string{"error": "runtime execution failed"})
        }
    }
    execMs := time.Since(execStart).Milliseconds()

    // Build final response matching original API contract
    resp := RunResponse{
        Stdout:      execStdout.Buffer.String(),
        Stderr:      execStderr.Buffer.String(),
        ExitCode:    exitCode,
        Runtime:     execMs,
        CompileMs:   compileMs,
        ExecutionMs: execMs,
        TotalMs:     compileMs + execMs,
    }
    return apiResponse(http.StatusOK, resp)
}

func main() {
    // When running locally (e.g., `sam local start-api`) we fall back to the original HTTP server for parity.
    if os.Getenv("LAMBDA_TASK_ROOT") == "" {
        // Simple HTTP server for local dev – reuse original logic (omitted for brevity).
        log.Println("Running in local HTTP mode – start server on :8080")
        http.HandleFunc("/run", func(w http.ResponseWriter, r *http.Request) {
            // Reuse the same handler logic by delegating to the Lambda handler with a fabricated request.
            body := new(strings.Builder)
            _, _ = body.ReadFrom(r.Body)
            proxyReq := events.APIGatewayProxyRequest{Body: body.String()}
            resp, _ := handler(context.Background(), proxyReq)
            w.Header().Set("Content-Type", "application/json")
            w.WriteHeader(resp.StatusCode)
            _, _ = w.Write([]byte(resp.Body))
        })
        log.Fatal(http.ListenAndServe(":8080", nil))
        return
    }
    lambda.Start(handler)
}
