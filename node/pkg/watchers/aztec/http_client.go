package aztec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"go.uber.org/zap"
)

// HTTPClient provides a wrapper for HTTP requests with retries and timeouts
type HTTPClient interface {
	DoRequest(ctx context.Context, url string, payload map[string]any) ([]byte, error)
}

// retryableHTTPClient is the implementation of HTTPClient using retryablehttp
type retryableHTTPClient struct {
	client *http.Client
	logger *zap.Logger
}

// NewHTTPClient creates a new HTTP client with built-in retry functionality
func NewHTTPClient(timeout time.Duration, maxRetries int, initialBackoff time.Duration, backoffMultiplier float64, logger *zap.Logger) HTTPClient {
	// Create a retryable HTTP client
	retryClient := retryablehttp.NewClient()

	// Configure the retry settings
	retryClient.RetryMax = maxRetries
	retryClient.RetryWaitMin = initialBackoff
	retryClient.RetryWaitMax = time.Duration(float64(initialBackoff) * backoffMultiplier * float64(maxRetries))
	retryClient.HTTPClient.Timeout = timeout

	// Configure the logger
	// Use a custom logger that wraps our zap logger
	retryClient.Logger = newRetryableHTTPZapLogger(logger)

	// Get the standard *http.Client from the retryable client
	standardClient := retryClient.StandardClient()

	return &retryableHTTPClient{
		client: standardClient,
		logger: logger,
	}
}

// DoRequest sends an HTTP request with retries provided by retryablehttp
func (c *retryableHTTPClient) DoRequest(ctx context.Context, url string, payload map[string]any) ([]byte, error) {
	// Marshal the payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshaling JSON: %v", err)
	}

	// Create the request
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	// Execute the request with automatic retries
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request error: %v", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Warn("Non-200 status code",
			zap.String("url", url),
			zap.Int("status", resp.StatusCode),
			zap.String("response", string(body)))
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Check for JSON-RPC errors in the response
	hasError, rpcError := GetJSONRPCError(body)
	if hasError {
		return nil, rpcError
	}

	return body, nil
}

// Adapter to make zap logger work with retryablehttp's logger interface
type retryableHTTPZapLogger struct {
	logger *zap.Logger
}

func newRetryableHTTPZapLogger(logger *zap.Logger) *retryableHTTPZapLogger {
	return &retryableHTTPZapLogger{logger: logger}
}

func (l *retryableHTTPZapLogger) Error(msg string, _ ...interface{}) {
	l.logger.Error(msg)
}

func (l *retryableHTTPZapLogger) Info(msg string, _ ...interface{}) {
	l.logger.Info(msg)
}

func (l *retryableHTTPZapLogger) Debug(msg string, _ ...interface{}) {
	l.logger.Debug(msg)
}

func (l *retryableHTTPZapLogger) Warn(msg string, _ ...interface{}) {
	l.logger.Warn(msg)
}
