package provider

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

var providerHTTPClient = &http.Client{Timeout: 8 * time.Second}

func decodeProviderJSONResponse(statusCode int, bodyBytes []byte) (map[string]interface{}, error) {
	if len(bodyBytes) == 0 {
		return map[string]interface{}{}, nil
	}
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payloadMap); err != nil {
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			return map[string]interface{}{
				"rawBody": strings.TrimSpace(string(bodyBytes)),
			}, nil
		}
		return nil, fmt.Errorf("decode provider json: %w", err)
	}
	return payloadMap, nil
}

func postProviderJSON(ctx context.Context, endpoint string, token string, body interface{}) (int, map[string]interface{}, error) {
	payload := []byte("{}")
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		payload = raw
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if strings.TrimSpace(token) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(token))
	}

	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	payloadMap, err := decodeProviderJSONResponse(resp.StatusCode, bodyBytes)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, payloadMap, nil
}

func putProviderBytes(ctx context.Context, endpoint string, body []byte, headers map[string]string) (int, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	for key, value := range headers {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}

	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, resp.Header.Clone(), nil
}
