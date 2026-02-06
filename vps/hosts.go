package vps

import "context"

// Host represents an available private cloud host.
type Host struct {
	Name     string   `json:"name"`
	Cores    int64    `json:"cores"`
	RAM      int64    `json:"ram"`
	Disk     HostDisk `json:"disk"`
	FreeRAM  int64    `json:"free_ram"`
	FreeDisk HostDisk `json:"free_disk"`
}

// HostDisk represents the disk information of a Host.
type HostDisk struct {
	SSD int64 `json:"ssd"`
	HDD int64 `json:"hdd"`
}

// Hosts maps Host names to Host details.
type Hosts map[string]Host

// GetHosts retrieves the available private cloud hosts.
func (s *Service) GetHosts(ctx context.Context) (Hosts, error) {
	var result Hosts
	if _, _, err := s.GetJSON(ctx, "/vps/hosts", &result); err != nil {
		return nil, err
	}

	return result, nil
}
