package auth

import (
	"context"
	"database/sql"
	"encoding/json"

	sqlitestore "cloudpan-sync-go/internal/store/sqlite"
)

func listProfiles(ctx context.Context, store *sqlitestore.Store) ([]Profile, error) {
	rows, err := store.DB().QueryContext(ctx, `
SELECT id, provider_key, auth_mode, display_name, token, cookie, extra_json, status, created_at, updated_at
FROM auth_profiles
ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []Profile{}
	for rows.Next() {
		item, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func getProfile(ctx context.Context, store *sqlitestore.Store, id string) (Profile, bool, error) {
	row := store.DB().QueryRowContext(ctx, `
SELECT id, provider_key, auth_mode, display_name, token, cookie, extra_json, status, created_at, updated_at
FROM auth_profiles
WHERE id = ?`, id)
	item, err := scanProfile(row)
	if err == sql.ErrNoRows {
		return Profile{}, false, nil
	}
	if err != nil {
		return Profile{}, false, err
	}
	return item, true, nil
}

func createProfile(ctx context.Context, store *sqlitestore.Store, profile Profile) error {
	extraJSON, err := json.Marshal(profile.Extra)
	if err != nil {
		return err
	}
	_, err = store.DB().ExecContext(ctx, `
INSERT INTO auth_profiles(id, provider_key, auth_mode, display_name, token, cookie, extra_json, status, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		profile.ID, profile.ProviderKey, profile.AuthMode, profile.DisplayName, profile.Token, profile.Cookie, string(extraJSON), profile.Status, profile.CreatedAt, profile.UpdatedAt,
	)
	return err
}

func updateProfile(ctx context.Context, store *sqlitestore.Store, profile Profile) error {
	extraJSON, err := json.Marshal(profile.Extra)
	if err != nil {
		return err
	}
	_, err = store.DB().ExecContext(ctx, `
UPDATE auth_profiles
SET auth_mode = ?, display_name = ?, token = ?, cookie = ?, extra_json = ?, status = ?, updated_at = ?
WHERE id = ?`,
		profile.AuthMode, profile.DisplayName, profile.Token, profile.Cookie, string(extraJSON), profile.Status, profile.UpdatedAt, profile.ID,
	)
	return err
}

func deleteProfile(ctx context.Context, store *sqlitestore.Store, id string) (bool, error) {
	result, err := store.DB().ExecContext(ctx, `DELETE FROM auth_profiles WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func saveValidation(ctx context.Context, store *sqlitestore.Store, validation Validation) error {
	_, err := store.DB().ExecContext(ctx, `
INSERT INTO auth_validations(id, profile_id, provider_key, ok, status, message, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		validation.ID, validation.ProfileID, validation.ProviderKey, boolToInt(validation.OK), validation.Status, validation.Message, validation.CreatedAt,
	)
	return err
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanProfile(scanner rowScanner) (Profile, error) {
	var item Profile
	var extraJSON string
	if err := scanner.Scan(&item.ID, &item.ProviderKey, &item.AuthMode, &item.DisplayName, &item.Token, &item.Cookie, &extraJSON, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return Profile{}, err
	}
	item.Extra = map[string]string{}
	if extraJSON != "" {
		if err := json.Unmarshal([]byte(extraJSON), &item.Extra); err != nil {
			return Profile{}, err
		}
	}
	return item, nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
