package mythicbeasts

import (
	"github.com/paultibbetts/mythicbeasts-client-go/pi"
	"github.com/paultibbetts/mythicbeasts-client-go/proxy"
	"github.com/paultibbetts/mythicbeasts-client-go/vps"
)

// Pi returns the Raspberry Pi service client.
func (c *Client) Pi() *pi.Service {
	if c == nil {
		return nil
	}
	if c.piService == nil {
		c.piService = pi.NewService(c)
	}
	return c.piService
}

// VPS returns the VPS service client.
func (c *Client) VPS() *vps.Service {
	if c == nil {
		return nil
	}
	if c.vpsService == nil {
		c.vpsService = vps.NewService(c)
	}
	return c.vpsService
}

// Proxy returns the Proxy service client.
func (c *Client) Proxy() *proxy.Service {
	if c == nil {
		return nil
	}
	if c.proxyService == nil {
		c.proxyService = proxy.NewService(c)
	}
	return c.proxyService
}
