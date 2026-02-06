package mythicbeasts

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRequest_ResolvesRelativeAgainstHost(t *testing.T) {
	t.Parallel()
	c, _ := NewClient("", "")
	baseURL := "https://example.com/base"

	req, err := c.NewRequest(context.Background(), http.MethodGet, baseURL, "/vps/disk-sizes", nil)
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}
	if got, want := req.URL.String(), "https://example.com/base/vps/disk-sizes"; got != want {
		t.Fatalf("url = %s, want %s", got, want)
	}
}

func TestNewRequest_KeepsAbsoluteURL(t *testing.T) {
	t.Parallel()
	c, _ := NewClient("", "")

	req, err := c.NewRequest(context.Background(), http.MethodGet, "https://example.com/base", "https://api.example.com/x", nil)
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}

	if req.URL.Host != "api.example.com" {
		t.Fatalf("expected absolute host, got %s", req.URL.Host)
	}
}

func TestNewRequest_InvalidHostURL(t *testing.T) {
	t.Parallel()
	c, _ := NewClient("", "")
	_, err := c.NewRequest(context.Background(), http.MethodGet, ":// bad base", "/anything", nil)
	if err == nil {
		t.Fatalf("expected error for invalid host url")
	}
}

func TestDo_AddsBearerToken(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tok" {
			t.Fatalf("Authorization = %q, want %q", got, "Bearer tok")
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	c.Token = "tok"
	req, _ := c.NewRequest(context.Background(), http.MethodGet, s.URL, "/", nil)

	res, err := c.Do(req)
	if err != nil {
		t.Fatalf("do error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
}

func TestDo_EnsureTokenConcurrentSingleSignIn(t *testing.T) {
	t.Parallel()

	var signInCalls int32
	loginStarted := make(chan struct{})
	allowLogin := make(chan struct{})

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/login":
			atomic.AddInt32(&signInCalls, 1)
			select {
			case <-loginStarted:
			default:
				close(loginStarted)
			}
			<-allowLogin
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"access_token":"XYZ","token_type":"bearer"}`))
		case "/resource":
			if got := r.Header.Get("Authorization"); got != "Bearer XYZ" {
				t.Fatalf("Authorization = %q, want %q", got, "Bearer XYZ")
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	t.Cleanup(s.Close)

	c, err := NewClient("keyid", "secret")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	c.AuthURL = s.URL
	c.HTTPClient = s.Client()

	const callers = 5
	errs := make(chan error, callers)
	for i := 0; i < callers; i++ {
		go func() {
			req, err := c.NewRequest(context.Background(), http.MethodGet, s.URL, "/resource", nil)
			if err != nil {
				errs <- err
				return
			}
			_, err = c.Do(req)
			errs <- err
		}()
	}

	<-loginStarted
	close(allowLogin)

	for i := 0; i < callers; i++ {
		if err := <-errs; err != nil {
			t.Fatalf("request error: %v", err)
		}
	}

	if got := atomic.LoadInt32(&signInCalls); got != 1 {
		t.Fatalf("signIn calls = %d, want 1", got)
	}
	if c.Token != "XYZ" {
		t.Fatalf("token = %q, want %q", c.Token, "XYZ")
	}
}

func TestEnsureToken_RefreshesWhenExpired(t *testing.T) {
	t.Parallel()

	var signInCalls int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		atomic.AddInt32(&signInCalls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"NEW","token_type":"bearer","expires_in":30}`))
	}))
	t.Cleanup(s.Close)

	c, err := NewClient("keyid", "secret")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	c.AuthURL = s.URL
	c.Token = "OLD"
	c.tokenExpiresIn = 30 * time.Second
	c.tokenLastUsedAt = time.Now().Add(-25 * time.Second)

	token, err := c.ensureToken(context.Background())
	if err != nil {
		t.Fatalf("ensureToken error: %v", err)
	}
	if token != "NEW" {
		t.Fatalf("token = %q, want %q", token, "NEW")
	}
	if got := atomic.LoadInt32(&signInCalls); got != 1 {
		t.Fatalf("signIn calls = %d, want 1", got)
	}
}

func TestEnsureToken_NoRefreshWhenFresh(t *testing.T) {
	t.Parallel()

	var signInCalls int32
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/login" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		atomic.AddInt32(&signInCalls, 1)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"access_token":"NEW","token_type":"bearer","expires_in":30}`))
	}))
	t.Cleanup(s.Close)

	c, err := NewClient("keyid", "secret")
	if err != nil {
		t.Fatalf("NewClient error: %v", err)
	}
	c.AuthURL = s.URL
	c.Token = "OLD"
	c.tokenExpiresIn = 30 * time.Second
	c.tokenLastUsedAt = time.Now()

	token, err := c.ensureToken(context.Background())
	if err != nil {
		t.Fatalf("ensureToken error: %v", err)
	}
	if token != "OLD" {
		t.Fatalf("token = %q, want %q", token, "OLD")
	}
	if got := atomic.LoadInt32(&signInCalls); got != 0 {
		t.Fatalf("signIn calls = %d, want 0", got)
	}
}

