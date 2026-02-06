package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"time"
)

// Requester provides the shared transport operations used by service clients.
type Requester interface {
	NewRequest(ctx context.Context, method string, baseURL string, endpoint string, reader io.Reader) (*http.Request, error)
	Do(req *http.Request) (*http.Response, error)
	Get(ctx context.Context, baseURL string, endpoint string) (*http.Response, error)
	Delete(ctx context.Context, baseURL string, endpoint string) error
	Body(res *http.Response) ([]byte, error)
	PollProvisioning(ctx context.Context, baseURL string, pollURL string, timeout time.Duration, identifier string, check func(map[string]any, string) (string, bool)) (string, error)
}

// BaseService holds shared transport state for service clients.
type BaseService struct {
	Client  Requester
	BaseURL string
}

// NewBaseService constructs a BaseService for the given client and base URL.
func NewBaseService(client Requester, baseURL string) BaseService {
	return BaseService{Client: client, BaseURL: baseURL}
}

// NewRequest builds a request relative to the service base URL.
func (s BaseService) NewRequest(ctx context.Context, method string, endpoint string, reader io.Reader) (*http.Request, error) {
	return s.Client.NewRequest(ctx, method, s.BaseURL, endpoint, reader)
}

// Do executes an HTTP request.
func (s BaseService) Do(req *http.Request) (*http.Response, error) {
	return s.Client.Do(req)
}

// Get issues a GET request relative to the service base URL.
func (s BaseService) Get(ctx context.Context, endpoint string) (*http.Response, error) {
	return s.Client.Get(ctx, s.BaseURL, endpoint)
}

// Delete issues a DELETE request relative to the service base URL.
func (s BaseService) Delete(ctx context.Context, endpoint string) error {
	return s.Client.Delete(ctx, s.BaseURL, endpoint)
}

// Body reads and closes the response body.
func (s BaseService) Body(res *http.Response) ([]byte, error) {
	return s.Client.Body(res)
}

// PollProvisioning repeatedly polls a provisioning URL relative to the base URL.
func (s BaseService) PollProvisioning(ctx context.Context, pollURL string, timeout time.Duration, identifier string, check func(map[string]any, string) (string, bool)) (string, error) {
	return s.Client.PollProvisioning(ctx, s.BaseURL, pollURL, timeout, identifier, check)
}

// GetJSON issues a GET and unmarshals the JSON response.
// If allowedStatus is provided it is validated before unmarshalling.
func (s BaseService) GetJSON(ctx context.Context, endpoint string, out any, allowedStatus ...int) (*http.Response, []byte, error) {
	res, err := s.Get(ctx, endpoint)
	if err != nil {
		return nil, nil, err
	}

	body, err := s.Body(res)
	if err != nil {
		return res, nil, err
	}

	if len(allowedStatus) > 0 {
		if err := ExpectStatus(res, body, allowedStatus...); err != nil {
			return res, body, err
		}
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return res, body, err
		}
	}

	return res, body, nil
}

// DoJSON issues a request with an optional JSON body and unmarshals the JSON response.
// If allowedStatus is provided it is validated before unmarshalling.
func (s BaseService) DoJSON(ctx context.Context, method string, endpoint string, in any, out any, allowedStatus ...int) (*http.Response, []byte, error) {
	var reader io.Reader
	if in != nil {
		payload, err := json.Marshal(in)
		if err != nil {
			return nil, nil, err
		}
		reader = bytes.NewBuffer(payload)
	}

	req, err := s.NewRequest(ctx, method, endpoint, reader)
	if err != nil {
		return nil, nil, err
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := s.Do(req)
	if err != nil {
		return nil, nil, err
	}

	body, err := s.Body(res)
	if err != nil {
		return res, nil, err
	}

	if len(allowedStatus) > 0 {
		if err := ExpectStatus(res, body, allowedStatus...); err != nil {
			return res, body, err
		}
	}

	if out != nil {
		if err := json.Unmarshal(body, out); err != nil {
			return res, body, err
		}
	}

	return res, body, nil
}

// ExpectStatus returns an error if the response status code is not allowed.
func ExpectStatus(res *http.Response, body []byte, allowedStatus ...int) error {
	if slices.Contains(allowedStatus, res.StatusCode) {
		return nil
	}

	return fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
}
