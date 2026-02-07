package vps

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// Server represents a provisioned VPS.
type Server struct {
	Identifier string      `json:"identifier"`
	Name       string      `json:"name"`
	Status     string      `json:"status"`
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

// VNC represents VNC connection details for a provisioned VPS.
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
	if _, _, err := s.GetJSON(ctx, url, &result, http.StatusOK); err != nil {
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
	IPv4           bool   `json:"ipv4,omitempty"`
	Zone           string `json:"zone,omitempty"`
	Image          string `json:"image,omitempty"`
	UserData       string `json:"user_data,omitempty"` // id or name
	UserDataString string `json:"user_data_string,omitempty"`
	SSHKeys        string `json:"ssh_keys,omitempty"`
	CPUMode        string `json:"cpu_mode,omitempty"`
	NetDevice      string `json:"net_device,omitempty"`
	DiskBus        string `json:"disk_bus,omitempty"`
	Tablet         *bool  `json:"tablet,omitempty"`
}

// SetTablet includes the tablet field in create requests.
func (r *CreateRequest) SetTablet(v bool) { r.Tablet = &v }

// UnsetTablet omits the tablet field from create requests.
func (r *CreateRequest) UnsetTablet() { r.Tablet = nil }

// Bool returns a pointer to v.
func Bool(v bool) *bool { return &v }

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
		status, _ := data["status"].(string)
		log.Printf("vps[%s] provisioning status=%q", identifier, status)
		if status == "running" {
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

// UpdateSpecs represents updatable VPS specification fields.
type UpdateSpecs struct {
	DiskSize   *int64 `json:"disk_size,omitempty"`
	ExtraCores *int64 `json:"extra_cores,omitempty"`
	ExtraRAM   *int64 `json:"extra_ram,omitempty"`
}

// NewUpdateSpecs constructs an empty specs update payload.
func NewUpdateSpecs() UpdateSpecs {
	return UpdateSpecs{}
}

// SetDiskSize sets disk size in MB.
func (s *UpdateSpecs) SetDiskSize(v int64) { s.DiskSize = &v }

// SetExtraCores sets additional CPU cores.
func (s *UpdateSpecs) SetExtraCores(v int64) { s.ExtraCores = &v }

// SetExtraRAM sets additional RAM in MB.
func (s *UpdateSpecs) SetExtraRAM(v int64) { s.ExtraRAM = &v }

// UpdateRequest represents the fields that can be updated for a VPS.
type UpdateRequest struct {
	Product    *string      `json:"product,omitempty"`
	Specs      *UpdateSpecs `json:"specs,omitempty"`
	Name       *string      `json:"name,omitempty"`
	BootDevice *string      `json:"boot_device,omitempty"`
	ISOImage   *string      `json:"iso_image,omitempty"`
	CPUMode    *string      `json:"cpu_mode,omitempty"`
	NetDevice  *string      `json:"net_device,omitempty"`
	DiskBus    *string      `json:"disk_bus,omitempty"`
	Tablet     *bool        `json:"tablet,omitempty"`

	// nullable fields with tri-state semantics for PATCH:
	// unset (omit), set value, set null.
	clearName     bool
	clearISOImage bool
}

// UpdateResponse represents the response from a VPS update request.
type UpdateResponse struct {
	Message string `json:"message"`
}

// NewUpdateRequest constructs an empty VPS update request.
func NewUpdateRequest() UpdateRequest {
	return UpdateRequest{}
}

// SetProduct sets the VPS product code.
func (r *UpdateRequest) SetProduct(v string) { r.Product = &v }

// SetSpecs sets the VPS specs payload.
func (r *UpdateRequest) SetSpecs(v UpdateSpecs) { r.Specs = &v }

// SetBootDevice sets the boot device.
func (r *UpdateRequest) SetBootDevice(v string) { r.BootDevice = &v }

// SetCPUMode sets the CPU mode.
func (r *UpdateRequest) SetCPUMode(v string) { r.CPUMode = &v }

// SetNetDevice sets the network device type.
func (r *UpdateRequest) SetNetDevice(v string) { r.NetDevice = &v }

// SetDiskBus sets the disk bus type.
func (r *UpdateRequest) SetDiskBus(v string) { r.DiskBus = &v }

// SetTablet sets tablet mode.
func (r *UpdateRequest) SetTablet(v bool) { r.Tablet = &v }

// SetName sets the VPS name (non-null).
func (r *UpdateRequest) SetName(v string) {
	r.Name = &v
	r.clearName = false
}

// ClearName sets the VPS name to null.
func (r *UpdateRequest) ClearName() {
	r.Name = nil
	r.clearName = true
}

// UnsetName omits the name field from the PATCH body.
func (r *UpdateRequest) UnsetName() {
	r.Name = nil
	r.clearName = false
}

// SetISOImage sets the ISO image (non-null).
func (r *UpdateRequest) SetISOImage(v string) {
	r.ISOImage = &v
	r.clearISOImage = false
}

// ClearISOImage sets the ISO image to null.
func (r *UpdateRequest) ClearISOImage() {
	r.ISOImage = nil
	r.clearISOImage = true
}

// UnsetISOImage omits the iso_image field from the PATCH body.
func (r *UpdateRequest) UnsetISOImage() {
	r.ISOImage = nil
	r.clearISOImage = false
}

// MarshalJSON encodes tri-state nullable fields used by PATCH updates.
func (r UpdateRequest) MarshalJSON() ([]byte, error) {
	body := map[string]any{}

	if r.Product != nil {
		body["product"] = *r.Product
	}
	if r.Specs != nil {
		body["specs"] = r.Specs
	}
	if r.BootDevice != nil {
		body["boot_device"] = *r.BootDevice
	}
	if r.CPUMode != nil {
		body["cpu_mode"] = *r.CPUMode
	}
	if r.NetDevice != nil {
		body["net_device"] = *r.NetDevice
	}
	if r.DiskBus != nil {
		body["disk_bus"] = *r.DiskBus
	}
	if r.Tablet != nil {
		body["tablet"] = *r.Tablet
	}

	switch {
	case r.clearName:
		body["name"] = nil
	case r.Name != nil:
		body["name"] = *r.Name
	}

	switch {
	case r.clearISOImage:
		body["iso_image"] = nil
	case r.ISOImage != nil:
		body["iso_image"] = *r.ISOImage
	}

	return json.Marshal(body)
}

// RequiresPoweredOff reports whether this update includes fields that
// the API requires the VPS to be powered off before changing.
func (r UpdateRequest) RequiresPoweredOff() bool {
	return r.BootDevice != nil ||
		r.ISOImage != nil ||
		r.clearISOImage ||
		r.CPUMode != nil ||
		r.NetDevice != nil ||
		r.DiskBus != nil ||
		r.Tablet != nil
}

// Update updates the settings for a provisioned VPS.
//
// Returns ErrEmptyIdentifier if the identifier is blank.
func (s *Service) Update(ctx context.Context, identifier string, req UpdateRequest) (UpdateResponse, error) {
	if strings.TrimSpace(identifier) == "" {
		return UpdateResponse{}, ErrEmptyIdentifier
	}

	url := fmt.Sprintf("/vps/servers/%s", identifier)

	var result UpdateResponse
	if _, _, err := s.DoJSON(ctx, http.MethodPatch, url, req, &result, http.StatusOK); err != nil {
		return UpdateResponse{}, err
	}

	return result, nil
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
