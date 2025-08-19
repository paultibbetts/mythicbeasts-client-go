package mythicbeasts

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// DiskSizes represents the available disk sizes for a VPS.
type DiskSizes struct {
	HDD []int64 `json:"hdd"`
	SSD []int64 `json:"ssd"`
}

// GetVPSDiskSizes retrieves the list of available disk sizes.
func (c *Client) GetVPSDiskSizes() (*DiskSizes, error) {
	res, err := c.get("/vps/disk-sizes")
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	var result DiskSizes
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// VPSImage represents a VPS operating system image.
type VPSImage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// VPSImages repreesnts a list of VPSImages.
type VPSImages map[string]VPSImage

// GetVPSImages retrieves the list of available operating
// system images available for a VPS.
func (c *Client) GetVPSImages() (VPSImages, error) {
	res, err := c.get("/vps/images")
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	var result VPSImages
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// Zones represents a list of available Zones a VPS may be
// provisioned in.
type Zones map[string]Zone

// Zone represents a zone (datacentre) a VPS may be
// provisioned in. It can include its parent zones.
type Zone struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parents     []string `json:"parents"`
}

// GetVPSZones retrieves the list of available zones
// a VPS may be provisioned in.
func (c *Client) GetVPSZones() (Zones, error) {
	res, err := c.get("/vps/zones")
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	var result Zones
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// VPSHosts represents a list of VPSHostInfo
type VPSHosts map[string]VPSHostInfo

// VPSHostInfo represents an available private cloud host.
type VPSHostInfo struct {
	Name     string          `json:"name"`
	Cores    int64           `json:"cores"`
	RAM      int64           `json:"ram"`
	Disk     VPSHostDiskInfo `json:"disk"`
	FreeRAM  int64           `json:"free_ram"`
	FreeDisk VPSHostDiskInfo `json:"free_disk"`
}

// VPSHostDiskInfo represents the disk information of a
// VPSHost.
type VPSHostDiskInfo struct {
	SSD int64 `json:"ssd"`
	HDD int64 `json:"hdd"`
}

// GetVPSHosts retrieves the list of available private cloud
// hosts.
func (c *Client) GetVPSHosts() (VPSHosts, error) {
	res, err := c.get("/vps/hosts")
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	var result VPSHosts
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// VPSPricing represents the pricing information
// used for on-demand VPS resources.
type VPSPricing struct {
	Disk     VPSDiskPrices    `json:"disk"`
	IPv4     int64            `json:"ipv4"`
	Products map[string]int64 `json:"products"`
}

// VPSDiskPrices represents the pricing information
// for different disk types available for a VPS.
type VPSDiskPrices struct {
	SSD VPSDiskPricing `json:"ssd"`
	HDD VPSDiskPricing `json:"hdd"`
}

// VPSDiskPricing represents the price of a type of
// disk available to a VPS. The extent represents
// the price per GB per unit of disk space.
type VPSDiskPricing struct {
	Price  int64 `json:"price"`
	Extent int64 `json:"extent"`
}

// GetVPSPricing retreives the VPSPricing for
// on-demand VPS products.
func (c *Client) GetVPSPricing() (VPSPricing, error) {
	res, err := c.get("/vps/pricing")
	if err != nil {
		return VPSPricing{}, err
	}

	body, err := c.body(res)
	if err != nil {
		return VPSPricing{}, err
	}

	var result VPSPricing
	err = json.Unmarshal(body, &result)
	if err != nil {
		return VPSPricing{}, err
	}

	return result, nil
}

// VPSZone repreents the Zone (datacentre) that a VPS
// is provisioned in.
type VPSZone struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// VPS represents a provisioned VPS.
type VPS struct {
	Identifier string   `json:"identifier"`
	Name       string   `json:"name"`
	HostServer string   `json:"host_server"`
	Zone       VPSZone  `json:"zone"`
	Product    string   `json:"product"`
	Family     string   `json:"family"`
	CPUMode    string   `json:"cpu_mode"`
	NetDevice  string   `json:"net_device"`
	DiskBus    string   `json:"disk_bus"`
	Tablet     bool     `json:"tablet"`
	Price      float64  `json:"price"`
	Period     string   `json:"period"`
	ISOImage   string   `json:"iso_image"`
	Dormant    bool     `json:"dormant"`
	BootDevice string   `json:"boot_device"`
	IPv4       []string `json:"ipv4"`
	IPv6       []string `json:"ipv6"`
	Specs      VPSSpecs `json:"specs"`
	Macs       []string `json:"macs"`
	SSHProxy   SSHProxy `json:"ssh_proxy"`
	VNC        VNC      `json:"vnc"`
}

// VPSSpecs represents the specifications of a
// provisioned VPS.
type VPSSpecs struct {
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

// ErrEmptyIdentifier is returned when an identifier is not used.
// Identifiers are required for all VPS resources.
var ErrEmptyIdentifier = errors.New("identifier is required")

// GetVPS retrieves the details for the VPS with the given identifier.
// Returns ErrEmptyIdentifier if the identifier is blank.
func (c *Client) GetVPS(identifier string) (VPS, error) {
	if strings.TrimSpace(identifier) == "" {
		return VPS{}, ErrEmptyIdentifier
	}
	url := fmt.Sprintf("/vps/servers/%s", identifier)

	res, err := c.get(url)
	if err != nil {
		return VPS{}, err
	}
	body, err := c.body(res)
	if err != nil {
		return VPS{}, err
	}

	var result VPS
	err = json.Unmarshal(body, &result)
	if err != nil {
		return VPS{}, err
	}

	return result, nil
}

// VPSProduct represents an available VPS product.
// Defaults to a Period of "on-demand" which
// returns the products that can be provisioned using
// the client.
type VPSProduct struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Code        string          `json:"code"`
	Family      string          `json:"family"`
	Period      string          `json:"period"`
	Specs       VPSProductSpecs `json:"specs"`
}

// VPSProductSpecs represents the specifications of a
// VPSProduct.
type VPSProductSpecs struct {
	Cores     int `json:"cores"`
	RAM       int `json:"ram"`
	Bandwidth int `json:"bandwidth"`
}

// GetVPSProducts retrieves all VPSProducts available.
func (c *Client) GetVPSProducts() ([]VPSProduct, error) {
	res, err := c.get("/vps/products")
	if err != nil {
		return nil, err
	}

	body, err := c.body(res)
	if err != nil {
		return nil, err
	}

	var result map[string]VPSProduct
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	products := make([]VPSProduct, 0, len(result))
	for _, product := range result {
		products = append(products, product)
	}

	var numberRegex = regexp.MustCompile(`\d+`)

	sort.Slice(products, func(i, j int) bool {
		ni := numberRegex.FindString(products[i].Name)
		nj := numberRegex.FindString(products[j].Name)

		vi, _ := strconv.Atoi(ni)
		vj, _ := strconv.Atoi(nj)

		return vi < vj
	})

	return products, nil
}

// NewVPS represents the data required for provisioning a VPS.
// Some fields are optional and some are only used on creation.
type NewVPS struct {
	Product        string `json:"product"`
	Name           string `json:"name,omitempty"`
	HostServer     string `json:"host_server,omitempty"`
	HostName       string `json:"hostname,omitempty"`
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

// CreateVPS provisions a new VPS with the given identifier and
// request parameters. It blocks until the server becomes live or the timeout
// is reached. Returns ErrIdentifierConflict if the identifier is already in use.
func (c *Client) CreateVPS(identifier string, server NewVPS) (VPS, error) {
	requestUrl := fmt.Sprintf("/vps/servers/%s", identifier)

	requestJson, err := json.Marshal(server)
	if err != nil {
		return VPS{}, err
	}

	req, err := c.NewRequest(http.MethodPost, requestUrl, bytes.NewBuffer(requestJson))
	if err != nil {
		return VPS{}, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := c.do(req)
	if err != nil {
		return VPS{}, err
	}

	body, err := c.body(res)
	if err != nil {
		return VPS{}, fmt.Errorf("unexpected status %d", res.StatusCode)
	}

	if res.StatusCode == http.StatusConflict {
		return VPS{}, &ErrIdentifierConflict{Identifier: identifier}
	}

	if res.StatusCode != http.StatusAccepted {
		return VPS{}, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	pollUrl := res.Header.Get("Location")
	if pollUrl == "" {
		return VPS{}, fmt.Errorf("missing header location for polling")
	}

	isVPSReady := func(data map[string]any, identifier string) (string, bool) {
		if status, ok := data["status"].(string); ok && status == "running" {
			return fmt.Sprintf("/vps/servers/%s", identifier), true
		}
		return "", false
	}

	serverUrl, err := c.pollProvisioning(pollUrl, 5*time.Minute, identifier, isVPSReady)
	if err != nil {
		return VPS{}, err
	}

	serverRes, err := c.get(serverUrl)
	if err != nil {
		return VPS{}, err
	}

	serverBody, err := c.body(serverRes)
	if err != nil {
		return VPS{}, fmt.Errorf("unexpected status %s", string(serverBody))
	}

	if serverRes.StatusCode != http.StatusOK {
		return VPS{}, fmt.Errorf("failed to fetch server info: %s", string(serverBody))
	}

	var created VPS
	err = json.Unmarshal(serverBody, &created)
	if err != nil {
		return VPS{}, err
	}

	return created, nil
}

// DeleteVPS removes a provisioned VPS.
// Returns ErrEmptyIdentifier if the identifier is blank.
// Considers a 404 as a successful deletion.
func (c *Client) DeleteVPS(identifier string) error {
	if strings.TrimSpace(identifier) == "" {
		return ErrEmptyIdentifier
	}

	url := fmt.Sprintf("/vps/servers/%s", identifier)

	return c.delete(url)
}