func TestGet(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("method = %s, want GET", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	res, err := c.Get(context.Background(), s.URL, "/")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("method = %s, want DELETE", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	if err := c.Delete(context.Background(), s.URL, "/"); err != nil {
		t.Fatalf("delete error: %v", err)
	}
}

func TestBody_ReadsAllAndCloses(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))

	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	res, err := c.Get(context.Background(), s.URL, "/")
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	b, err := c.Body(res)
	if err != nil {
		t.Fatalf("body error: %v", err)
	}
	if string(b) != "hello" {
		t.Fatalf("body = %q, want %q", string(b), "hello")
	}
}

type step struct {
	status  int
	headers map[string]string
	body    string
}

func scriptHandler(steps []step) http.HandlerFunc {
	i := 0
	return func(w http.ResponseWriter, r *http.Request) {
		if i >= len(steps) {
			i = len(steps) - 1
		}
		st := steps[i]
		i++
		for k, v := range st.headers {
			w.Header().Set(k, v)
		}
		w.WriteHeader(st.status)
		_, _ = w.Write([]byte(st.body))
	}
}

func TestPoll_SeeOtherReturnsLocation(t *testing.T) {
	t.Parallel()
	want := "https://done.example.com/vps/123"
	s := httptest.NewServer(scriptHandler([]step{
		{status: http.StatusSeeOther, headers: map[string]string{"Location": want}},
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	c.PollInterval = time.Millisecond

	url, err := c.PollProvisioning(context.Background(), s.URL, s.URL, 2*time.Second, "id", func(map[string]any, string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("pollProvisioning error: %v", err)
	}
	if url != want {
		t.Fatalf("url = %s, want %s", url, want)
	}
}

func TestPoll_InternalServerError(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(scriptHandler([]step{
		{status: http.StatusInternalServerError, body: "boom"},
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	c.PollInterval = time.Millisecond

	_, err := c.PollProvisioning(context.Background(), s.URL, s.URL, time.Second, "id", func(map[string]any, string) (string, bool) {
		return "", false
	})
	if err == nil || err.Error() != "provisioning failed: boom" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPoll_AcceptedWithLocation(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(scriptHandler([]step{
		{status: http.StatusAccepted, headers: map[string]string{"Location": "/ready/123"}},
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	c.PollInterval = time.Millisecond

	url, err := c.PollProvisioning(context.Background(), s.URL, s.URL, time.Second, "id", func(map[string]any, string) (string, bool) {
		return "", false
	})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if url != "/ready/123" {
		t.Fatalf("got %s", url)
	}
}

func TestPoll_OKWithCompletionChecker(t *testing.T) {
	t.Parallel()
	want := "https://srv/ok"
	payload := map[string]any{"state": "done", "url": want}
	b, _ := json.Marshal(payload)

	s := httptest.NewServer(scriptHandler([]step{
		{status: http.StatusOK, body: string(b)},
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	c.PollInterval = time.Millisecond

	checker := func(data map[string]any, id string) (string, bool) {
		if data["state"] == "done" {
			return data["url"].(string), true
		}
		return "", false
	}

	url, err := c.PollProvisioning(context.Background(), s.URL, s.URL, time.Second, "id", checker)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if url != want {
		t.Fatalf("url = %s", url)
	}
}

func TestPoll_Timeout(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(scriptHandler([]step{
		{status: http.StatusOK, body: `{"state":"pending"}`},
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	c.PollInterval = 5 * time.Millisecond

	_, err := c.PollProvisioning(context.Background(), s.URL, s.URL, 20*time.Millisecond, "id", func(map[string]any, string) (string, bool) {
		return "", false
	})
	if err == nil || err.Error() != "timed out while provisioning" {
		t.Fatalf("expected timeout, got: %v", err)
	}
}

func TestPoll_OKBadJSON(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(scriptHandler([]step{
		{status: http.StatusOK, body: "{not-json"},
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	c.PollInterval = time.Millisecond

	_, err := c.PollProvisioning(context.Background(), s.URL, s.URL, time.Second, "id", func(map[string]any, string) (string, bool) {
		return "", false
	})
	if err == nil {
		t.Fatalf("expected unmarshall error")
	}
}

func TestPoll_UnexpectedStatus(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(scriptHandler([]step{
		{status: http.StatusTeapot},
	}))
	t.Cleanup(s.Close)

	c, _ := NewClient("", "")
	c.PollInterval = time.Millisecond

	_, err := c.PollProvisioning(context.Background(), s.URL, s.URL, time.Second, "id", func(map[string]any, string) (string, bool) {
		return "", false
	})

	if err == nil || !strings.Contains(err.Error(), "unexpected status while polling: 418") {
		t.Fatalf("unexpected error: %v", err)
	}
}
