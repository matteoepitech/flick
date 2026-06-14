/*
** FLICK PROJECT, 2026
** flick/internal/cli/network/httpclient
** File description:
** Shared HTTP client trying QUIC (HTTP/3) first, falling back to HTTP/2.
 */

package network

import (
	"net/http"
	"sync/atomic"

	"github.com/quic-go/quic-go/http3"
)

// Structure to transport a HTTP request. Start with HTTP/3 and if fails
// continue with HTTP/2.
// forceH2 is used to prevent the re-check of HTTP/3 if already failed.
type fallbackTransport struct {
	h3      *http3.Transport
	h2      *http.Transport
	forceH2 atomic.Bool
}

// canRetry: Tell whether a request can safely be replayed on the fallback transport after an HTTP/3 failure.
//
// Params:
// - req (*http.Request): The request to check.
//
// Returns:
// - result1 (bool): True if the request body can be replayed.
func canRetry(req *http.Request) bool {
	return req.Body == nil || req.Body == http.NoBody || req.GetBody != nil
}

// RoundTrip: Try HTTP/3, then fall back to HTTP/2 when QUIC is unavailable.
//
// Params:
// - req (*http.Request): The request to perform.
//
// Returns:
// - result1 (*http.Response): The response.
// - result2 (error): An error if both transports failed.
func (t *fallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.forceH2.Load() {
		return t.h2.RoundTrip(req)
	}

	resp, err := t.h3.RoundTrip(req)
	if err == nil {
		return resp, nil
	}

	t.forceH2.Store(true)
	if canRetry(req) == false {
		return nil, err
	}

	if req.GetBody != nil {
		body, err := req.GetBody()
		if err != nil {
			return nil, err
		}
		req.Body = body
	}
	return t.h2.RoundTrip(req)
}

// SharedClient is the HTTP client used by every CLI call.
var SharedClient = &http.Client{
	Transport: &fallbackTransport{
		h3: &http3.Transport{},
		h2: http.DefaultTransport.(*http.Transport).Clone(),
	},
}
