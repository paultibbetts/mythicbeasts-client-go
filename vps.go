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

type DiskSizes struct {
	HDD []int64 `json:"hdd"`
	SSD []int64 `json:"ssd"`
}

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

type VPSImage struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type VPSImages map[string]VPSImage

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

type Zones map[string]Zone

type Zone struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parents     []string `json:"parents"`
}

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

type VPSHosts map[string]VPSHostInfo

type VPSHostInfo struct {
	Name     string          `json:"name"`
	Cores    int64           `json:"cores"`
	RAM      int64           `json:"ram"`
	Disk     VPSHostDiskInfo `json:"disk"`
	FreeRAM  int64           `json:"free_ram"`
	FreeDisk VPSHostDiskInfo `json:"free_disk"`
}

type VPSHostDiskInfo struct {
	SSD int64 `json:"ssd"`
	HDD int64 `json:"hdd"`
}

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

type VPSPricing struct {
	Disk     VPSDiskPrices    `json:"disk"`
	IPv4     int64            `json:"ipv4"`
	Products map[string]int64 `json:"products"`
}

type VPSDiskPrices struct {
	SSD VPSDiskPricing `json:"ssd"`
	HDD VPSDiskPricing `json:"hdd"`
}

type VPSDiskPricing struct {
	Price  int64 `json:"price"`
	Extent int64 `json:"extent"`
}

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

type VPSZone struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

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

type VPSSpecs struct {
	DiskType   string `json:"disk_type"`
	DiskSize   int64  `json:"disk_size"`
	Cores      int64  `json:"cores"`
	ExtraCores int64  `json:"extra_cores"`
	ExtraRAM   int64  `json:"extra_ram"`
	RAM        int64  `json:"ram"`
}

type SSHProxy struct {
	Hostname string `json:"hostname"`
	Port     int64  `json:"port"`
}

type VNC struct {
	Mode     string `json:"mode"`
	Password string `json:"password"`
	IPv4     string `json:"ipv4"`
	IPv6     string `json:"ipv6"`
	Port     int64  `json:"port"`
	Display  int64  `json:"display"`
}

var ErrEmptyIdentifier = errors.New("identifier is required")

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

type VPSProduct struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Code        string          `json:"code"`
	Family      string          `json:"family"`
	Period      string          `json:"period"`
	Specs       VPSProductSpecs `json:"specs"`
}

type VPSProductSpecs struct {
	Cores     int `json:"cores"`
	RAM       int `json:"ram"`
	Bandwidth int `json:"bandwidth"`
}

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

type NewVNC struct {
	Mode     string `json:"mode,omitempty"`
	Password string `json:"password,omitempty"`
}

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

func (c *Client) DeleteVPS(identifier string) error {
	url := fmt.Sprintf("/vps/servers/%s", identifier)

	req, err := c.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return err
	}

	_, deleteErr := c.do(req)
	if deleteErr != nil {
		return deleteErr
	}

	return nil
}
