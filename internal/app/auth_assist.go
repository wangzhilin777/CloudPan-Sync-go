package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type authAssistDiscoverRequest struct {
	Kind    string `json:"kind"`
	BaseURL string `json:"baseUrl"`
	Token   string `json:"token"`
}

type authAssistStorage struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Driver    string `json:"driver"`
	MountPath string `json:"mountPath"`
	Status    string `json:"status"`
}

type authAssistDiscoverResponse struct {
	Kind      string             `json:"kind"`
	BaseURL   string             `json:"baseUrl"`
	Reachable bool               `json:"reachable"`
	Storages  []authAssistStorage `json:"storages"`
}

func normalizeAuthAssistKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "openlist":
		return "openlist"
	case "alist":
		return "alist"
	default:
		return ""
	}
}

func normalizeAuthAssistBaseURL(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(strings.ToLower(trimmed), "http://") && !strings.HasPrefix(strings.ToLower(trimmed), "https://") {
		trimmed = "http://" + trimmed
	}
	return strings.TrimRight(trimmed, "/")
}

func (a *App) handleAuthAssistDiscover(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	var req authAssistDiscoverRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
		return
	}
	kind := normalizeAuthAssistKind(req.Kind)
	if kind == "" {
		writeError(w, http.StatusBadRequest, "assist_kind_required", "Assist kind is required.")
		return
	}
	baseURL := normalizeAuthAssistBaseURL(req.BaseURL)
	if baseURL == "" {
		writeError(w, http.StatusBadRequest, "assist_url_required", "Assist base URL is required.")
		return
	}
	items, err := discoverAuthAssistStorages(r.Context(), kind, baseURL, strings.TrimSpace(req.Token))
	if err != nil {
		handleError(w, err)
		return
	}
	writeOK(w, http.StatusOK, authAssistDiscoverResponse{
		Kind:      kind,
		BaseURL:   baseURL,
		Reachable: true,
		Storages:  items,
	})
}

func discoverAuthAssistStorages(ctx context.Context, kind, baseURL, token string) ([]authAssistStorage, error) {
	requestBody := map[string]interface{}{
		"page":     1,
		"per_page": 1000,
	}
	payload, err := json.Marshal(requestBody)
	if err != nil {
		return nil, &HTTPError{Status: http.StatusInternalServerError, Code: "assist_request_build_failed", Message: "Failed to build assist request."}
	}

	reqCtx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, baseURL+"/api/admin/storage/list", bytes.NewReader(payload))
	if err != nil {
		return nil, &HTTPError{Status: http.StatusBadRequest, Code: "assist_url_invalid", Message: "Assist URL is invalid."}
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", token)
		req.Header.Set("X-Token", token)
	}

	resp, err := (&http.Client{Timeout: 6 * time.Second}).Do(req)
	if err != nil {
		return nil, &HTTPError{Status: http.StatusBadGateway, Code: "assist_connect_failed", Message: fmt.Sprintf("%s connection failed.", strings.Title(kind))}
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, &HTTPError{Status: http.StatusBadRequest, Code: "invalid_assist_token", Message: fmt.Sprintf("%s token is invalid or lacks permission.", strings.Title(kind))}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &HTTPError{Status: http.StatusBadGateway, Code: "assist_storage_list_failed", Message: fmt.Sprintf("%s storage list request failed.", strings.Title(kind))}
	}

	items, parseErr := parseAuthAssistStorages(body)
	if parseErr != nil {
		return nil, &HTTPError{Status: http.StatusBadGateway, Code: "assist_storage_parse_failed", Message: fmt.Sprintf("%s storage list response could not be parsed.", strings.Title(kind))}
	}
	return items, nil
}

func parseAuthAssistStorages(body []byte) ([]authAssistStorage, error) {
	var payload map[string]interface{}
	if err := decodeJSONFromBytes(body, &payload); err != nil {
		return nil, err
	}

	rawData := payload["data"]
	switch typed := rawData.(type) {
	case []interface{}:
		return convertAuthAssistStorages(typed), nil
	case map[string]interface{}:
		for _, key := range []string{"content", "items", "list"} {
			if list, ok := typed[key].([]interface{}); ok {
				return convertAuthAssistStorages(list), nil
			}
		}
	}
	return []authAssistStorage{}, nil
}

func convertAuthAssistStorages(items []interface{}) []authAssistStorage {
	out := make([]authAssistStorage, 0, len(items))
	for _, item := range items {
		row, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		out = append(out, authAssistStorage{
			ID:        stringValueOr(row["id"], ""),
			Name:      firstNonEmptyStringValue(stringValueOr(row["remark"], ""), stringValueOr(row["mount_path"], ""), stringValueOr(row["driver"], "")),
			Driver:    stringValueOr(row["driver"], ""),
			MountPath: stringValueOr(row["mount_path"], ""),
			Status:    stringValueOr(row["status"], ""),
		})
	}
	return out
}

func stringValueOr(value interface{}, fallback string) string {
	if value == nil {
		return fallback
	}
	text := strings.TrimSpace(fmt.Sprint(value))
	if text == "" || text == "<nil>" {
		return fallback
	}
	return text
}

func firstNonEmptyStringValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
