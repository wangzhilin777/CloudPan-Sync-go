package auth

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"cloudpan-sync-go/internal/provider"
	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
)

type CreateProfileInput struct {
	ProviderKey string            `json:"providerKey"`
	AuthMode    string            `json:"authMode"`
	DisplayName string            `json:"displayName"`
	Token       string            `json:"token"`
	Cookie      string            `json:"cookie"`
	Extra       map[string]string `json:"extra"`
}

type UpdateProfileInput struct {
	AuthMode    *string           `json:"authMode,omitempty"`
	DisplayName *string           `json:"displayName,omitempty"`
	Token       *string           `json:"token,omitempty"`
	Cookie      *string           `json:"cookie,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

type Service struct {
	store    *sqlitestore.Store
	registry *provider.Registry
}

func NewService(store *sqlitestore.Store, registry *provider.Registry) *Service {
	return &Service{store: store, registry: registry}
}

func (s *Service) ListProfiles(ctx context.Context) ([]Profile, error) {
	return listProfiles(ctx, s.store)
}

func (s *Service) GetProfile(ctx context.Context, id string) (Profile, bool, error) {
	return getProfile(ctx, s.store, id)
}

func (s *Service) CreateProfile(ctx context.Context, input CreateProfileInput) (Profile, error) {
	if err := s.validateProviderAndMode(input.ProviderKey, input.AuthMode); err != nil {
		return Profile{}, err
	}
	if strings.TrimSpace(input.DisplayName) == "" {
		return Profile{}, fmt.Errorf("display_name_required")
	}
	now := time.Now().UTC().Format(time.RFC3339)
	profile := Profile{
		ID:          uuid.NewString(),
		ProviderKey: strings.TrimSpace(input.ProviderKey),
		AuthMode:    strings.TrimSpace(input.AuthMode),
		DisplayName: strings.TrimSpace(input.DisplayName),
		Token:       strings.TrimSpace(input.Token),
		Cookie:      strings.TrimSpace(input.Cookie),
		Extra:       normalizeExtra(input.Extra),
		Status:      "saved",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := createProfile(ctx, s.store, profile); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func (s *Service) UpdateProfile(ctx context.Context, id string, input UpdateProfileInput) (Profile, bool, error) {
	current, ok, err := getProfile(ctx, s.store, id)
	if err != nil || !ok {
		return Profile{}, ok, err
	}
	if input.AuthMode != nil {
		if err := s.validateProviderAndMode(current.ProviderKey, strings.TrimSpace(*input.AuthMode)); err != nil {
			return Profile{}, true, err
		}
		current.AuthMode = strings.TrimSpace(*input.AuthMode)
	}
	if input.DisplayName != nil {
		current.DisplayName = strings.TrimSpace(*input.DisplayName)
	}
	if input.Token != nil {
		current.Token = strings.TrimSpace(*input.Token)
	}
	if input.Cookie != nil {
		current.Cookie = strings.TrimSpace(*input.Cookie)
	}
	if input.Extra != nil {
		current.Extra = normalizeExtra(input.Extra)
	}
	current.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if strings.TrimSpace(current.DisplayName) == "" {
		return Profile{}, true, fmt.Errorf("display_name_required")
	}
	if err := updateProfile(ctx, s.store, current); err != nil {
		return Profile{}, true, err
	}
	return current, true, nil
}

func (s *Service) DeleteProfile(ctx context.Context, id string) (bool, error) {
	return deleteProfile(ctx, s.store, id)
}

func (s *Service) ValidateProfile(ctx context.Context, id string) (Validation, bool, error) {
	profile, ok, err := getProfile(ctx, s.store, id)
	if err != nil || !ok {
		return Validation{}, ok, err
	}
	entry, exists := s.registry.Get(profile.ProviderKey)
	result := Validation{
		ID:          uuid.NewString(),
		ProfileID:   profile.ID,
		ProviderKey: profile.ProviderKey,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	switch {
	case !exists:
		result.Status = "provider_not_found"
		result.Message = "Provider is not registered."
	default:
		adapterResult := entry.Adapter.ValidateAuth(toProviderProfile(profile))
		result.OK = adapterResult.OK
		result.Status = adapterResult.Status
		result.Message = adapterResult.Message
		if result.OK {
			profile.Status = "verified"
		} else {
			profile.Status = "invalid"
		}
		profile.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
		if err := updateProfile(ctx, s.store, profile); err != nil {
			return Validation{}, true, err
		}
	}
	if err := saveValidation(ctx, s.store, result); err != nil {
		return Validation{}, true, err
	}
	return result, true, nil
}

func (s *Service) validateProviderAndMode(providerKey, authMode string) error {
	entry, ok := s.registry.Get(strings.TrimSpace(providerKey))
	if !ok {
		return fmt.Errorf("provider_not_found")
	}
	for _, item := range entry.Meta.AuthModes {
		if item == strings.TrimSpace(authMode) {
			return nil
		}
	}
	return fmt.Errorf("auth_mode_not_supported")
}

func normalizeExtra(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		out[key] = strings.TrimSpace(value)
	}
	return out
}

func toProviderProfile(profile Profile) provider.AuthProfile {
	return provider.AuthProfile{
		ID:          profile.ID,
		ProviderKey: profile.ProviderKey,
		AuthMode:    profile.AuthMode,
		DisplayName: profile.DisplayName,
		Token:       profile.Token,
		Cookie:      profile.Cookie,
		Extra:       profile.Extra,
	}
}
