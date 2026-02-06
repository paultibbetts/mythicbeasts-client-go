package vps

import "context"

// Zone represents a zone (datacentre) a VPS may be
// provisioned in. It can include its parent zones.
type Zone struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Parents     []string `json:"parents"`
}

// Zones maps Zone names to Zone details.
type Zones map[string]Zone

// GetZones retrieves the available zones
// a VPS may be provisioned in.
func (s *Service) GetZones(ctx context.Context) (Zones, error) {
	var result Zones
	if _, _, err := s.GetJSON(ctx, "/vps/zones", &result); err != nil {
		return nil, err
	}

	return result, nil
}
