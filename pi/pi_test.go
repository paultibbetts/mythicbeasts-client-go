package pi_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/paultibbetts/mythicbeasts-client-go"
	piapi "github.com/paultibbetts/mythicbeasts-client-go/pi"
)

func newTestClient(t *testing.T, mux *http.ServeMux) (*mythicbeasts.Client, *httptest.Server) {
	t.Helper()
	srv := httptest.NewServer(mux)
	c, _ := mythicbeasts.NewClient("", "")
	c.Pi().BaseURL = srv.URL
	return c, srv
}

func testContext() context.Context {
	return context.Background()
}

func TestRaspberryPis_ListModels_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/models", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"model": 3, "memory": 2048, "nic_speed": 100, "cpu_speed": 1000},
				{"model": 4, "memory": 4096, "nic_speed": 1000, "cpu_speed": 2000},
			},
		})

	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	models, err := c.Pi().ListModels(testContext())
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(models) != 2 {
		t.Fatalf("len(models)=%d, want 2", len(models))
	}
	if models[0].Model != 3 || models[0].Memory != 2048 || models[0].NICSpeed != 100 || models[0].CPUSpeed != 1000 {
		t.Fatalf("models[0]=%+v", models[0])
	}
	if models[1].Model != 4 || models[1].Memory != 4096 || models[1].NICSpeed != 1000 || models[1].CPUSpeed != 2000 {
		t.Fatalf("models[1]=%+v", models[1])
	}
}

func TestRaspberryPis_ListModels_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/models", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if _, err := c.Pi().ListModels(testContext()); err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

func TestRaspberryPis_ListModels_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/models", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().ListModels(testContext())
	if err == nil || !strings.Contains(err.Error(), "unexpected status: 503, down") {
		t.Fatalf("got err=%v", err)
	}
}

func TestRaspberryPis_GetOperatingSystems_OK(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/images/3", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(piapi.OperatingSystems{
			"raspian-buster": "Raspbian Buster",
			"raspian-jessie": "Raspbian Jessie",
		})

	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	images, err := c.Pi().GetOperatingSystems(testContext(), 3)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(images) != 2 {
		t.Fatalf("len(images)=%d, want 2", len(images))
	}
	if images["raspian-buster"] != "Raspbian Buster" {
		t.Fatalf("images[0]=%+v", images["raspbian-buster"])
	}
	if images["raspian-jessie"] != "Raspbian Jessie" {
		t.Fatalf("images[0]=%+v", images["raspbian-jessie"])
	}
}

func TestRaspberryPis_GetOperatingSystems_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/images/3", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if _, err := c.Pi().GetOperatingSystems(testContext(), 3); err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

func TestRaspberryPis_GetOperatingSystems_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/images/3", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().GetOperatingSystems(testContext(), 3)
	if err == nil {
		t.Fatalf("got err=%v", err)
	}
}

func TestRaspberryPis_List(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(piapi.Servers{
			Servers: []piapi.Server{
				{IP: "12.34.56.78", SSHPort: 22, DiskSize: "1", InitializedKeys: false, Location: "eu", Model: 3, Memory: 1024, CPUSpeed: 1200, NICSpeed: 100},
				{IP: "21.43.65.87", SSHPort: 2222, DiskSize: "2", InitializedKeys: false, Location: "lon", Model: 4, Memory: 2048, CPUSpeed: 1300, NICSpeed: 1000},
			},
		})

	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	pis, err := c.Pi().List(testContext())
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(pis) != 2 {
		t.Fatalf("len(pis)=%d, want 2", len(pis))
	}
	if pis[0].IP != "12.34.56.78" {
		t.Fatalf("pis[0]=%+v", pis[0])
	}
	if pis[1].IP != "21.43.65.87" {
		t.Fatalf("pis[1]=%+v", pis[1])
	}
}

func TestRaspberryPis_List_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if _, err := c.Pi().List(testContext()); err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

func TestRaspberryPis_List_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().List(testContext())
	if err == nil {
		t.Fatalf("got err=%v", err)
	}
}

func TestRaspberryPis_Get(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(piapi.Server{
			IP: "12.34.56.78", SSHPort: 22, DiskSize: "1", InitializedKeys: false, Location: "eu", Model: 3, Memory: 1024, CPUSpeed: 1200, NICSpeed: 100,
		})

	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	pi, err := c.Pi().Get(testContext(), "1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if pi.IP != "12.34.56.78" || pi.SSHPort != 22 || pi.DiskSize != "1" || pi.InitializedKeys != false || pi.Location != "eu" || pi.Model != 3 || pi.Memory != 1024 || pi.CPUSpeed != 1200 || pi.NICSpeed != 100 {
		t.Fatalf("pis[0]=%+v", pi)
	}
}

func TestRaspberryPis_Get_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if _, err := c.Pi().Get(testContext(), "1"); err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

func TestRaspberryPis_Get_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().Get(testContext(), "1")
	if err == nil {
		t.Fatalf("got err=%v", err)
	}
}

