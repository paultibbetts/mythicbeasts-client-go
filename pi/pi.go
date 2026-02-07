package pi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/paultibbetts/mythicbeasts-client-go/internal/transport"
)

// BaseURL is the default base URL for Raspberry Pi API requests.
const BaseURL string = "https://api.mythic-beasts.com/beta"

// Service provides access to the Raspberry Pi API.
type Service struct {
	transport.BaseService
}

// NewService constructs a Raspberry Pi API service client.
func NewService(c transport.Requester) *Service {
	return &Service{BaseService: transport.NewBaseService(c, BaseURL)}
}

// Model represents the specifications of a Pi model
// that can be provisioned by Mythic Beasts.
type Model struct {
	Model    int64 `json:"model"`
	Memory   int64 `json:"memory"`
	NICSpeed int64 `json:"nic_speed"`
	CPUSpeed int64 `json:"cpu_speed"`
}

// ListModels retrieves the list of available Pi models
// that can be provisioned by Mythic Beasts.
func (s *Service) ListModels(ctx context.Context) ([]Model, error) {
	res, err := s.BaseService.Get(ctx, "/pi/models")
	if err != nil {
		return nil, err
	}

	body, err := s.Body(res)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d, %s", res.StatusCode, string(body))
	}

	var result struct {
		Models []Model `json:"models"`
	}

	if err = json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return result.Models, nil
}

// OperatingSystems maps OS identifiers to their display names.
type OperatingSystems map[string]string

// GetOperatingSystems retrieves the available operating
// system images for the specified Pi model.
func (s *Service) GetOperatingSystems(ctx context.Context, model int64) (OperatingSystems, error) {
	url := fmt.Sprintf("/pi/images/%d", model)

	var result OperatingSystems
	_, _, err := s.GetJSON(ctx, url, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Server represents a provisioned Pi server and its attributes.
type Server struct {
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

// Servers represents the list of provisioned Pi servers.
type Servers struct {
	Servers []Server `json:"servers"`
}

// List returns the list of provisioned Pi servers.
// It does **not** return the identifiers, so it is only
// useful for listing all servers.
func (s *Service) List(ctx context.Context) ([]Server, error) {
	var result Servers
	_, _, err := s.GetJSON(ctx, "/pi/servers", &result, http.StatusOK)
	if err != nil {
		return nil, err
	}

	return result.Servers, nil
}

// Get retrieves details for a single Pi server by its identifier.
// Returns ErrEmptyIdentifier if the identifier is blank.
func (s *Service) Get(ctx context.Context, identifier string) (Server, error) {
	if strings.TrimSpace(identifier) == "" {
		return Server{}, ErrEmptyIdentifier
	}
	url := fmt.Sprintf("/pi/servers/%s", identifier)

	var result Server
	_, _, err := s.GetJSON(ctx, url, &result)
	if err != nil {
		return Server{}, err
	}

	return result, nil
}

// CreateRequest represents the parameters for provisioning
// a new Pi server.
type CreateRequest struct {
	Model      int64  `json:"model,omitempty"`
	Memory     int64  `json:"memory,omitempty"`
	CPUSpeed   int64  `json:"cpu_speed,omitempty"`
	DiskSize   int64  `json:"disk,omitempty"` // intentionally different
	SSHKey     string `json:"ssh_key,omitempty"`
	OSImage    string `json:"os_image,omitempty"`
	WaitForDNS bool   `json:"wait_for_dns,omitempty"`
}

// Create provisions a new Pi server with the given identifier and
// request parameters. It blocks until the server becomes live or the timeout
// is reached. Returns ErrIdentifierConflict if the identifier is already in use.
func (s *Service) Create(ctx context.Context, identifier string, server CreateRequest) (*Server, error) {
	requestURL := fmt.Sprintf("/pi/servers/%s", identifier)

	requestJSON, err := json.Marshal(server)
	if err != nil {
		return nil, err
	}

	req, err := s.NewRequest(ctx, http.MethodPost, requestURL, bytes.NewBuffer(requestJSON))
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := s.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := s.Body(res)
	if err != nil {
		return nil, fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	if res.StatusCode == http.StatusConflict {
		return nil, &ErrIdentifierConflict{Identifier: identifier}
	}

	if res.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	pollURL := res.Header.Get("Location")
	if pollURL == "" {
		return nil, fmt.Errorf("missing header location for polling")
	}

	isPiReady := func(data map[string]any, identifier string) (string, bool) {
		if status, ok := data["status"].(string); ok && status == "live" {
			return fmt.Sprintf("/pi/servers/%s", identifier), true
		}
		return "", false
	}

	serverURL, err := s.PollProvisioning(ctx, pollURL, 5*time.Minute, identifier, isPiReady)
	if err != nil {
		return nil, err
	}

	serverRes, err := s.BaseService.Get(ctx, serverURL)
	if err != nil {
		return nil, err
	}

	serverBody, err := s.Body(serverRes)
	if err != nil {
		return nil, fmt.Errorf("unexpected status %s", string(serverBody))
	}

	if serverRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch server info: %s", string(serverBody))
	}

	var created Server
	err = json.Unmarshal(serverBody, &created)
	if err != nil {
		return nil, err
	}

	return &created, nil
}

type UpdateSSHKeyRequest struct {
	SSHKey string `json:"ssh_key"`
}

type UpdateSSHKeyResponse struct {
	SSHKey string `json:"ssh_key"`
}

// UpdateSSHKey will replace the contents of
// /root/.ssh/authorized_keys with the provided key.
// It returns the contents of that file.
func (s *Service) UpdateSSHKey(ctx context.Context, identifier string, req UpdateSSHKeyRequest) (UpdateSSHKeyResponse, error) {
	if strings.TrimSpace(identifier) == "" {
		return UpdateSSHKeyResponse{}, ErrEmptyIdentifier
	}

	if strings.TrimSpace(req.SSHKey) == "" {
		return UpdateSSHKeyResponse{}, errors.New("ssh key is required")
	}

	url := fmt.Sprintf("/pi/servers/%s/ssh-key", identifier)

	var result UpdateSSHKeyResponse
	if _, _, err := s.DoJSON(ctx, http.MethodPut, url, req, &result, http.StatusOK); err != nil {
		return UpdateSSHKeyResponse{}, err
	}

	return result, nil
}

// Delete removes the Pi server with the given identifier.
// Returns ErrEmptyIdentifier if the identifier is blank.
// Considers a 404 as a successful deletion.
func (s *Service) Delete(ctx context.Context, identifier string) error {
	if strings.TrimSpace(identifier) == "" {
		return ErrEmptyIdentifier
	}

	url := fmt.Sprintf("/pi/servers/%s", identifier)

	return s.BaseService.Delete(ctx, url)
}
