package vps

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
)

// NewUserData represents the data required to create
// a new User Data snippet.
type NewUserData struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// UpdateUserData represents the data required to update
// an existing User Data snippet.
type UpdateUserData struct {
	Data string `json:"data"`
}

// UserData represents a User Data snippet.
type UserData struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	// Data is the snippet payload. The API may return this as "data" or "content".
	Data string `json:"data"`
	Size int64  `json:"size"`
}

// UserDataSnippets maps snippet IDs to user data snippets.
type UserDataSnippets map[string]UserData

// CreateUserData creates a new User Data snippet.
func (s *Service) CreateUserData(ctx context.Context, data NewUserData) (UserData, error) {
	path := "/vps/user-data"

	var created UserData
	if _, _, err := s.DoJSON(ctx, http.MethodPost, path, data, &created, http.StatusOK, http.StatusCreated); err != nil {
		return UserData{}, err
	}

	return created, nil
}

// GetUserData retrieves the User Data snippet with the given ID.
func (s *Service) GetUserData(ctx context.Context, id int64) (UserData, error) {
	requestURL := fmt.Sprintf("/vps/user-data/%d", id)

	var raw map[string]any
	if _, _, err := s.GetJSON(ctx, requestURL, &raw, http.StatusOK); err != nil {
		return UserData{}, err
	}

	return parseUserData(raw, true)
}

func (s *Service) GetUserDataSnippets(ctx context.Context) (UserDataSnippets, error) {
	var resp struct {
		UserData map[string]map[string]any `json:"user_data"`
	}
	if _, _, err := s.GetJSON(ctx, "/vps/user-data", &resp, http.StatusOK); err != nil {
		return nil, err
	}

	snippets := make(UserDataSnippets, len(resp.UserData))
	for key, raw := range resp.UserData {
		data, err := parseUserData(raw, false)
		if err != nil {
			return nil, err
		}
		snippets[key] = data
	}

	return snippets, nil
}

func (s *Service) GetUserDataByName(ctx context.Context, name string) (UserData, error) {
	snippets, err := s.GetUserDataSnippets(ctx)
	if err != nil {
		return UserData{}, err
	}

	var id int64
	for _, data := range snippets {
		if data.Name == name {
			id = data.ID
		}
	}

	if id == 0 {
		return UserData{}, &ErrUserDataNotFound{Name: name}
	}

	return s.GetUserData(ctx, id)
}

// UpdateUserData updates the User Data snippet with the given ID.
func (s *Service) UpdateUserData(ctx context.Context, id int64, data UpdateUserData) error {
	url := fmt.Sprintf("/vps/user-data/%d", id)
	_, _, err := s.DoJSON(ctx, http.MethodPut, url, data, nil, http.StatusOK)
	return err
}

// DeleteUserData removes the User Data snippet with the given ID.
func (s *Service) DeleteUserData(ctx context.Context, id int64) error {
	url := fmt.Sprintf("/vps/user-data/%d", id)

	return s.BaseService.Delete(ctx, url)
}

func parseUserData(raw map[string]any, requireData bool) (UserData, error) {
	if raw == nil {
		return UserData{}, &ErrMalformedResponse{Resource: "user_data", Reason: "empty object"}
	}

	id, err := parseFlexibleInt(raw["id"], "id")
	if err != nil {
		return UserData{}, err
	}

	size, err := parseFlexibleInt(raw["size"], "size")
	if err != nil {
		return UserData{}, err
	}

	name, ok := raw["name"].(string)
	if !ok {
		return UserData{}, &ErrMalformedResponse{Resource: "user_data", Field: "name", Reason: "expected string"}
	}

	data, ok, err := parseSnippetContent(raw)
	if err != nil {
		return UserData{}, err
	}
	if requireData && !ok {
		return UserData{}, &ErrMalformedResponse{Resource: "user_data", Field: "data", Reason: "missing field"}
	}

	return UserData{
		ID:   id,
		Name: name,
		Data: data,
		Size: size,
	}, nil
}

// parseSnippetContent reads snippet payload from "data", falling back to "content".
// If neither exists, ok is false.
func parseSnippetContent(raw map[string]any) (value string, ok bool, err error) {
	if data, exists := raw["data"]; exists {
		str, valid := data.(string)
		if !valid {
			return "", false, &ErrMalformedResponse{Resource: "user_data", Field: "data", Reason: "expected string"}
		}
		return str, true, nil
	}
	if content, exists := raw["content"]; exists {
		str, valid := content.(string)
		if !valid {
			return "", false, &ErrMalformedResponse{Resource: "user_data", Field: "content", Reason: "expected string"}
		}
		return str, true, nil
	}
	return "", false, nil
}

func parseFlexibleInt(v any, field string) (int64, error) {
	switch value := v.(type) {
	case nil:
		return 0, &ErrMalformedResponse{Resource: "user_data", Field: field, Reason: "missing field"}
	case float64:
		if math.Trunc(value) != value {
			return 0, &ErrMalformedResponse{Resource: "user_data", Field: field, Reason: "expected integer"}
		}
		return int64(value), nil
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return 0, &ErrMalformedResponse{Resource: "user_data", Field: field, Reason: "invalid integer string"}
		}
		return n, nil
	default:
		return 0, &ErrMalformedResponse{Resource: "user_data", Field: field, Reason: "expected integer or string"}
	}
}
