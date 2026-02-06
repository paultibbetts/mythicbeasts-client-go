package vps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Server represents a provisioned VPS.
type Server struct {
	Identifier string      `json:"identifier"`
	Name       string      `json:"name"`
	HostServer string      `json:"host_server"`
	Zone       ServerZone  `json:"zone"`
	Product    string      `json:"product"`
	Family     string      `json:"family"`
	CPUMode    string      `json:"cpu_mode"`
	NetDevice  string      `json:"net_device"`
	DiskBus    string      `json:"disk_bus"`
	Tablet     bool        `json:"tablet"`
	Price      float64     `json:"price"`
	Period     string      `json:"period"`
	ISOImage   string      `json:"iso_image"`
	Dormant    bool        `json:"dormant"`
	BootDevice string      `json:"boot_device"`
	IPv4       []string    `json:"ipv4"`
	IPv6       []string    `json:"ipv6"`
	Specs      ServerSpecs `json:"specs"`
	Macs       []string    `json:"macs"`
	SSHProxy   SSHProxy    `json:"ssh_proxy"`
	VNC        VNC         `json:"vnc"`
}

// ServerZone represents the Zone (datacentre) that a VPS
// is provisioned in.
type ServerZone struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// ServerSpecs represents the specifications of a
// provisioned VPS.
type ServerSpecs struct {
	DiskType   string `json:"disk_type"`
	DiskSize   int64  `json:"disk_size"`
	Cores      int64  `json:"cores"`
	ExtraCores int64  `json:"extra_cores"`
	ExtraRAM   int64  `json:"extra_ram"`
	RAM        int64  `json:"ram"`
}

// SSHProxy represents the details of the
// SSH Proxy in use by the VPS.
type SSHProxy struct {
	Hostname string `json:"hostname"`
	Port     int64  `json:"port"`
}

// VNC represents the details of VNC that should
// be used when provisioning a new VPS.
type VNC struct {
	Mode     string `json:"mode"`
	Password string `json:"password"`
	IPv4     string `json:"ipv4"`
	IPv6     string `json:"ipv6"`
	Port     int64  `json:"port"`
	Display  int64  `json:"display"`
}

// Get retrieves the details for the VPS with the given identifier.
// Returns ErrEmptyIdentifier if the identifier is blank.
func (s *Service) Get(ctx context.Context, identifier string) (Server, error) {
	if strings.TrimSpace(identifier) == "" {
		return Server{}, ErrEmptyIdentifier
	}
	url := fmt.Sprintf("/vps/servers/%s", identifier)

	var result Server
	if _, _, err := s.GetJSON(ctx, url, &result); err != nil {
		return Server{}, err
	}

	return result, nil
}

// CreateRequest represents the data required for provisioning a VPS.
// Some fields are optional and some are only used on creation.
type CreateRequest struct {
	Product        string `json:"product"`
	Name           string `json:"name,omitempty"`
	HostServer     string `json:"host_server,omitempty"`
	Hostname       string `json:"hostname,omitempty"`
	SetForwardDNS  bool   `json:"set_forward_dns,omitempty"`
	SetReverseDNS  bool   `json:"set_reverse_dns,omitempty"`
	DiskType       string `json:"disk_type,omitempty"`
	DiskSize       int64  `json:"disk_size"`
	ExtraCores     int64  `json:"extra_cores,omitempty"`
	ExtraRAM       int64  `json:"extra_ram,omitempty"`
	IPv4           bool   `json:"ipv4"`
	Zone           string `json:"zone,omitempty"`
	Image          string `json:"image"`
	UserData       string `json:"user_data,omitempty"` // id or name
	UserDataString string `json:"user_data_string,omitempty"`
	SSHKeys        string `json:"ssh_keys"`
	VNC            NewVNC `json:"vnc"`
	CPUMode        string `json:"cpu_mode,omitempty"`
	NetDevice      string `json:"net_device,omitempty"`
	DiskBus        string `json:"disk_bus,omitempty"`
	Tablet         bool   `json:"tablet"`
}

// NewVNC represents the data required to set VNC details when
// provisioning a new VPS.
type NewVNC struct {
	Mode     string `json:"mode,omitempty"`
	Password string `json:"password,omitempty"`
}

// Create provisions a new VPS with the given identifier and
// request parameters.
//
// It blocks until the server becomes live or the timeout
// is reached.
// Returns ErrIdentifierConflict if the identifier is already in use.
func (s *Service) Create(ctx context.Context, identifier string, server CreateRequest) (Server, error) {
	requestURL := fmt.Sprintf("/vps/servers/%s", identifier)

	requestJson, err := json.Marshal(server)
	if err != nil {
		return Server{}, err
	}

	req, err := s.NewRequest(ctx, http.MethodPost, requestURL, bytes.NewBuffer(requestJson))
	if err != nil {
		return Server{}, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := s.Do(req)
	if err != nil {
		return Server{}, err
	}

	body, err := s.Body(res)
	if err != nil {
		return Server{}, fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	if res.StatusCode == http.StatusConflict {
		return Server{}, &ErrIdentifierConflict{Identifier: identifier}
	}

	if res.StatusCode != http.StatusAccepted {
		return Server{}, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	pollURL := res.Header.Get("Location")
	if pollURL == "" {
		return Server{}, fmt.Errorf("missing header location for polling")
	}

	isVPSReady := func(data map[string]any, identifier string) (string, bool) {
		if status, ok := data["status"].(string); ok && status == "running" {
			return fmt.Sprintf("/vps/servers/%s", identifier), true
		}
		return "", false
	}

	serverURL, err := s.PollProvisioning(ctx, pollURL, 5*time.Minute, identifier, isVPSReady)
	if err != nil {
		return Server{}, err
	}

	serverRes, err := s.BaseService.Get(ctx, serverURL)
	if err != nil {
		return Server{}, err
	}

	serverBody, err := s.Body(serverRes)
	if err != nil {
		return Server{}, fmt.Errorf("unexpected status %s", string(serverBody))
	}

	if serverRes.StatusCode != http.StatusOK {
		return Server{}, fmt.Errorf("failed to fetch server info: %s", string(serverBody))
	}

	var created Server
	err = json.Unmarshal(serverBody, &created)
	if err != nil {
		return Server{}, err
	}

	return created, nil
}

// Delete removes a provisioned VPS.
//
// Returns ErrEmptyIdentifier if the identifier is blank.
// Considers a 404 as a successful deletion.
func (s *Service) Delete(ctx context.Context, identifier string) error {
	if strings.TrimSpace(identifier) == "" {
		return ErrEmptyIdentifier
	}

	url := fmt.Sprintf("/vps/servers/%s", identifier)

	return s.BaseService.Delete(ctx, url)
}
