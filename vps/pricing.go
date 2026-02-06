package vps

import "context"

// Pricing represents the pricing information
// used for on-demand VPS resources.
type Pricing struct {
	Disk     DiskPrices       `json:"disk"`
	IPv4     int64            `json:"ipv4"`
	Products map[string]int64 `json:"products"`
}

// DiskPrices represents the pricing information
// for different disk types available for a VPS.
type DiskPrices struct {
	SSD DiskPricing `json:"ssd"`
	HDD DiskPricing `json:"hdd"`
}

// DiskPricing represents the price of a disk type.
// Price is in pence per unit, and Extent is the number of GB per unit.
type DiskPricing struct {
	Price  int64 `json:"price"`
	Extent int64 `json:"extent"`
}

// GetPricing retrieves the Pricing for
// on-demand VPS products.
func (s *Service) GetPricing(ctx context.Context) (Pricing, error) {
	var result Pricing
	if _, _, err := s.GetJSON(ctx, "/vps/pricing", &result); err != nil {
		return Pricing{}, err
	}

	return result, nil
}
