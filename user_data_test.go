package mythicbeasts

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestUserData_Create(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%s, want application/json", ct)
		}

		var req NewUserData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req.Name == "" || req.Data == "" {
			t.Fatalf("missing fields: %+v", req)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":      123,
			"name":    req.Name,
			"content": req.Data,
			"size":    int64(len(req.Data)),
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	payload := NewUserData{Name: "test", Data: "testing"}
	got, err := c.CreateUserData(payload)
	if err != nil {
		t.Fatalf("User Data Create: %v", err)
	}

	if got.ID != 123 {
		t.Fatalf("ID=%d, want 123", got.ID)
	}
	if got.Name != payload.Name {
		t.Fatalf("Name=%q want %q", got.Name, payload.Name)
	}
	if got.Data != payload.Data {
		t.Fatalf("Data=%q want %q", got.Data, payload.Data)
	}
	wantedSize := int64(len(payload.Data))
	if got.Size != wantedSize {
		t.Fatalf("Size=%d, want %d", got.Size, wantedSize)
	}
}

func TestUserData_Create_UnexpectedStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.CreateUserData(NewUserData{})
	if err == nil {
		t.Fatalf("expected error for non-201 status")
	}

	want := "unexpected status 400: bad payload"
	if err.Error() != want {
		t.Fatalf("err=%q want %q", err.Error(), want)
	}
}

func TestUserData_Create_BadJSON(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("{not-json}"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.CreateUserData(NewUserData{})
	if err == nil {
		t.Fatalf("user data create expected marshall error")
	}
}

func TestUserData_Get(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		_, _ = w.Write([]byte(`{"id":1,"name":"test","content":"123abc", "size":123}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	data, err := c.GetUserData(1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if data.ID != 1 || data.Name != "test" || !strings.Contains(data.Data, "123abc") || data.Size != 123 {
		t.Fatalf("user data = %+v", data)
	}
}
