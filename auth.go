package mythicbeasts

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func (c *Client) SignIn() (*AuthResponse, error) {
	if c.Auth.KeyID == "" || c.Auth.Secret == "" {
		return nil, fmt.Errorf("define keyid and secret")
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/login", c.AuthURL), strings.NewReader("grant_type=client_credentials"))
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
