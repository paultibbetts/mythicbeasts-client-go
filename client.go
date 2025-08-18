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

const HostURL string = "https://api.mythic-beasts.com/beta"
const AuthURL string = "https://auth.mythic-beasts.com"

type Client struct {
	HostURL      string
	AuthURL      string
	HTTPClient   *http.Client
	Token        string
	Auth         AuthStruct
	PollInterval time.Duration
}

type AuthStruct struct {
	KeyID  string `json:"keyid"`
	Secret string `json:"secret"`
}

type AuthResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

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

	authResponse, err := c.SignIn()
	if err != nil {
		return nil, err
	}

	c.Token = authResponse.AccessToken

	return &c, nil
}

type ErrIdentifierConflict struct {
	Identifier string
}

func (e *ErrIdentifierConflict) Error() string {
	return fmt.Sprintf("identifier %q already in use", e.Identifier)
}

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

func (c *Client) doRequest(method, endpoint string, reader io.Reader) (*http.Response, error) {
	req, err := c.NewRequest(method, endpoint, reader)
	if err != nil {
		return nil, err
	}

	return c.do(req)
}

func (c *Client) get(endpoint string) (*http.Response, error) {
	return c.doRequest(http.MethodGet, endpoint, nil)
}

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

func truncateBody(b []byte) string {
	const max = 512
	if len(b) <= max {
		return string(b)
	}
	return string(b[:max]) + "..."
}

func (c *Client) body(res *http.Response) ([]byte, error) {
	defer res.Body.Close()
	return io.ReadAll(res.Body)
}

type CompletionChecker func(data map[string]any, identifier string) (string, bool)

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
