package provider

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/wunderpus/wunderpus/internal/errors"
)

// DefaultClient is the shared HTTP client for all providers.
var DefaultClient = &http.Client{
	Timeout: 60 * time.Second,
	Transport: &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		ForceAttemptHTTP2: true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	},
}

// RetryOptions configures the retry logic.
type RetryOptions struct {
	MaxRetries int
	Initial    time.Duration
	MaxDelay   time.Duration
}

var DefaultRetryOptions = RetryOptions{
	MaxRetries: 3,
	Initial:    1 * time.Second,
	MaxDelay:   10 * time.Second,
}

// RetryDo executes an HTTP request with exponential backoff.
func RetryDo(ctx context.Context, client *http.Client, req *http.Request, opts RetryOptions) (*http.Response, error) {
	var lastErr error
	var bodyBytes []byte

	// If there's a body, we need to buffer it to replay it
	if req.Body != nil {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, errors.Wrap(errors.ProviderError, "reading request body for retry", err)
		}
		_ = req.Body.Close()
	}

	for i := 0; i <= opts.MaxRetries; i++ {
		// Prepare the request for this attempt
		if bodyBytes != nil {
			req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		}

		resp, err := client.Do(req.WithContext(ctx))
		if err != nil {
			lastErr = err
			if !isRetryable(err) {
				return nil, errors.Wrap(errors.ProviderError, "request failed (non-retryable)", err)
			}
		} else {
			// Check status codes
			if resp.StatusCode == http.StatusOK {
				return resp, nil
			}

			// Read body to include in error
			respBody, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastErr = fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))

			if !isStatusRetryable(resp.StatusCode) {
				return nil, errors.Wrap(errors.ProviderError, "API returned non-retryable error", lastErr)
			}
		}

		if i < opts.MaxRetries {
			delay := time.Duration(math.Pow(2, float64(i))) * opts.Initial
			if delay > opts.MaxDelay {
				delay = opts.MaxDelay
			}

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return nil, ctx.Err()
			}
		}
	}

	return nil, errors.Wrap(errors.ProviderError, fmt.Sprintf("failed after %d retries", opts.MaxRetries), lastErr)
}

func isRetryable(err error) bool {
	if errors.IsRetryable(err) {
		return true
	}
	if netErr, ok := err.(net.Error); ok {
		return netErr.Timeout() || netErr.Temporary()
	}
	// Many connection reset errors aren't net.Error but are transient
	msg := err.Error()
	return strings.Contains(msg, "connection reset") || strings.Contains(msg, "broken pipe")
}

func isStatusRetryable(code int) bool {
	return code == http.StatusTooManyRequests || code >= 500
}

// MaxResponseSize is the default limit for API responses (10MB).
const MaxResponseSize = 10 * 1024 * 1024

// LimitResponseReader wraps a response body with a size limit.
func LimitResponseReader(body io.ReadCloser) io.ReadCloser {
	return &limitedReadCloser{
		Reader: io.LimitReader(body, MaxResponseSize),
		Closer: body,
	}
}

type limitedReadCloser struct {
	io.Reader
	io.Closer
}
