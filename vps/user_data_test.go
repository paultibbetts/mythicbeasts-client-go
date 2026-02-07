package vps_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	vpsapi "github.com/paultibbetts/mythicbeasts-client-go/vps"
)

func TestUserData_Create(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%s, want application/json", ct)
		}

		var req vpsapi.NewUserData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req.Name == "" || req.Data == "" {
			t.Fatalf("missing fields: %+v", req)
		}

		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   123,
			"name": req.Name,
			"data": req.Data,
			"size": int64(len(req.Data)),
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	payload := vpsapi.NewUserData{Name: "test", Data: "testing"}
	got, err := c.VPS().CreateUserData(testContext(), payload)
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

func TestUserData_Create_Created(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method=%s, want POST", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%s, want application/json", ct)
		}

		var req vpsapi.NewUserData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req.Name == "" || req.Data == "" {
			t.Fatalf("missing fields: %+v", req)
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":   123,
			"name": req.Name,
			"data": req.Data,
			"size": int64(len(req.Data)),
		})
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	payload := vpsapi.NewUserData{Name: "test", Data: "testing"}
	got, err := c.VPS().CreateUserData(testContext(), payload)
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
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().CreateUserData(testContext(), vpsapi.NewUserData{})
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}

	want := "unexpected status 400: bad payload"
	if err.Error() != want {
		t.Fatalf("err=%q want %q", err.Error(), want)
	}
}

func TestUserData_Create_BadJSON(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{not-json}"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().CreateUserData(testContext(), vpsapi.NewUserData{})
	if err == nil {
		t.Fatalf("user data create expected marshall error")
	}
}

func TestUserData_Get(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"test","data":"123abc", "size":123}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	data, err := c.VPS().GetUserData(testContext(), 1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if data.ID != 1 || data.Name != "test" || !strings.Contains(data.Data, "123abc") || data.Size != 123 {
		t.Fatalf("user data = %+v", data)
	}
}

func TestUserData_Get_StringIDAndSize(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","name":"test","data":"123abc","size":"123"}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	data, err := c.VPS().GetUserData(testContext(), 1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if data.ID != 1 || data.Name != "test" || !strings.Contains(data.Data, "123abc") || data.Size != 123 {
		t.Fatalf("user data = %+v", data)
	}
}

func TestUserData_Get_ContentAlias(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"test","content":"123abc","size":123}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	data, err := c.VPS().GetUserData(testContext(), 1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if data.ID != 1 || data.Name != "test" || !strings.Contains(data.Data, "123abc") || data.Size != 123 {
		t.Fatalf("user data = %+v", data)
	}
}

func TestUserData_Get_DataPreferredOverContent(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1,"name":"test","data":"primary","content":"secondary","size":7}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	data, err := c.VPS().GetUserData(testContext(), 1)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if data.Data != "primary" {
		t.Fatalf("data=%q, want primary", data.Data)
	}
}

func TestUserData_Get_MissingData(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"1","name":"test","size":"123"}`))
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().GetUserData(testContext(), 1)
	if err == nil {
		t.Fatalf("expected malformed response error")
	}

	var malformed *vpsapi.ErrMalformedResponse
	if !errors.As(err, &malformed) {
		t.Fatalf("want ErrMalformedResponse, got %T: %v", err, err)
	}
	if malformed.Field != "data" {
		t.Fatalf("field=%q, want data", malformed.Field)
	}
}

func TestUserData_GetIDFromName(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatal("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]vpsapi.UserDataSnippets{
			"user_data": {
				"12": {
					ID:   12,
					Name: "test1",
					Size: 129,
				},
			},
		})
	})
	mux.HandleFunc("/vps/user-data/12", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(vpsapi.UserData{
			ID:   12,
			Name: "test1",
			Data: "terraform",
			Size: 129,
		})
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	data, err := c.VPS().GetUserDataByName(testContext(), "test1")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if data.ID != 12 {
		t.Fatalf("user data id wrong, got %+v", data)
	}
}

func TestUserData_GetIDFromName_Fails(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatal("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]vpsapi.UserDataSnippets{
			"user_data": {
				"12": {
					ID:   12,
					Name: "test1",
					Size: 129,
				},
			},
		})
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	_, err := c.VPS().GetUserDataByName(testContext(), "test2")
	if err == nil {
		t.Fatalf("expected ErrUserDataNotFound")
	}
	if _, ok := err.(*vpsapi.ErrUserDataNotFound); !ok {
		t.Fatalf("want ErrUserDataNotFound, got %T: %v", err, err)
	}
}

func TestUserData_GetIDFromName_StringID(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatal("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"user_data":{"12":{"id":"12","name":"test1","size":"129"}}}`))
	})
	mux.HandleFunc("/vps/user-data/12", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("want GET")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(vpsapi.UserData{
			ID:   12,
			Name: "test1",
			Data: "terraform",
			Size: 129,
		})
	})
	c, srv := newTestClient(t, mux)
	defer srv.Close()

	data, err := c.VPS().GetUserDataByName(testContext(), "test1")
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if data.ID != 12 {
		t.Fatalf("user data id wrong, got %+v", data)
	}
}

func TestUserData_Update(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data/1", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("method=%s, want PUT", r.Method)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("Content-Type=%s, want application/json", ct)
		}

		var req vpsapi.UpdateUserData
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode req: %v", err)
		}
		if req.Data == "" {
			t.Fatalf("missing data: %+v", req)
		}

		w.WriteHeader(http.StatusOK)
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	err := c.VPS().UpdateUserData(testContext(), 1, vpsapi.UpdateUserData{Data: "terraform"})
	if err != nil {
		t.Fatalf("User Data Update: %v", err)
	}
}

func TestUserData_Update_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/vps/user-data/1", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("bad payload"))
	})

	c, srv := newTestClient(t, mux)
	defer srv.Close()

	err := c.VPS().UpdateUserData(testContext(), 1, vpsapi.UpdateUserData{})
	if err == nil {
		t.Fatalf("expected error for non-200 status")
	}

	want := "unexpected status 400: bad payload"
	if err.Error() != want {
		t.Fatalf("err=%q want %q", err.Error(), want)
	}
}
