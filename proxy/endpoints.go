package proxy

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"path"
	"strings"

	"github.com/paultibbetts/mythicbeasts-client-go/internal/transport"
)

// BaseURL is the default base URL for the Proxy API.
const BaseURL string = "https://api.mythic-beasts.com/proxy"

// Service provides access to the Proxy API.
type Service struct {
	transport.BaseService
}

// NewService constructs a Proxy API service client.
func NewService(c transport.Requester) *Service {
	return &Service{BaseService: transport.NewBaseService(c, BaseURL)}
}

// Endpoint represents a proxy endpoint configuration.
type Endpoint struct {
	Domain        string   `json:"domain"`
	Hostname      string   `json:"hostname"`
	Address       IPv6Addr `json:"address"`
	Site          string   `json:"site"`
	ProxyProtocol bool     `json:"proxy_protocol"`
}

type IPv6Addr struct {
	netip.Addr
}

func (a *IPv6Addr) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	parsed, err := parseIPv6Addr(s)
	if err != nil {
		return err
	}

	a.Addr = parsed.Addr
	return nil
}

func (a *IPv6Addr) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Addr.String())
}

type endpointsResponse struct {
	Endpoints []Endpoint `json:"endpoints"`
}

type EndpointRequest struct {
	Domain        string   `json:"domain,omitempty"`
	Hostname      string   `json:"hostname,omitempty"`
	Address       IPv6Addr `json:"address"`
	Site          string   `json:"site,omitempty"`
	ProxyProtocol bool     `json:"proxy_protocol"`
}

type endpointsRequest struct {
	Endpoints []EndpointRequest `json:"endpoints"`
}

// ListEndpoints retrieves all endpoints, optionally filtered by domain.
func (s *Service) ListEndpoints(ctx context.Context, domain string) ([]Endpoint, error) {
	endpoint := "/endpoints"
	if strings.TrimSpace(domain) != "" {
		endpoint = "/" + path.Join("endpoints", domain)
	}

	var result endpointsResponse
	if _, _, err := s.GetJSON(ctx, endpoint, &result, http.StatusOK); err != nil {
		return nil, err
	}

	return result.Endpoints, nil
}

// GetEndpoints retrieves endpoints for a specific hostname (and optionally address/site).
// A 404 response is treated as "not found" and returns found=false with no error.
func (s *Service) GetEndpoints(ctx context.Context, domain, hostname, address, site string) ([]Endpoint, bool, error) {
	endpoint, err := endpointPath(domain, hostname, address, site)
	if err != nil {
		return nil, false, err
	}

	res, err := s.BaseService.Get(ctx, endpoint)
	if err != nil {
		return nil, false, err
	}

	body, err := s.Body(res)
	if err != nil {
		return nil, false, err
	}

	if res.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("unexpected status %d: %s", res.StatusCode, string(body))
	}

	var result endpointsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, false, err
	}

	return result.Endpoints, true, nil
}

// GetEndpoint retrieves a single endpoint for a fully-qualified path.
// A 404 response is treated as "not found" and returns found=false with no error.
func (s *Service) GetEndpoint(ctx context.Context, domain, hostname, address, site string) (Endpoint, bool, error) {
	if strings.TrimSpace(domain) == "" {
		return Endpoint{}, false, errors.New("domain is required")
	}
	if strings.TrimSpace(hostname) == "" {
		return Endpoint{}, false, errors.New("hostname is required")
	}
	if strings.TrimSpace(address) == "" {
		return Endpoint{}, false, errors.New("address is required")
	}
	if strings.TrimSpace(site) == "" {
		return Endpoint{}, false, errors.New("site is required")
	}

	endpoints, found, err := s.GetEndpoints(ctx, domain, hostname, address, site)
	if err != nil || !found {
		return Endpoint{}, found, err
	}
	if len(endpoints) == 0 {
		return Endpoint{}, false, errors.New("expected 1 endpoint, got 0")
	}
	if len(endpoints) > 1 {
		return Endpoint{}, false, fmt.Errorf("expected 1 endpoint, got %d", len(endpoints))
	}

	return endpoints[0], true, nil
}

// AddEndpointsForHost adds endpoints for a specific domain and hostname.
func (s *Service) AddEndpointsForHost(ctx context.Context, domain, hostname string, endpoints []EndpointRequest) ([]Endpoint, error) {
	endpoint, err := endpointPath(domain, hostname, "", "")
	if err != nil {
		return nil, err
	}

	requests, err := normalizeEndpointRequests(domain, hostname, "", "", endpoints)
	if err != nil {
		return nil, err
	}

	var result endpointsResponse
	if _, _, err := s.DoJSON(ctx, http.MethodPost, endpoint, endpointsRequest{Endpoints: requests}, &result, http.StatusOK); err != nil {
		return nil, err
	}

	return result.Endpoints, nil
}

