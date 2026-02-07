package vps_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

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

func TestGet_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"error":"Server does not exist or access denied"}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().Get(testContext(), "my-id")
	if err == nil {
		t.Fatalf("expected unexpected status error")
	}
	if got, want := err.Error(), `unexpected status 403: {"error":"Server does not exist or access denied"}`; got != want {
		t.Fatalf("err=%q, want %q", got, want)
	}
}

func TestCreateRequest_Marshal_OmitsUnsetOptionalFields(t *testing.T) {
	t.Parallel()

	req := vpsapi.CreateRequest{
		Product:  "VPSX4",
		DiskSize: 10240,
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["product"] != "VPSX4" {
		t.Fatalf("product=%v, want VPSX4", got["product"])
	}
	if got["disk_size"] != float64(10240) {
		t.Fatalf("disk_size=%v, want 10240", got["disk_size"])
	}

	for _, field := range []string{"vnc", "image", "ssh_keys", "ipv4", "tablet"} {
		if _, ok := got[field]; ok {
			t.Fatalf("field %q should be omitted, body=%s", field, string(body))
		}
	}
}

func TestCreateRequest_Marshal_IncludesExplicitOptionalFields(t *testing.T) {
	t.Parallel()

	req := vpsapi.CreateRequest{
		Product:  "VPSX4",
		DiskSize: 10240,
		IPv4:     true,
		Image:    "cloudinit-ubuntu-noble.raw.gz",
		SSHKeys:  "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIC5cSqQNVmTIWz9901r8HB+DiwmnFYRWYXChyqigkzAA",
		Tablet:   vpsapi.Bool(false),
	}

	body, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["ipv4"] != true {
		t.Fatalf("ipv4=%v, want true", got["ipv4"])
	}
	if got["image"] != "cloudinit-ubuntu-noble.raw.gz" {
		t.Fatalf("image=%v", got["image"])
	}
	if got["ssh_keys"] == "" {
		t.Fatalf("ssh_keys should be present")
	}
	if got["tablet"] != false {
		t.Fatalf("tablet=%v, want false", got["tablet"])
	}
}

func TestUpdate(t *testing.T) {
	t.Parallel()

	specs := vpsapi.NewUpdateSpecs()
	specs.SetDiskSize(20480)
	specs.SetExtraCores(4)
	specs.SetExtraRAM(2048)

	payload := vpsapi.NewUpdateRequest()
	payload.SetProduct("VPSX16")
	payload.SetSpecs(specs)
	payload.SetName("web-server-01")
	payload.SetBootDevice("cdrom")
	payload.SetISOImage("debian-10.10.0-amd64-netinst")
	payload.SetCPUMode("performance")
	payload.SetNetDevice("virtio")
	payload.SetDiskBus("virtio")
	payload.SetTablet(true)

	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method=%s, want PATCH", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%s, want application/json", ct)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req["product"] != "VPSX16" {
			t.Fatalf("product=%v, want VPSX16", req["product"])
		}
		specsMap, ok := req["specs"].(map[string]any)
		if !ok {
			t.Fatalf("specs=%v, want object", req["specs"])
		}
		if specsMap["disk_size"] != float64(20480) {
			t.Fatalf("specs.disk_size=%v, want 20480", specsMap["disk_size"])
		}
		if req["tablet"] != true {
			t.Fatalf("tablet=%v, want true", req["tablet"])
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(vpsapi.UpdateResponse{Message: "Operation successful"})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	resp, err := c.VPS().Update(testContext(), "my-id", payload)
	if err != nil {
		t.Fatalf("update err: %v", err)
	}
	if resp.Message != "Operation successful" {
		t.Fatalf("message=%q, want %q", resp.Message, "Operation successful")
	}
}

func TestUpdate_ClearNullableFields(t *testing.T) {
	t.Parallel()

	payload := vpsapi.NewUpdateRequest()
	payload.ClearName()
	payload.ClearISOImage()

	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("method=%s, want PATCH", r.Method)
		}

		var req map[string]any
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}

		if v, ok := req["name"]; !ok || v != nil {
			t.Fatalf("name=%v (exists=%v), want null", v, ok)
		}
		if v, ok := req["iso_image"]; !ok || v != nil {
			t.Fatalf("iso_image=%v (exists=%v), want null", v, ok)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(vpsapi.UpdateResponse{Message: "Operation successful"})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	resp, err := c.VPS().Update(testContext(), "my-id", payload)
	if err != nil {
		t.Fatalf("update err: %v", err)
	}
	if resp.Message != "Operation successful" {
		t.Fatalf("message=%q, want %q", resp.Message, "Operation successful")
	}
}

