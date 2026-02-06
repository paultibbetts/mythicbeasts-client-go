package mythicbeasts

import (
	"context"
	"encoding/base64"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBasicAuth(t *testing.T) {
	t.Parallel()
	got := basicAuth("user", "pass")
	want := base64.StdEncoding.EncodeToString([]byte("user:pass"))
	if got != want {
		t.Fatalf("basicAuth = %q, want %q", got, want)
	}
}

func TestSignIn_Success(t *testing.T) {
	t.Parallel()

	key := "myKeyID"
	secret := "mySecret"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/login" {
			t.Fatalf("path = %s, want /login", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/x-www-form-urlencoded" {
			t.Fatalf("Content-Type: %q, want application/x-www-form-urlencoded", ct)
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Basic") {
			t.Fatalf("Authorization = %q, want Basic ...", auth)
		}
		dec, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
		if err != nil {
			t.Fatalf("bad base64 in Authorization: %v", err)
		}
		if string(dec) != key+":"+secret {
			t.Fatalf("decoded creds = %q, want %q", string(dec), key+":"+secret)
		}

		b, _ := io.ReadAll(r.Body)
		if string(b) != "grant_type=client_credentials" {
			t.Fatalf("body = %q, want grant_type=client_credentials", string(b))
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"XYZ","token_type":"bearer"}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClient("", "")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	c.AuthURL = srv.URL
	c.Auth = AuthStruct{KeyID: key, Secret: secret}
	c.Token = ""

	ar, err := c.signIn(context.Background())
	if err != nil {
		t.Fatalf("signIn error: %v", err)
	}
	if ar.AccessToken != "XYZ" || strings.ToLower(ar.TokenType) != "bearer" {
		t.Fatalf("got token=%q type=%q; want token=XYZ type=bearer", ar.AccessToken, ar.TokenType)
	}
}

func TestSignIn_MissingCreds(t *testing.T) {
	t.Parallel()
	c, _ := NewClient("", "")
	c.AuthURL = "http://example.com"
	_, err := c.signIn(context.Background())
	if err == nil || !strings.Contains(err.Error(), "define keyid and secret") {
		t.Fatalf("expected missing creds error, got %v", err)
	}
}

func TestSignIn_ServerBadJSONOrStatus(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("nope"))
	}))
	t.Cleanup(srv.Close)

	c, _ := NewClient("", "")
	c.AuthURL = srv.URL
	c.Auth = AuthStruct{KeyID: "id", Secret: "sec"}

	_, err := c.signIn(context.Background())
	if err == nil {
		t.Fatalf("expected error for non-200 response")
	}
	want := "auth failed: status 401: nope"
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}
