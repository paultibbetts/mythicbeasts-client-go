package mythicbeasts

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/paultibbetts/mythicbeasts-client-go/pi"
	"github.com/paultibbetts/mythicbeasts-client-go/proxy"
	"github.com/paultibbetts/mythicbeasts-client-go/vps"
)

// AuthURL is the URL of the auth service to sign in.
const AuthURL string = "https://auth.mythic-beasts.com"

// DefaultUserAgent is the default user agent to send with requests.
const DefaultUserAgent string = "mythicbeasts-client-go"

// Client uses http client to wrap communication.
type Client struct {
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
	// UserAgent is the User-Agent header used for requests.
	UserAgent string

	authMu          sync.RWMutex
	tokenExpiresIn  time.Duration
	tokenLastUsedAt time.Time

	piService    *pi.Service
	vpsService   *vps.Service
	proxyService *proxy.Service
}

// AuthStruct contains the API key credentials used to request a token.
type AuthStruct struct {
	KeyID  string `json:"keyid"`
	Secret string `json:"secret"`
}

// AuthResponse represents the response from the authentication service.
type AuthResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
	TokenType   string `json:"token_type"`
}

// NewClient constructs a client with sensible defaults.
// Credentials are required for most API calls; if provided, they are stored
// and a token is fetched on the first authenticated request.
// If they are empty it will return an unauthenticated client.
// The returned client does not follow redirects.
func NewClient(keyid, secret string) (*Client, error) {
	hc := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	c := Client{
		HTTPClient:   hc,
		AuthURL:      AuthURL,
		PollInterval: 10 * time.Second,
		UserAgent:    DefaultUserAgent,
	}

	if keyid == "" || secret == "" {
		return &c, nil
	}

	c.Auth = AuthStruct{
		KeyID:  keyid,
		Secret: secret,
	}

	return &c, nil
}

// Do sends the request with the configured client,
// injecting the token if it is present.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req.Header.Get("Authorization") == "" {
		token, err := c.ensureToken(req.Context())
		if err != nil {
			return nil, err
		}
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
			c.markTokenUsed()
		}
	}
	if c.UserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}

	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// ensureToken ensures the client has a valid token.
//
// The auth service returns expires_in once at sign-in, but the token
// expiry is based on time since last use (sliding TTL).
// The client tracks tokenLastUsedAt per request and refreshes when near expiry.
// See [making API requests] for details.
//
// [making API requests]: https://www.mythic-beasts.com/support/api/auth#sec-making-api-requests
func (c *Client) ensureToken(ctx context.Context) (string, error) {
	c.authMu.RLock()
	token := c.Token
	expiresIn := c.tokenExpiresIn
	lastUsedAt := c.tokenLastUsedAt
	hasCreds := c.Auth.KeyID != "" && c.Auth.Secret != ""
	c.authMu.RUnlock()

	if token != "" {
		if !hasCreds {
			return token, nil
		}
		if !tokenExpired(expiresIn, lastUsedAt) {
			return token, nil
		}
	}
	if !hasCreds {
		return "", nil
	}

	c.authMu.Lock()
	defer c.authMu.Unlock()

	if c.Token != "" {
		if c.Auth.KeyID == "" || c.Auth.Secret == "" {
			return c.Token, nil
		}
		if !tokenExpired(c.tokenExpiresIn, c.tokenLastUsedAt) {
			return c.Token, nil
		}
	}
	if c.Auth.KeyID == "" || c.Auth.Secret == "" {
		return "", nil
	}

	authResponse, err := c.signIn(ctx)
	if err != nil {
		return "", err
	}
	c.Token = authResponse.AccessToken
	c.tokenExpiresIn = time.Duration(authResponse.ExpiresIn) * time.Second
	c.tokenLastUsedAt = time.Time{}

	return c.Token, nil
}

func (c *Client) markTokenUsed() {
	c.authMu.Lock()
	defer c.authMu.Unlock()
	c.tokenLastUsedAt = time.Now()
}

func tokenExpired(expiresIn time.Duration, lastUsedAt time.Time) bool {
	if expiresIn <= 0 || lastUsedAt.IsZero() {
		return false
	}
	expiry := expiresIn - 10*time.Second
	if expiry < 0 {
		expiry = 0
	}
	return time.Since(lastUsedAt) >= expiry
}

// NewRequest builds an *http.Request for the given endpoint.
// If the endpoint is absolute it is used as-is; otherwise
// it is resolved relative to the baseURL.
// Returns an error if the baseURL is invalid.
func (c *Client) NewRequest(ctx context.Context, method string, baseURL string, endpoint string, reader io.Reader) (*http.Request, error) {
	parsedURL, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if parsedURL.IsAbs() {
		return http.NewRequestWithContext(ctx, method, parsedURL.String(), reader)
	}

	base, err := url.Parse(baseURL)
	if err != nil || base.Scheme == "" || base.Host == "" {
		return nil, fmt.Errorf("invalid base url: %q", baseURL)
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

	return http.NewRequestWithContext(ctx, method, full.String(), reader)
}

// DoRequest is a convenience wrapper around NewRequest + Do.
func (c *Client) DoRequest(ctx context.Context, method, baseURL, endpoint string, reader io.Reader) (*http.Response, error) {
	req, err := c.NewRequest(ctx, method, baseURL, endpoint, reader)
	if err != nil {
		return nil, err
	}

	return c.Do(req)
}

// Get issues a GET request to the endpoint, relative to the baseURL.
func (c *Client) Get(ctx context.Context, baseURL, endpoint string) (*http.Response, error) {
	return c.DoRequest(ctx, http.MethodGet, baseURL, endpoint, nil)
}

// Delete issues a DELETE request to the endpoint, relative to the baseURL.
// It accepts a 404 as a successful deletion.
func (c *Client) Delete(ctx context.Context, baseURL, endpoint string) error {
	res, err := c.DoRequest(ctx, http.MethodDelete, baseURL, endpoint, nil)
	if err != nil {
		return err
	}

	body, err := c.Body(res)
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

// Body reads and closes the body of a response.
// It **must** be used after a GET request to close the body.
func (c *Client) Body(res *http.Response) ([]byte, error) {
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

// PollProvisioning repeatedly polls the pollURL until completion, error
// or timeout. It uses a check function to determine completion.
// On success it returns the final resource URL.
func (c *Client) PollProvisioning(ctx context.Context, baseURL, pollURL string, timeout time.Duration, identifier string, check func(map[string]any, string) (string, bool)) (serverURL string, error error) {
	deadline := time.Now().Add(timeout)

	req, err := c.NewRequest(ctx, "GET", baseURL, pollURL, nil)
	if err != nil {
		return "", err
	}

	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if time.Now().After(deadline) {
			return "", errors.New("timed out while provisioning")
		}

		res, err := c.Do(req)
		if err != nil {
			return "", err
		}
		body, err := c.Body(res)
		if err != nil {
			return "", err
		}

		location := res.Header.Get("Location")

		switch res.StatusCode {
		case http.StatusSeeOther:
			if location == "" {
				return "", errors.New("polling returned no location")
			}
			return location, nil
		case http.StatusInternalServerError:
			return "", fmt.Errorf("provisioning failed: %s", string(body))
		case http.StatusAccepted:
			if location != "" {
				return location, nil
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(c.PollInterval):
				continue
			}
		case http.StatusOK:
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

			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(c.PollInterval):
				continue
			}
		default:
			return "", fmt.Errorf("unexpected status while polling: %d", res.StatusCode)
		}

	}
}
