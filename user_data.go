package mythicbeasts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// NewUserData represents the data required to create
// a new User Data snippet.
type NewUserData struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// UserData represents a User Data snippet.
type UserData struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Data string `json:"content"`
	Size int64  `json:"size"`
}

// CreateUserData creates a new User Data snippet with the given ID.
func (c *Client) CreateUserData(data NewUserData) (*UserData, error) {
	requestURL := fmt.Sprintf("vps/user-data")

	requestJson, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(http.MethodPost, requestURL, bytes.NewBuffer(requestJson))
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := c.do(req)
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	if res.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	var created UserData
	err = json.Unmarshal(body, &created)
	if err != nil {
		return nil, err
	}

	return &created, nil
}

// GetUserData retrieves the User Data snippet with the given ID.
func (c *Client) GetUserData(id int64) (*UserData, error) {
	requestUrl := fmt.Sprintf("/vps/user-data/%d", id)

	res, err := c.get(requestUrl)
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	var result UserData
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// DeleteUserData removes the User Data snippet with the given ID.
func (c *Client) DeleteUserData(id int64) error {
	url := fmt.Sprintf("/vps/user-data/%d", id)

	return c.delete(url)
}
