package mythicbeasts

import (
	"testing"

	piapi "github.com/paultibbetts/mythicbeasts-client-go/pi"
	proxyapi "github.com/paultibbetts/mythicbeasts-client-go/proxy"
	vpsapi "github.com/paultibbetts/mythicbeasts-client-go/vps"
)

func TestServiceBaseURLs(t *testing.T) {
	t.Parallel()
	c, _ := NewClient("", "")

	if got, want := c.Pi().BaseURL, piapi.BaseURL; got != want {
		t.Fatalf("Pi BaseURL=%q want %q", got, want)
	}
	if got, want := c.VPS().BaseURL, vpsapi.BaseURL; got != want {
		t.Fatalf("VPS BaseURL=%q want %q", got, want)
	}
	if got, want := c.Proxy().BaseURL, proxyapi.BaseURL; got != want {
		t.Fatalf("Proxy BaseURL=%q want %q", got, want)
	}
	if c.Pi().BaseURL != c.VPS().BaseURL {
		t.Fatalf("Pi and VPS BaseURL should match")
	}
	if c.Proxy().BaseURL == c.VPS().BaseURL {
		t.Fatalf("Proxy BaseURL should differ from VPS BaseURL")
	}
}
