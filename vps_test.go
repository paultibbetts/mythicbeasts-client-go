package mythicbeasts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestClient(t *testing.T, mux *http.ServeMux) (*Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	c, _ := NewClient(nil, nil)
	c.HostURL = srv.URL
	return c, srv
}

// DiskSizes

func TestGetVPSDiskSizes_OK(t *testing.T) {
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

	ds, err := c.GetVPSDiskSizes()
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

func TestGetVPSDiskSizes_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/disk-sizes", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.GetVPSDiskSizes()
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// Images

func TestGetVPSImages_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/images", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]VPSImage{
			"ubuntu": {Name: "ubuntu-lts", Description: "Ubuntu LTS"},
			"debian": {Name: "debian-13", Description: "Debian 13"},
		})
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	images, err := c.GetVPSImages()
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

func TestGetVPSImages_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/images", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.GetVPSDiskSizes()
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// Zones

func TestGetVPSZones_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/zones", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]Zone{
			"eu":  {Name: "EU", Description: "EU zone", Parents: []string{}},
			"lon": {Name: "London", Description: "London zone", Parents: []string{"eu"}},
		})
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	zones, err := c.GetVPSZones()
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

func TestGetVPSZones_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/zones", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.GetVPSDiskSizes()
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// Hosts

func TestGetVPSHosts_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/hosts", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]VPSHostInfo{
			"one": {Name: "One", Cores: 1, RAM: 64, Disk: VPSHostDiskInfo{SSD: 1, HDD: 1}, FreeRAM: 32, FreeDisk: VPSHostDiskInfo{SSD: 1, HDD: 1}},
			"two": {Name: "Two", Cores: 2, RAM: 64, Disk: VPSHostDiskInfo{SSD: 1, HDD: 1}, FreeRAM: 32, FreeDisk: VPSHostDiskInfo{SSD: 1, HDD: 1}},
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	hosts, err := c.GetVPSHosts()
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

func TestGetVPSHosts_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/hosts", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.GetVPSDiskSizes()
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// Pricing

func TestGetVPSPricing_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/pricing", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(VPSPricing{
			Disk: VPSDiskPrices{
				SSD: VPSDiskPricing{Price: 1, Extent: 1},
				HDD: VPSDiskPricing{Price: 1, Extent: 1},
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

	pricing, err := c.GetVPSPricing()
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

func TestGetVPSPricing_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/hosts", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"not-json}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.GetVPSDiskSizes()
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

// VPS

func TestGetVPS_ByID(t *testing.T) {
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

	v, err := c.GetVPS("my-id")
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

func TestGetVPS_EmptyIdentifier(t *testing.T) {
	t.Parallel()
	c, _ := NewClient(nil, nil)
	_, err := c.GetVPS("")
	if err == nil {
		t.Fatalf("expected error for empty identifier")
	}
}