func TestRaspberryPis_Create_Success(t *testing.T) {
	t.Parallel()
	const id = "test-pi"
	const pollPath = "/poll/test"

	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/"+id, func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			if ct := r.Header.Get("Content-Type"); ct != "application/json" {
				t.Fatalf("Content-Type=%q, want application/json", ct)
			}
			w.Header().Set("Location", pollPath)
			w.WriteHeader(http.StatusAccepted)
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode(piapi.Server{
				IP: "12.34.56.78", SSHPort: 22, DiskSize: "1", InitializedKeys: false, Location: "eu", Model: 3, Memory: 1024, CPUSpeed: 1200, NICSpeed: 100,
			})
		default:
			t.Fatalf("unexpected method %s", r.Method)
		}
	})

	mux.HandleFunc(pollPath, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Location", "/pi/servers/"+id)
		w.WriteHeader(http.StatusOK)
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	got, err := c.Pi().Create(testContext(), id, piapi.CreateRequest{})
	if err != nil {
		t.Fatalf("create pi error: %v", err)
	}
	if got == nil || got.IP != "12.34.56.78" || got.SSHPort != 22 || got.DiskSize != "1" || got.InitializedKeys != false || got.Location != "eu" || got.Model != 3 || got.Memory != 1024 || got.CPUSpeed != 1200 || got.NICSpeed != 100 {
		t.Fatalf("got=%+v", got)
	}
}

func TestRaspberryPis_Create_Conflict(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/existing", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("want POST")
		}
		w.WriteHeader(http.StatusConflict)
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().Create(testContext(), "existing", piapi.CreateRequest{})
	if err == nil {
		t.Fatalf("expected ErrIdentifierConflict")
	}

	if _, ok := err.(*piapi.ErrIdentifierConflict); !ok {
		t.Fatalf("want ErrIdentifierConflict, got %T: %v", err, err)
	}
}

func TestRaspberryPis_Create_MissingLocation(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/x", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		// no location header
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().Create(testContext(), "x", piapi.CreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "missing header location") {
		t.Fatalf("expected missing header location, got %v", err)
	}
}

func TestRaspberryPis_Create_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/y", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().Create(testContext(), "y", piapi.CreateRequest{})
	if err == nil || !strings.Contains(err.Error(), "unexpected status 400: bad payload") {
		t.Fatalf("expected unexpected status error, got %v", err)
	}
}

func TestRaspberryPis_UpdateSSHKey_Success(t *testing.T) {
	t.Parallel()
	const id = "1"
	const key = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC user@example.com"

	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/"+id+"/ssh-key", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method=%s, want PUT", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%q, want application/json", ct)
		}

		var req piapi.UpdateSSHKeyRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req.SSHKey != key {
			t.Fatalf("ssh_key=%q, want %q", req.SSHKey, key)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(piapi.UpdateSSHKeyResponse{SSHKey: key})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	got, err := c.Pi().UpdateSSHKey(testContext(), id, piapi.UpdateSSHKeyRequest{SSHKey: key})
	if err != nil {
		t.Fatalf("update ssh key: %v", err)
	}

	if got.SSHKey != key {
		t.Fatalf("ssh_key=%q, want %q", got.SSHKey, key)
	}
}

func TestRaspberryPis_UpdateSSHKey_EmptyIdentifier(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().UpdateSSHKey(testContext(), " ", piapi.UpdateSSHKeyRequest{SSHKey: "ssh-rsa AAAAB..."})
	if !errors.Is(err, piapi.ErrEmptyIdentifier) {
		t.Fatalf("want ErrEmptyIdentifier, got %v", err)
	}
}

func TestRaspberryPis_UpdateSSHKey_EmptyKey(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().UpdateSSHKey(testContext(), "1", piapi.UpdateSSHKeyRequest{})
	if err == nil || !strings.Contains(err.Error(), "ssh key is required") {
		t.Fatalf("want ssh key required error, got %v", err)
	}
}

func TestRaspberryPis_UpdateSSHKey_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/1/ssh-key", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{"error":"Server is not fully provisioned"}`))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.Pi().UpdateSSHKey(testContext(), "1", piapi.UpdateSSHKeyRequest{SSHKey: "ssh-rsa AAAAB..."})
	if err == nil || !strings.Contains(err.Error(), `unexpected status 409: {"error":"Server is not fully provisioned"}`) {
		t.Fatalf("want unexpected status 409 error, got %v", err)
	}
}

func TestRaspberryPis_Delete_EmptyIdentifier(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := c.Pi().Delete(testContext(), " "); !errors.Is(err, piapi.ErrEmptyIdentifier) {
		t.Fatalf("want ErrEmptyIdentifier, got %v", err)
	}
}

func TestRaspberryPis_Delete_204(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/test", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("want DELETE")
		}
		w.WriteHeader(http.StatusNoContent)
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := c.Pi().Delete(testContext(), "test"); err != nil {
		t.Fatalf("deletePi err: %v", err)
	}
}

func TestRaspberryPis_Delete_404(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/missing", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := c.Pi().Delete(testContext(), "missing"); err != nil {
		t.Fatalf("expected nil err despite 404, got %v", err)
	}
}

func TestRaspberryPis_Delete_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/bad", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad request"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	err := c.Pi().Delete(testContext(), "bad")
	if err == nil || !strings.Contains(err.Error(), "unexpected status 400: bad request") {
		t.Fatalf("want unexpected status 400, got %v", err)
	}
}

func TestRaspBerryPis_Delete_NetworkError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	c, srv := newTestClient(t, mux)
	srv.Close()

	if err := c.Pi().Delete(testContext(), "test"); err == nil {
		t.Fatalf("expected network error, got nil")
	}
}
