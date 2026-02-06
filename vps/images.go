package vps

import "context"

// Image represents a VPS operating system image.
type Image struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// Images maps image names to image details.
type Images map[string]Image

// GetImages retrieves the available operating system images for a VPS.
func (s *Service) GetImages(ctx context.Context) (Images, error) {
	var result Images
	if _, _, err := s.GetJSON(ctx, "/vps/images", &result); err != nil {
		return nil, err
	}

	return result, nil
}
