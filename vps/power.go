package vps

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// PowerAction represents a supported VPS power operation.
type PowerAction string

const (
	PowerActionOn       PowerAction = "power-on"
	PowerActionOff      PowerAction = "power-off"
	PowerActionShutdown PowerAction = "shutdown"

	// DefaultRebootGracePeriod is the default wait time after a reboot request.
	DefaultRebootGracePeriod = 2 * time.Minute
	// DefaultShutdownGracePeriod is the default wait time after a shutdown request.
	DefaultShutdownGracePeriod = 2 * time.Minute
)

// IsValid reports whether the power action is accepted by the API.
func (p PowerAction) IsValid() bool {
	switch p {
	case PowerActionOn, PowerActionOff, PowerActionShutdown:
		return true
	default:
		return false
	}
}

// PowerRequest represents the request payload for a power operation.
type PowerRequest struct {
	Power PowerAction `json:"power"`
}

// PowerResponse represents the response from a power operation.
type PowerResponse struct {
	Message string `json:"message"`
}

// RebootResponse represents the response from a reboot operation.
type RebootResponse struct {
	Message string `json:"message"`
}

// Reboot initiates an ACPI reboot for the VPS.
// The call returns once reboot has been initiated.
func (s *Service) Reboot(ctx context.Context, identifier string) (RebootResponse, error) {
	if strings.TrimSpace(identifier) == "" {
		return RebootResponse{}, ErrEmptyIdentifier
	}

	url := fmt.Sprintf("/vps/servers/%s/reboot", identifier)

	var result RebootResponse
	if _, _, err := s.DoJSON(ctx, http.MethodPost, url, nil, &result, http.StatusOK); err != nil {
		return RebootResponse{}, err
	}

	return result, nil
}

// RebootWithGrace initiates an ACPI reboot and waits for a grace period.
// If gracePeriod <= 0, DefaultRebootGracePeriod is used.
func (s *Service) RebootWithGrace(ctx context.Context, identifier string, gracePeriod time.Duration) (RebootResponse, error) {
	resp, err := s.Reboot(ctx, identifier)
	if err != nil {
		return RebootResponse{}, err
	}

	if err := waitWithDefaultGrace(ctx, identifier, "reboot", gracePeriod, DefaultRebootGracePeriod); err != nil {
		return RebootResponse{}, err
	}

	return resp, nil
}

// SetPower changes VPS power state (power-on, power-off, or shutdown).
func (s *Service) SetPower(ctx context.Context, identifier string, action PowerAction) (PowerResponse, error) {
	if strings.TrimSpace(identifier) == "" {
		return PowerResponse{}, ErrEmptyIdentifier
	}
	if !action.IsValid() {
		return PowerResponse{}, fmt.Errorf("invalid power action %q", action)
	}

	url := fmt.Sprintf("/vps/servers/%s/power", identifier)
	payload := PowerRequest{Power: action}

	var result PowerResponse
	if _, _, err := s.DoJSON(ctx, http.MethodPut, url, payload, &result, http.StatusOK); err != nil {
		return PowerResponse{}, err
	}

	return result, nil
}

// ShutdownWithGrace requests ACPI shutdown and waits for a grace period.
// If gracePeriod <= 0, DefaultShutdownGracePeriod is used.
func (s *Service) ShutdownWithGrace(ctx context.Context, identifier string, gracePeriod time.Duration) (PowerResponse, error) {
	resp, err := s.SetPower(ctx, identifier, PowerActionShutdown)
	if err != nil {
		return PowerResponse{}, err
	}

	if err := waitWithDefaultGrace(ctx, identifier, "shutdown", gracePeriod, DefaultShutdownGracePeriod); err != nil {
		return PowerResponse{}, err
	}

	return resp, nil
}

func waitWithDefaultGrace(ctx context.Context, identifier string, op string, gracePeriod time.Duration, defaultGrace time.Duration) error {
	grace := gracePeriod
	if grace <= 0 {
		grace = defaultGrace
	}

	log.Printf("vps[%s] %s requested; waiting grace period %s", identifier, op, grace)

	timer := time.NewTimer(grace)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
