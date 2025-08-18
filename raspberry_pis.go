package mythicbeasts

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type PiModel struct {
	Model    int64 `json:"model"`
	Memory   int64 `json:"memory"`
	NICSpeed int64 `json:"nic_speed"`
	CPUSpeed int64 `json:"cpu_speed"`
}

func (c *Client) GetPiModels() ([]PiModel, error) {
	res, err := c.get("/pi/models")
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d, %s", res.StatusCode, string(body))
	}

	var result struct {
		Models []PiModel `json:"models"`
	}

	if err = json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Models, nil
}

type PiOperatingSystem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PiOperatingSystems map[string]string

func (c *Client) GetPiOperatingSystems(model int64) (map[string]string, error) {
	url := fmt.Sprintf("/pi/images/%d", model)

	res, err := c.get(url)
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	var result PiOperatingSystems

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

type Pi struct {
	IP              string `json:"ip"`
	SSHPort         int64  `json:"ssh_port"`
	DiskSize        string `json:"disk_size"`
	InitializedKeys bool   `json:"initialized_keys"`
	Location        string `json:"location"`
	Model           int64  `json:"model"`
	Memory          int64  `json:"memory"`
	CPUSpeed        int64  `json:"cpu_speed"`
	NICSpeed        int64  `json:"nic_speed"`
}

type PiServers struct {
	Servers []Pi `json:"servers"`
}

// TODO get rid of this?
// when will I use this?
func (c *Client) GetPis() ([]Pi, error) {
	res, err := c.get("/pi/servers")
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	var result PiServers
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result.Servers, nil
}

func (c *Client) GetPi(identifier string) (Pi, error) {
	if strings.TrimSpace(identifier) == "" {
		return Pi{}, ErrEmptyIdentifier
	}
	url := fmt.Sprintf("/pi/servers/%s", identifier)

	res, err := c.get(url)
	if err != nil {
		return Pi{}, err
	}

	body, err := c.body(res)
	if err != nil {
		return Pi{}, err
	}

	var result Pi
	err = json.Unmarshal(body, &result)
	if err != nil {
		return Pi{}, err
	}

	return result, nil
}

type CreatePiRequest struct {
	Model      int64  `json:"model,omitempty"`
	Memory     int64  `json:"memory,omitempty"`
	CPUSpeed   int64  `json:"cpu_speed,omitempty"`
	DiskSize   int64  `json:"disk,omitempty"` // intentionally different
	SSHKey     string `json:"ssh_key,omitempty"`
	OSImage    string `json:"os_image,omitempty"`
	WaitForDNS bool   `json:"wait_for_dns,omitempty"`
}

func (c *Client) CreatePi(identifier string, server CreatePiRequest) (*Pi, error) {
	requestUrl := fmt.Sprintf("/pi/servers/%s", identifier)

	requestJson, err := json.Marshal(server)
	if err != nil {
		return nil, err
	}

	req, err := c.NewRequest(http.MethodPost, requestUrl, bytes.NewBuffer(requestJson))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := c.body(res)
	if err != nil {
		return nil, fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	if res.StatusCode == http.StatusConflict {
		return nil, &ErrIdentifierConflict{Identifier: identifier}
	}

	if res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	pollUrl := res.Header.Get("Location")
	if pollUrl == "" {
		return nil, fmt.Errorf("missing header location for polling")
	}

	isPiReady := func(data map[string]any, identifier string) (string, bool) {
		if status, ok := data["status"].(string); ok && status == "live" {
			return fmt.Sprintf("pi/servers/%s", identifier), true
		}
		return "", false
	}

	serverUrl, err := c.pollProvisioning(pollUrl, 5*time.Minute, identifier, isPiReady)
	if err != nil {
		return nil, err
	}

	serverRes, err := c.get(serverUrl)
	if err != nil {
		return nil, err
	}

	serverBody, err := c.body(serverRes)
	if err != nil {
		return nil, fmt.Errorf("unexpected status %s", string(serverBody))
	}

	if serverRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch server info: %s", string(serverBody))
	}

	var created Pi
	err = json.Unmarshal(serverBody, &created)
	if err != nil {
		return nil, err
	}

	return &created, nil
}

func (c *Client) DeletePi(identifier string) error {
	if strings.TrimSpace(identifier) == "" {
		return ErrEmptyIdentifier
	}

	url := fmt.Sprintf("/pi/servers/%s", identifier)

	return c.delete(url)
}