func TestUpdate_RequiresPoweredOff(t *testing.T) {
	t.Parallel()

	unset := vpsapi.NewUpdateRequest()
	if unset.RequiresPoweredOff() {
		t.Fatalf("unset update should not require powered off")
	}

	nonPower := vpsapi.NewUpdateRequest()
	nonPower.SetProduct("VPSX16")
	if nonPower.RequiresPoweredOff() {
		t.Fatalf("product-only update should not require powered off")
	}

	powerFields := []vpsapi.UpdateRequest{
		func() vpsapi.UpdateRequest { r := vpsapi.NewUpdateRequest(); r.SetBootDevice("hd"); return r }(),
		func() vpsapi.UpdateRequest { r := vpsapi.NewUpdateRequest(); r.SetISOImage("debian-12"); return r }(),
		func() vpsapi.UpdateRequest { r := vpsapi.NewUpdateRequest(); r.ClearISOImage(); return r }(),
		func() vpsapi.UpdateRequest { r := vpsapi.NewUpdateRequest(); r.SetCPUMode("performance"); return r }(),
		func() vpsapi.UpdateRequest { r := vpsapi.NewUpdateRequest(); r.SetNetDevice("virtio"); return r }(),
		func() vpsapi.UpdateRequest { r := vpsapi.NewUpdateRequest(); r.SetDiskBus("virtio"); return r }(),
		func() vpsapi.UpdateRequest { r := vpsapi.NewUpdateRequest(); r.SetTablet(true); return r }(),
	}

	for i, req := range powerFields {
		if !req.RequiresPoweredOff() {
			t.Fatalf("expected update %d to require powered off", i)
		}
	}
}

func TestUpdate_EmptyIdentifier(t *testing.T) {
	t.Parallel()
	c, _ := mythicbeasts.NewClient("", "")

	_, err := c.VPS().Update(testContext(), " ", vpsapi.UpdateRequest{})
	if !errors.Is(err, vpsapi.ErrEmptyIdentifier) {
		t.Fatalf("want ErrEmptyIdentifier, got %v", err)
	}
}

func TestUpdate_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().Update(testContext(), "my-id", vpsapi.UpdateRequest{})
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}

	want := "unexpected status 400: bad payload"
	if err.Error() != want {
		t.Fatalf("err=%q want %q", err.Error(), want)
	}
}

func TestReboot(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id/reboot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(vpsapi.RebootResponse{Message: "Operation successful"})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	resp, err := c.VPS().Reboot(testContext(), "my-id")
	if err != nil {
		t.Fatalf("reboot err: %v", err)
	}
	if resp.Message != "Operation successful" {
		t.Fatalf("message=%q, want %q", resp.Message, "Operation successful")
	}
}

func TestRebootWithGrace(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id/reboot", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(vpsapi.RebootResponse{Message: "Operation successful"})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	resp, err := c.VPS().RebootWithGrace(testContext(), "my-id", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("reboot with grace err: %v", err)
	}
	if resp.Message != "Operation successful" {
		t.Fatalf("message=%q, want %q", resp.Message, "Operation successful")
	}
}

func TestRebootWithGrace_ContextCanceled(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id/reboot", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(vpsapi.RebootResponse{Message: "Operation successful"})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	ctx, cancel := context.WithCancel(testContext())
	cancel()

	_, err := c.VPS().RebootWithGrace(ctx, "my-id", 10*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context canceled, got %v", err)
	}
}

func TestSetPower(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id/power", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method=%s, want PUT", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%s, want application/json", ct)
		}

		var req vpsapi.PowerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req.Power != vpsapi.PowerActionShutdown {
			t.Fatalf("power=%q, want %q", req.Power, vpsapi.PowerActionShutdown)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(vpsapi.PowerResponse{Message: "Operation successful"})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	resp, err := c.VPS().SetPower(testContext(), "my-id", vpsapi.PowerActionShutdown)
	if err != nil {
		t.Fatalf("set power err: %v", err)
	}
	if resp.Message != "Operation successful" {
		t.Fatalf("message=%q, want %q", resp.Message, "Operation successful")
	}
}

func TestShutdownWithGrace(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id/power", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method=%s, want PUT", r.Method)
		}

		var req vpsapi.PowerRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req.Power != vpsapi.PowerActionShutdown {
			t.Fatalf("power=%q, want %q", req.Power, vpsapi.PowerActionShutdown)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(vpsapi.PowerResponse{Message: "Operation successful"})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	resp, err := c.VPS().ShutdownWithGrace(testContext(), "my-id", 1*time.Millisecond)
	if err != nil {
		t.Fatalf("shutdown with grace err: %v", err)
	}
	if resp.Message != "Operation successful" {
		t.Fatalf("message=%q, want %q", resp.Message, "Operation successful")
	}
}

func TestShutdownWithGrace_ContextCanceled(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/servers/my-id/power", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(vpsapi.PowerResponse{Message: "Operation successful"})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	ctx, cancel := context.WithCancel(testContext())
	cancel()

	_, err := c.VPS().ShutdownWithGrace(ctx, "my-id", 10*time.Millisecond)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context canceled, got %v", err)
	}
}

func TestSetPower_InvalidAction(t *testing.T) {
	t.Parallel()
	c, _ := mythicbeasts.NewClient("", "")

	_, err := c.VPS().SetPower(testContext(), "my-id", vpsapi.PowerAction("invalid"))
	if err == nil || !strings.Contains(err.Error(), `invalid power action "invalid"`) {
		t.Fatalf("want invalid power action error, got %v", err)
	}
}
