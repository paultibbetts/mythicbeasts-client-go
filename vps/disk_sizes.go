package vps

import "context"

// DiskSizes represents the available disk sizes for a VPS.
type DiskSizes struct {
	HDD []int64 `json:"hdd"`
	SSD []int64 `json:"ssd"`
}

// GetDiskSizes retrieves the available disk sizes.
func (s *Service) GetDiskSizes(ctx context.Context) (*DiskSizes, error) {
	var result DiskSizes
	if _, _, err := s.GetJSON(ctx, "/vps/disk-sizes", &result); err != nil {
		return nil, err
	}

	return &result, nil
}