// CreateOrUpdateEndpoints creates or updates endpoints by replacing any that match the provided path.
func (s *Service) CreateOrUpdateEndpoints(ctx context.Context, domain, hostname, address, site string, endpoints []EndpointRequest) ([]Endpoint, error) {
	endpoint, err := endpointPath(domain, hostname, address, site)
	if err != nil {
		return nil, err
	}

	requests, err := normalizeEndpointRequests(domain, hostname, address, site, endpoints)
	if err != nil {
		return nil, err
	}

	var result endpointsResponse
	if _, _, err := s.DoJSON(ctx, http.MethodPut, endpoint, endpointsRequest{Endpoints: requests}, &result, http.StatusOK); err != nil {
		return nil, err
	}

	return result.Endpoints, nil
}

// DeleteEndpoints deletes endpoints matching the provided path.
func (s *Service) DeleteEndpoints(ctx context.Context, domain, hostname, address, site string) error {
	endpoint, err := endpointPath(domain, hostname, address, site)
	if err != nil {
		return err
	}

	_, _, err = s.DoJSON(ctx, http.MethodDelete, endpoint, nil, nil, http.StatusOK)
	return err
}

func endpointPath(domain, hostname, address, site string) (string, error) {
	if strings.TrimSpace(domain) == "" {
		return "", errors.New("domain is required")
	}
	if strings.TrimSpace(hostname) == "" {
		return "", errors.New("hostname is required")
	}

	parts := []string{"endpoints", domain, hostname}
	if strings.TrimSpace(address) != "" {
		parts = append(parts, address)
		if strings.TrimSpace(site) != "" {
			parts = append(parts, site)
		}
	} else if strings.TrimSpace(site) != "" {
		return "", errors.New("site requires address")
	}

	return "/" + path.Join(parts...), nil
}

func parseIPv6Addr(s string) (IPv6Addr, error) {
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return IPv6Addr{}, err
	}
	if err := validateIPv6Addr(addr); err != nil {
		return IPv6Addr{}, err
	}
	return IPv6Addr{Addr: addr}, nil
}

func validateIPv6Addr(addr netip.Addr) error {
	if !addr.IsValid() {
		return errors.New("address is required")
	}
	if !addr.Is6() {
		return fmt.Errorf("address %q is not IPv6", addr.String())
	}
	if addr.Is4In6() {
		return fmt.Errorf("address %q is IPv4-mapped, not pure IPv6", addr.String())
	}
	return nil
}

func normalizeEndpointRequests(domain, hostname, address, site string, endpoints []EndpointRequest) ([]EndpointRequest, error) {
	normalized := make([]EndpointRequest, len(endpoints))
	var pathAddr IPv6Addr
	var hasPathAddr bool
	if strings.TrimSpace(address) != "" {
		parsed, err := parseIPv6Addr(address)
		if err != nil {
			return nil, err
		}
		pathAddr = parsed
		hasPathAddr = true
	}

	for i, endpoint := range endpoints {
		if endpoint.Domain != "" && endpoint.Domain != domain {
			return nil, fmt.Errorf("domain %q does not match path domain %q", endpoint.Domain, domain)
		}
		if endpoint.Hostname != "" && endpoint.Hostname != hostname {
			return nil, fmt.Errorf("hostname %q does not match path hostname %q", endpoint.Hostname, hostname)
		}

		endpoint.Domain = domain
		endpoint.Hostname = hostname

		if hasPathAddr {
			if endpoint.Address.Addr.IsValid() && endpoint.Address.Addr != pathAddr.Addr {
				return nil, fmt.Errorf("address %q does not match path address %q", endpoint.Address.Addr, pathAddr.Addr)
			}
			endpoint.Address = pathAddr
		}

		if strings.TrimSpace(site) != "" {
			if endpoint.Site != "" && endpoint.Site != site {
				return nil, fmt.Errorf("site %q does not match path site %q", endpoint.Site, site)
			}
			endpoint.Site = site
		}

		if err := validateIPv6Addr(endpoint.Address.Addr); err != nil {
			return nil, err
		}

		normalized[i] = endpoint
	}

	return normalized, nil
}

func (s *Service) ListSites(ctx context.Context) ([]string, error) {
	var resp struct {
		Sites []string `json:"sites"`
	}
	_, _, err := s.GetJSON(ctx, "/sites", &resp)
	if err != nil {
		return nil, err
	}

	return resp.Sites, nil
}
