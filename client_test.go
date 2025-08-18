package mythicbeasts

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewRequest_ResolvesRelativeAgainstHost(t *testing.T) {
	t.Parallel()
	c, _ := NewClient(nil, nil)
	c.HostURL = "https://example.com/base"

	req, err := c.NewRequest(http.MethodGet, "/vps/disk-sizes", nil)
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}
	if got, want := req.URL.String(), "https://example.com/base/vps/disk-sizes"; got != want {
		t.Fatalf("url = %s, want %s", got, want)
	}
}

func TestNewRequest_KeepsAbsoluteURL(t *testing.T) {
	t.Parallel()
	c, _ := NewClient(nil, nil)

	req, err := c.NewRequest(http.MethodGet, "https://api.example.com/x", nil)
	if err != nil {
		t.Fatalf("NewRequest error: %v", err)
	}

	if req.URL.Host != "api.example.com" {
		t.Fatalf("expected absolute host, got %s", req.URL.Host)
	}
}

func TestNewRequest_InvalidHostURL(t *testing.T) {
	t.Parallel()
	c, _ := NewClient(nil, nil)
	c.HostURL = ":// bad base"
	_, err := c.NewRequest(http.MethodGet, "/anything", nil)
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

	c, _ := NewClient(nil, nil)
	c.Token = "tok"
	req, _ := c.NewRequest(http.MethodGet, s.URL, nil)

	res, err := c.do(req)
	if err != nil {
		t.Fatalf("do error: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", res.StatusCode)
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

	c, _ := NewClient(nil, nil)
	res, err := c.get(s.URL)
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

	c, _ := NewClient(nil, nil)
	if err := c.delete(s.URL); err != nil {
		t.Fatalf("delete error: %v", err)
	}
}

func TestBody_ReadsAllAndCloses(t *testing.T) {
	t.Parallel()
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("hello"))

	}))
	t.Cleanup(s.Close)

	c, _ := NewClient(nil, nil)
	res, err := c.get(s.URL)
	if err != nil {
		t.Fatalf("get error: %v", err)
	}
	b, err := c.body(res)
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

	c, _ := NewClient(nil, nil)
	c.PollInterval = time.Millisecond

	url, err := c.pollProvisioning(s.URL, 2*time.Second, "id", func(map[string]any, string) (string, bool) {
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

	c, _ := NewClient(nil, nil)
	c.PollInterval = time.Millisecond

	_, err := c.pollProvisioning(s.URL, time.Second, "id", func(map[string]any, string) (string, bool) {
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

	c, _ := NewClient(nil, nil)
	c.PollInterval = time.Millisecond

	url, err := c.pollProvisioning(s.URL, time.Second, "id", func(map[string]any, string) (string, bool) {
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

	c, _ := NewClient(nil, nil)
	c.PollInterval = time.Millisecond

	checker := func(data map[string]any, id string) (string, bool) {
		if data["state"] == "done" {
			return data["url"].(string), true
		}
		return "", false
	}

	url, err := c.pollProvisioning(s.URL, time.Second, "id", checker)
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

	c, _ := NewClient(nil, nil)
	c.PollInterval = 5 * time.Millisecond

	_, err := c.pollProvisioning(s.URL, 20*time.Millisecond, "id", func(map[string]any, string) (string, bool) {
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

	c, _ := NewClient(nil, nil)
	c.PollInterval = time.Millisecond

	_, err := c.pollProvisioning(s.URL, time.Second, "id", func(map[string]any, string) (string, bool) {
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

	c, _ := NewClient(nil, nil)
	c.PollInterval = time.Millisecond

	_, err := c.pollProvisioning(s.URL, time.Second, "id", func(map[string]any, string) (string, bool) {
		return "", false
	})

	if err == nil || !strings.Contains(err.Error(), "unexpected status while polling: 418") {
		t.Fatalf("unexpected error: %v", err)
	}
}
