package vps_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/paultibbetts/mythicbeasts-client-go"
	vpsapi "github.com/paultibbetts/mythicbeasts-client-go/vps"
)

func newTestClient(t *testing.T, mux *http.ServeMux) (*mythicbeasts.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	c, _ := mythicbeasts.NewClient("", "")
	c.VPS().BaseURL = srv.URL
	return c, srv
}

func testContext() context.Context {
	return context.Background()
}

// DiskSizes

func TestGetDiskSizes_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/disk-sizes", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		_, _ = w.Write([]byte(`{"hdd":[100,200], "ssd":[50, 150]}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	ds, err := c.VPS().GetDiskSizes(testContext())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got, want := ds.HDD, []int64{100, 200}; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("HDD=%v want %v", got, want)
	}
	if got, want := ds.SSD, []int64{50, 150}; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("SSD=%v want %v", got, want)
	}
}

func TestGetDiskSizes_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/disk-sizes", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().GetDiskSizes(testContext())
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// Images

func TestGetImages_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/images", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]vpsapi.Image{
			"ubuntu": {Name: "ubuntu-lts", Description: "Ubuntu LTS"},
			"debian": {Name: "debian-13", Description: "Debian 13"},
		})
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	images, err := c.VPS().GetImages(testContext())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(images) > 2 {
		t.Fatalf("len(images)=%d, want 2", len(images))
	}
	ubuntu, ok := images["ubuntu"]
	if !ok || ubuntu.Name != "ubuntu-lts" || !strings.Contains(ubuntu.Description, "Ubuntu LTS") {
		t.Fatalf("ubuntu image = %+v (ok=%v)", ubuntu, ok)
	}
}

func TestGetImages_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/images", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().GetImages(testContext())
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// Zones

func TestGetZones_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/zones", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]vpsapi.Zone{
			"eu":  {Name: "EU", Description: "EU zone", Parents: []string{}},
			"lon": {Name: "London", Description: "London zone", Parents: []string{"eu"}},
		})
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	zones, err := c.VPS().GetZones(testContext())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(zones) > 2 {
		t.Fatalf("len(zones)=%d, want 2", len(zones))
	}
	zone, ok := zones["lon"]
	if !ok || zone.Name != "London" || !strings.Contains(zone.Description, "London zone") {
		t.Fatalf("london zone = %+v (ok=%v)", zone, ok)
	}
}

func TestGetZones_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/zones", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().GetZones(testContext())
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// Hosts

func TestGetHosts_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/hosts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]vpsapi.Host{
			"one": {Name: "One", Cores: 1, RAM: 64, Disk: vpsapi.HostDisk{SSD: 1, HDD: 1}, FreeRAM: 32, FreeDisk: vpsapi.HostDisk{SSD: 1, HDD: 1}},
			"two": {Name: "Two", Cores: 2, RAM: 64, Disk: vpsapi.HostDisk{SSD: 1, HDD: 1}, FreeRAM: 32, FreeDisk: vpsapi.HostDisk{SSD: 1, HDD: 1}},
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	hosts, err := c.VPS().GetHosts(testContext())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(hosts) > 2 {
		t.Fatalf("len(zones)=%d, want 2", len(hosts))
	}
	host, ok := hosts["two"]
	if !ok || host.Name != "Two" || host.Cores != 2 || host.RAM != 64 || host.Disk.SSD != 1 || host.FreeRAM != 32 || host.FreeDisk.HDD != 1 {
		t.Fatalf("host two = %+v (ok=%v)", host, ok)
	}
}

func TestGetHosts_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/hosts", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().GetHosts(testContext())
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// Pricing

func TestGetPricing_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/pricing", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(vpsapi.Pricing{
			Disk: vpsapi.DiskPrices{
				SSD: vpsapi.DiskPricing{Price: 1, Extent: 1},
				HDD: vpsapi.DiskPricing{Price: 1, Extent: 1},
			},
			IPv4: 2,
			Products: map[string]int64{
				"one": 1,
				"two": 2,
			},
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	pricing, err := c.VPS().GetPricing(testContext())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if pricing.Disk.SSD.Price != 1 || pricing.Disk.SSD.Extent != 1 {
		t.Fatalf("SSD pricing = %+v, want {Price:1, Extent: 1}", pricing.Disk.SSD)
	}
	if pricing.Disk.HDD.Price != 1 || pricing.Disk.SSD.Extent != 1 {
		t.Fatalf("HDD pricing = %+v, want {Price:1, Extent: 1}", pricing.Disk.SSD)
	}
	if pricing.IPv4 != 2 {
		t.Fatalf("IPv4 pricing = %+d, want 2", pricing.IPv4)
	}
	if len(pricing.Products) > 2 || pricing.Products["one"] != 1 || pricing.Products["two"] != 2 {
		t.Fatalf("Products = %+v, want one:1, two:2", pricing.Products)
	}
}

func TestGetPricing_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/pricing", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().GetPricing(testContext())
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// VPS

func TestGet_ByID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"identifier":"my-id",
			"name":"box",
			"zone":{"code":"lon1", "name":"london"}
			}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	v, err := c.VPS().Get(testContext(), "my-id")
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if v.Identifier != "my-id" {
		t.Fatalf("id=%s", v.Identifier)
	}
	if v.Zone.Code != "lon1" {
		t.Fatalf("zone=%s", v.Zone.Code)
	}
}

func TestGet_EmptyIdentifier(t *testing.T) {
	t.Parallel()
	c, _ := mythicbeasts.NewClient("", "")
	_, err := c.VPS().Get(testContext(), "")
	if err == nil {
		t.Fatalf("expected error for empty identifier")
	}
}
