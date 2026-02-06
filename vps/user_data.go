package vps

import (
	"context"
	"fmt"
	"net/http"
)

// NewUserData represents the data required to create
// a new User Data snippet.
type NewUserData struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

// UserData represents a User Data snippet.
type UserData struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Data string `json:"content"`
	Size int64  `json:"size"`
}

// UserDataSnippets maps snippet IDs to user data snippets.
type UserDataSnippets map[string]UserData

// CreateUserData creates a new User Data snippet with the given ID.
func (s *Service) CreateUserData(ctx context.Context, data NewUserData) (UserData, error) {
	path := "/vps/user-data"

	var created UserData
	if _, _, err := s.DoJSON(ctx, http.MethodPost, path, data, &created, http.StatusCreated); err != nil {
		return UserData{}, err
	}

	return created, nil
}

// GetUserData retrieves the User Data snippet with the given ID.
func (s *Service) GetUserData(ctx context.Context, id int64) (UserData, error) {
	requestURL := fmt.Sprintf("/vps/user-data/%d", id)

	var result UserData
	if _, _, err := s.GetJSON(ctx, requestURL, &result); err != nil {
		return UserData{}, err
	}

	return result, nil
}

func (s *Service) GetUserDataSnippets(ctx context.Context) (UserDataSnippets, error) {
	var resp struct {
		UserData UserDataSnippets `json:"user_data"`
	}
	if _, _, err := s.GetJSON(ctx, "/vps/user-data", &resp); err != nil {
		return nil, err
	}

	return resp.UserData, nil
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

// DeleteUserData removes the User Data snippet with the given ID.
func (s *Service) DeleteUserData(ctx context.Context, id int64) error {
	url := fmt.Sprintf("/vps/user-data/%d", id)

	return s.BaseService.Delete(ctx, url)
}
