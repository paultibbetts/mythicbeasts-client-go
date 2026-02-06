package proxy_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	"github.com/paultibbetts/mythicbeasts-client-go"
	proxyapi "github.com/paultibbetts/mythicbeasts-client-go/proxy"
)

func newTestClient(t *testing.T, mux *http.ServeMux) (*mythicbeasts.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	c, _ := mythicbeasts.NewClient("", "")
	c.Proxy().BaseURL = srv.URL
	return c, srv
}

func testContext() context.Context {
	return context.Background()
}

func mustParseAddr(t *testing.T, addr string) netip.Addr {
	t.Helper()
	parsed, err := netip.ParseAddr(addr)
	if err != nil {
		t.Fatalf("parse addr: %v", err)
	}
	return parsed
}

func TestAddEndpointsForHost_OK(t *testing.T) {
	t.Parallel()
	endpoint := proxyapi.Endpoint{
		Domain:        "example.com",
		Hostname:      "www",
		Address:       proxyapi.IPv6Addr{Addr: mustParseAddr(t, "2a00:1098:0:82:1000:3b:1:1")},
		Site:          "all",
		ProxyProtocol: true,
	}
	endpointReq := proxyapi.EndpointRequest{
		Address:       endpoint.Address,
		Site:          endpoint.Site,
		ProxyProtocol: endpoint.ProxyProtocol,
	}
	expectedReq := proxyapi.EndpointRequest{
		Domain:        endpoint.Domain,
		Hostname:      endpoint.Hostname,
		Address:       endpoint.Address,
		Site:          endpoint.Site,
		ProxyProtocol: endpoint.ProxyProtocol,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%s, want application/json", ct)
		}

		var req struct {
			Endpoints []proxyapi.EndpointRequest `json:"endpoints"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if len(req.Endpoints) != 1 {
			t.Fatalf("endpoints=%d, want 1", len(req.Endpoints))
		}
		got := req.Endpoints[0]
		if got.Domain != expectedReq.Domain || got.Hostname != expectedReq.Hostname || got.Site != expectedReq.Site || got.ProxyProtocol != expectedReq.ProxyProtocol {
			t.Fatalf("endpoint=%+v, want %+v", got, expectedReq)
		}
		if got.Address.Addr != expectedReq.Address.Addr {
			t.Fatalf("address=%s, want %s", got.Address.Addr, expectedReq.Address.Addr)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string][]proxyapi.Endpoint{
			"endpoints": []proxyapi.Endpoint{endpoint},
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	got, err := c.Proxy().AddEndpointsForHost(testContext(), "example.com", "www", []proxyapi.EndpointRequest{endpointReq})
	if err != nil {
		t.Fatalf("AddEndpointsForHost: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("endpoints=%d, want 1", len(got))
	}
	if got[0].Domain != endpoint.Domain || got[0].Hostname != endpoint.Hostname || got[0].Site != endpoint.Site || got[0].ProxyProtocol != endpoint.ProxyProtocol {
		t.Fatalf("endpoint=%+v, want %+v", got[0], endpoint)
	}
	if got[0].Address.Addr != endpoint.Address.Addr {
		t.Fatalf("address=%s, want %s", got[0].Address.Addr, endpoint.Address.Addr)
	}
}

func TestAddEndpointsForHost_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	endpoint := proxyapi.Endpoint{
		Domain:   "example.com",
		Hostname: "www",
		Address:  proxyapi.IPv6Addr{Addr: mustParseAddr(t, "2a00:1098:0:82:1000:3b:1:1")},
	}
	endpointReq := proxyapi.EndpointRequest{
		Address: endpoint.Address,
	}

	_, err := c.Proxy().AddEndpointsForHost(testContext(), "example.com", "www", []proxyapi.EndpointRequest{endpointReq})
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}
	if err.Error() != "unexpected status 400: bad payload" {
		t.Fatalf("err=%q, want unexpected status error", err.Error())
	}
}

func TestGetEndpoints_OK(t *testing.T) {
	t.Parallel()
	endpoint := proxyapi.Endpoint{
		Domain:        "example.com",
		Hostname:      "www",
		Address:       proxyapi.IPv6Addr{Addr: mustParseAddr(t, "2a00:1098:0:82:1000:3b:1:1")},
		Site:          "all",
		ProxyProtocol: true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string][]proxyapi.Endpoint{
			"endpoints": []proxyapi.Endpoint{endpoint},
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	got, found, err := c.Proxy().GetEndpoints(testContext(), "example.com", "www", "", "")
	if err != nil {
		t.Fatalf("GetEndpoints: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true")
	}
	if len(got) != 1 {
		t.Fatalf("endpoints=%d, want 1", len(got))
	}
	if got[0].Domain != endpoint.Domain || got[0].Hostname != endpoint.Hostname || got[0].Site != endpoint.Site || got[0].ProxyProtocol != endpoint.ProxyProtocol {
		t.Fatalf("endpoint=%+v, want %+v", got[0], endpoint)
	}
	if got[0].Address.Addr != endpoint.Address.Addr {
		t.Fatalf("address=%s, want %s", got[0].Address.Addr, endpoint.Address.Addr)
	}
}

func TestGetEndpoints_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, found, err := c.Proxy().GetEndpoints(testContext(), "example.com", "www", "", "")
	if err != nil {
		t.Fatalf("GetEndpoints: %v", err)
	}
	if found {
		t.Fatalf("expected found=false")
	}
}

func TestGetEndpoint_OK(t *testing.T) {
	t.Parallel()
	endpoint := proxyapi.Endpoint{
		Domain:        "example.com",
		Hostname:      "www",
		Address:       proxyapi.IPv6Addr{Addr: mustParseAddr(t, "2a00:1098:0:82:1000:3b:1:1")},
		Site:          "all",
		ProxyProtocol: true,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www/2a00:1098:0:82:1000:3b:1:1/all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string][]proxyapi.Endpoint{
			"endpoints": []proxyapi.Endpoint{endpoint},
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	got, found, err := c.Proxy().GetEndpoint(testContext(), "example.com", "www", "2a00:1098:0:82:1000:3b:1:1", "all")
	if err != nil {
		t.Fatalf("GetEndpoint: %v", err)
	}
	if !found {
		t.Fatalf("expected found=true")
	}
	if got.Domain != endpoint.Domain || got.Hostname != endpoint.Hostname || got.Site != endpoint.Site || got.ProxyProtocol != endpoint.ProxyProtocol {
		t.Fatalf("endpoint=%+v, want %+v", got, endpoint)
	}
	if got.Address.Addr != endpoint.Address.Addr {
		t.Fatalf("address=%s, want %s", got.Address.Addr, endpoint.Address.Addr)
	}
}

func TestGetEndpoint_NotFound(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www/2a00:1098:0:82:1000:3b:1:1/all", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, found, err := c.Proxy().GetEndpoint(testContext(), "example.com", "www", "2a00:1098:0:82:1000:3b:1:1", "all")
	if err != nil {
		t.Fatalf("GetEndpoint: %v", err)
	}
	if found {
		t.Fatalf("expected found=false")
	}
}

func TestCreateOrUpdateEndpoints_OK(t *testing.T) {
	t.Parallel()
	endpoint := proxyapi.Endpoint{
		Domain:        "example.com",
		Hostname:      "www",
		Address:       proxyapi.IPv6Addr{Addr: mustParseAddr(t, "2a00:1098:0:82:1000:3b:1:1")},
		Site:          "all",
		ProxyProtocol: true,
	}
	endpointReq := proxyapi.EndpointRequest{
		ProxyProtocol: endpoint.ProxyProtocol,
	}
	expectedReq := proxyapi.EndpointRequest{
		Domain:        endpoint.Domain,
		Hostname:      endpoint.Hostname,
		Address:       endpoint.Address,
		Site:          endpoint.Site,
		ProxyProtocol: endpoint.ProxyProtocol,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www/2a00:1098:0:82:1000:3b:1:1/all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method=%s, want PUT", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%s, want application/json", ct)
		}

		var req struct {
			Endpoints []proxyapi.EndpointRequest `json:"endpoints"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if len(req.Endpoints) != 1 {
			t.Fatalf("endpoints=%d, want 1", len(req.Endpoints))
		}
		got := req.Endpoints[0]
		if got.Domain != expectedReq.Domain || got.Hostname != expectedReq.Hostname || got.Site != expectedReq.Site || got.ProxyProtocol != expectedReq.ProxyProtocol {
			t.Fatalf("endpoint=%+v, want %+v", got, expectedReq)
		}
		if got.Address.Addr != expectedReq.Address.Addr {
			t.Fatalf("address=%s, want %s", got.Address.Addr, expectedReq.Address.Addr)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string][]proxyapi.Endpoint{
			"endpoints": []proxyapi.Endpoint{endpoint},
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	got, err := c.Proxy().CreateOrUpdateEndpoints(testContext(), "example.com", "www", "2a00:1098:0:82:1000:3b:1:1", "all", []proxyapi.EndpointRequest{endpointReq})
	if err != nil {
		t.Fatalf("CreateOrUpdateEndpoints: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("endpoints=%d, want 1", len(got))
	}
}

func TestCreateOrUpdateEndpoints_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www/2a00:1098:0:82:1000:3b:1:1/all", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	endpoint := proxyapi.Endpoint{
		Domain:   "example.com",
		Hostname: "www",
		Address:  proxyapi.IPv6Addr{Addr: mustParseAddr(t, "2a00:1098:0:82:1000:3b:1:1")},
		Site:     "all",
	}
	endpointReq := proxyapi.EndpointRequest{
		Domain:   endpoint.Domain,
		Hostname: endpoint.Hostname,
		Address:  endpoint.Address,
		Site:     endpoint.Site,
	}

	_, err := c.Proxy().CreateOrUpdateEndpoints(testContext(), "example.com", "www", "2a00:1098:0:82:1000:3b:1:1", "all", []proxyapi.EndpointRequest{endpointReq})
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}
	if err.Error() != "unexpected status 400: bad payload" {
		t.Fatalf("err=%q, want unexpected status error", err.Error())
	}
}

func TestDeleteEndpoints_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www/2a00:1098:0:82:1000:3b:1:1/all", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method=%s, want DELETE", r.Method)
		}
		w.WriteHeader(http.StatusOK)
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := c.Proxy().DeleteEndpoints(testContext(), "example.com", "www", "2a00:1098:0:82:1000:3b:1:1", "all"); err != nil {
		t.Fatalf("DeleteEndpoints: %v", err)
	}
}

func TestDeleteEndpoints_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/endpoints/example.com/www/2a00:1098:0:82:1000:3b:1:1/all", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	err := c.Proxy().DeleteEndpoints(testContext(), "example.com", "www", "2a00:1098:0:82:1000:3b:1:1", "all")
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}
	if err.Error() != "unexpected status 400: bad payload" {
		t.Fatalf("err=%q, want unexpected status error", err.Error())
	}
}
