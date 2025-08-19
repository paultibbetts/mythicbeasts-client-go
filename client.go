package mythicbeasts

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HostURL is the default base URL to use for requests.
const HostURL string = "https://api.mythic-beasts.com/beta"

// AuthURL is the URL of the auth service to sign in.
const AuthURL string = "https://auth.mythic-beasts.com"

// Client uses http client to wrap communication.
type Client struct {
	// HostURL is the base endpoint for API calls.
	HostURL string
	// AuthURL is the endpoint to request tokens to sign in.
	AuthURL string
	// HTTPClient is the HTTP transport.
	HTTPClient *http.Client
	// Token is the bearer token used for requests.
	Token string
	// Auth holds the API credentials used to obtain a token.
	Auth AuthStruct
	// PollInterval controls the wait between provisioning poll attempts.
	PollInterval time.Duration
}

// AuthStruct contains the API key credentials used to request a token.
type AuthStruct struct {
	KeyID  string `json:"keyid"`
	Secret string `json:"secret"`
}

// AuthResponse represents the response from the authentication service.
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

// NewClient constructs a client with sensible defaults.
// If Key ID and secret are provided it performs an auth flow.
// If they are empty it will return an unauthenticated client.
// The returned client does not follow redirects.
func NewClient(keyid, secret *string) (*Client, error) {
	hc := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	c := Client{
		HTTPClient:   hc,
		HostURL:      HostURL,
		AuthURL:      AuthURL,
		PollInterval: 10 * time.Second,
	}

	if keyid == nil || secret == nil {
		return &c, nil
	}

	c.Auth = AuthStruct{
		KeyID:  *keyid,
		Secret: *secret,
	}

	authResponse, err := c.signIn()
	if err != nil {
		return nil, err
	}

	c.Token = authResponse.AccessToken

	return &c, nil
}

// ErrIdentifierConflict indicates the requested resource identifier
// has alreasdy been used.
type ErrIdentifierConflict struct {
	Identifier string
}

func (e *ErrIdentifierConflict) Error() string {
	return fmt.Sprintf("identifier %q already in use", e.Identifier)
}

// do sends the request with the configured client,
// injecting the token if it is present.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	token := c.Token

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// NewRequst builds an *http.Request for the given endpoint.
// If the endpoint is absolute it is used as-is; otherwise
// it is resolved relative to the c.HostURL.
// Returns an error if the c.HostURL is invalid.
func (c *Client) NewRequest(method string, endpoint string, reader io.Reader) (*http.Request, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if parsedURL.IsAbs() {
		return http.NewRequest(method, parsedURL.String(), reader)
	}

	base, err := url.Parse(c.HostURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("invalid host url: %q", c.HostURL)
	}

	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}

	rel := &url.URL{
		Path:     strings.TrimPrefix(parsedURL.Path, "/"),
		RawQuery: parsedURL.RawQuery,
		Fragment: parsedURL.Fragment,
	}

	full := base.ResolveReference(rel)

	return http.NewRequest(method, full.String(), reader)
}

// doRequest is a conveniance wrapper around NewRequest + do.
func (c *Client) doRequest(method, endpoint string, reader io.Reader) (*http.Response, error) {
	req, err := c.NewRequest(method, endpoint, reader)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

// get issues a GET request to the endpoint, relative to the c.HostURL.
func (c *Client) get(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodGet, endpoint, nil)
}

// delete issues a DELETE request to the endpoint, relative to the c.HostURL.
// It accepts a 404 as a successful deletion.
func (c *Client) delete(endpoint string) error {
	res, err := c.doRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	body, err := c.body(res)
	if err != nil {
		return err
	}

	switch res.StatusCode {
	case http.StatusNoContent, http.StatusOK, http.StatusAccepted, http.StatusNotFound:
		return nil
	default:
		return fmt.Errorf("unexpected status %d: %s", res.StatusCode, truncateBody(body))
	}
}

// truncateBody is a helper function to truncate the body of a response.
func truncateBody(b []byte) string {
	const max = 512
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

// body reads and closes the body of a response.
// It **must** be used after a GET request to close the body.
func (c *Client) body(res *http.Response) ([]byte, error) {
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// CompletionChecker represents the function used to check if provisioning
// is complete.
type CompletionChecker func(data map[string]any, identifier string) (string, bool)

// pollProvisioning repeatedly polls the pollURL until completion, error
// or timeout. It uses a check function to determine completion.
// On success it returns the final resource URL.
func (c *Client) pollProvisioning(pollUrl string, timeout time.Duration, identifier string, check CompletionChecker) (serverUrl string, error error) {
	deadline := time.Now().Add(timeout)

	req, err := c.NewRequest("GET", pollUrl, nil)
	if err != nil {
		return "", err
	}

	for {
		if time.Now().After(deadline) {
			return "", errors.New("timed out while provisioning")
		}

		res, err := c.do(req)
		if err != nil {
			return "", error
		}
		body, err := c.body(res)
		if err != nil {
			return "", err
		}

		location := res.Header.Get("Location")

		switch res.StatusCode {
		case http.StatusSeeOther: // 303
			if location == "" {
				return "", errors.New("polling returned no location")
			}
			return location, nil
		case http.StatusInternalServerError: // 500
			return "", fmt.Errorf("provisioning failed: %s", string(body))
		case http.StatusAccepted: // 202
			if location != "" {
				return location, nil
			}
			time.Sleep(c.PollInterval)
			continue
		case http.StatusOK: // 200
			if location != "" {
				return location, nil
			}

			var data map[string]any
			err = json.Unmarshal(body, &data)
			if err != nil {
				return "", fmt.Errorf("could not umnarshal ok json: %w", err)
			}

			if url, done := check(data, identifier); done {
				return url, nil
			}

			// nope
			time.Sleep(c.PollInterval)
			continue
		default:
			return "", fmt.Errorf("unexpected status while polling: %d", res.StatusCode)
		}

	}
}
