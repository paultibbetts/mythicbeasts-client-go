package mythicbeasts

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// basicAuth encodes basic auth for use in the auth header.
func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// signIn signs in to the auth service and returns the token
// used for future requests.
func (c *Client) signIn() (*AuthResponse, error) {
	if c.Auth.KeyID == "" || c.Auth.Secret == "" {
		return nil, fmt.Errorf("define keyid and secret")
	}

	url := fmt.Sprintf("%s/login", c.AuthURL)
	req, err := http.NewRequest("POST", url, strings.NewReader("grant_type=client_credentials"))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Basic "+basicAuth(c.Auth.KeyID, c.Auth.Secret))
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	res, err := c.do(req)
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth failed: status %d: %s", res.StatusCode, string(body))
	}

	ar := AuthResponse{}
	err = json.Unmarshal(body, &ar)
	if err != nil {
		return nil, err
	}

	return &ar, nil
}
