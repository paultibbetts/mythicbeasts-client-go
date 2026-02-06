package vps

import "github.com/paultibbetts/mythicbeasts-client-go/internal/transport"

// BaseURL is the default base URL for VPS API requests.
const BaseURL string = "https://api.mythic-beasts.com/beta"

// Service provides access to the VPS API.
type Service struct {
	transport.BaseService
}

// NewService constructs a VPS API service client.
func NewService(c transport.Requester) *Service {
	return &Service{BaseService: transport.NewBaseService(c, BaseURL)}
}
