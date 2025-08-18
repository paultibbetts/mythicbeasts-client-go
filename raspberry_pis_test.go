package mythicbeasts

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

func TestRaspberryPis_GetModels_OK(t *testing.T) {
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

	models, err := c.GetPiModels()
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

func TestRaspberryPis_GetModels_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/models", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if _, err := c.GetPiModels(); err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

func TestRaspberryPis_GetModels_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/models", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.GetPiModels()
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
		_ = json.NewEncoder(w).Encode(PiOperatingSystems{
			"raspian-buster": "Raspbian Buster",
			"raspian-jessie": "Raspbian Jessie",
		})

	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	images, err := c.GetPiOperatingSystems(3)
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

	if _, err := c.GetPiOperatingSystems(3); err == nil {
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

	_, err := c.GetPiOperatingSystems(3)
	if err == nil {
		t.Fatalf("got err=%v", err)
	}
}

func TestRaspberryPis_GetPis(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(PiServers{
			Servers: []Pi{
				{IP: "12.34.56.78", SSHPort: 22, DiskSize: "1", InitializedKeys: false, Location: "eu", Model: 3, Memory: 1024, CPUSpeed: 1200, NICSpeed: 100},
				{IP: "21.43.65.87", SSHPort: 2222, DiskSize: "2", InitializedKeys: false, Location: "lon", Model: 4, Memory: 2048, CPUSpeed: 1300, NICSpeed: 1000},
			},
		})

	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	pis, err := c.GetPis()
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if len(pis) != 2 {
		t.Fatalf("len(pis)=%d, want 2", len(pis))
	}
	if pis[0].IP != "12.34.56.78" || pis[0].SSHPort != 22 || pis[0].DiskSize != "1" || pis[0].InitializedKeys != false || pis[0].Location != "eu" || pis[0].Model != 3 || pis[0].Memory != 1024 || pis[0].CPUSpeed != 1200 || pis[0].NICSpeed != 100 {
		t.Fatalf("pis[0]=%+v", pis[0])
	}
	if pis[1].IP != "21.43.65.87" || pis[1].SSHPort != 2222 || pis[1].DiskSize != "2" || pis[1].InitializedKeys != false || pis[1].Location != "lon" || pis[1].Model != 4 || pis[1].Memory != 2048 || pis[1].CPUSpeed != 1300 || pis[1].NICSpeed != 1000 {
		t.Fatalf("pis[1]=%+v", pis[1])
	}
}

func TestRaspberryPis_GetPis_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if _, err := c.GetPis(); err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

func TestRaspberryPis_GetPis_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.GetPis()
	if err == nil {
		t.Fatalf("got err=%v", err)
	}
}

func TestRaspberryPis_GetPi(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method=%s, want GET", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(Pi{
			IP: "12.34.56.78", SSHPort: 22, DiskSize: "1", InitializedKeys: false, Location: "eu", Model: 3, Memory: 1024, CPUSpeed: 1200, NICSpeed: 100,
		})

	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	pi, err := c.GetPi("1")
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if pi.IP != "12.34.56.78" || pi.SSHPort != 22 || pi.DiskSize != "1" || pi.InitializedKeys != false || pi.Location != "eu" || pi.Model != 3 || pi.Memory != 1024 || pi.CPUSpeed != 1200 || pi.NICSpeed != 100 {
		t.Fatalf("pis[0]=%+v", pi)
	}
}

func TestRaspberryPis_GetPi_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if _, err := c.GetPi("1"); err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

func TestRaspberryPis_GetPi_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/pi/servers/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("down"))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.GetPi("1")
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
			_ = json.NewEncoder(w).Encode(Pi{
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

	got, err := c.CreatePi(id, CreatePiRequest{})
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

	_, err := c.CreatePi("existing", CreatePiRequest{})
	if err == nil {
		t.Fatalf("expected ErrIdentifierConflict")
	}

	if _, ok := err.(*ErrIdentifierConflict); !ok {
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

	_, err := c.CreatePi("x", CreatePiRequest{})
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

	_, err := c.CreatePi("y", CreatePiRequest{})
	if err == nil || !strings.Contains(err.Error(), "unexpected status 400: bad payload") {
		t.Fatalf("expected unexpected status error, got %v", err)
	}
}

func TestRaspberryPis_Delete_EmptyIdentifier(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	if err := c.DeletePi(" "); !errors.Is(err, ErrEmptyIdentifier) {
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

	if err := c.DeletePi("test"); err != nil {
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

	if err := c.DeletePi("missing"); err != nil {
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

	err := c.DeletePi("bad")
	if err == nil || !strings.Contains(err.Error(), "unexpected status 400: bad request") {
		t.Fatalf("want unexpected status 400, got %v", err)
	}
}

func TestRaspBerryPis_Delete_NetworkError(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	c, srv := newTestClient(t, mux)
	srv.Close()

	if err := c.DeletePi("test"); err == nil {
		t.Fatalf("expected network error, got nil")
	}
}
