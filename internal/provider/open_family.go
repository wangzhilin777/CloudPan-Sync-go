package provider

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

type OpenFamilyAdapter struct {
	StaticAdapter
	RequireDomainDrive bool
}

func NewOpenFamilyAdapter(meta Provider, capability CapabilitySet, requireDomainDrive bool) Adapter {
	return OpenFamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
		RequireDomainDrive: requireDomainDrive,
	}
}

func (a OpenFamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	if strings.TrimSpace(profile.Token) == "" {
		return OperationResult{
			Status:  "missing_access_token",
			Message: "Open-family adapter requires a token.",
			Mode:    "open_family_placeholder",
		}
	}
	if a.RequireDomainDrive {
		domainID := strings.TrimSpace(profile.Extra["domainId"])
		driveID := strings.TrimSpace(profile.Extra["driveId"])
		if domainID == "" || driveID == "" {
			return OperationResult{
				Status:  "missing_domain_or_drive_id",
				Message: "Aliyun Open adapter requires extra.domainId and extra.driveId.",
				Mode:    "open_family_placeholder",
			}
		}
		return a.validateAliyunOpenAuth(profile, domainID, driveID)
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Open-family scaffold validation passed credential checks.",
		Mode:    "open_family_placeholder",
	}
}

func (a OpenFamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Open-family adapter returned placeholder live list data.",
			Mode:    "open_family_placeholder",
		},
		Items: []map[string]interface{}{
			{
				"name":     inferName(req.Path, "remote.bin"),
				"path":     defaultPath(req.Path, "/"),
				"parentId": req.ParentID,
				"provider": a.MetaInfo.Key,
			},
		},
	}
}

func (a OpenFamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Open-family adapter returned placeholder live metadata.",
			Mode:    "open_family_placeholder",
		},
		Entry: map[string]interface{}{
			"name":     inferName(req.Path, "remote.bin"),
			"path":     defaultPath(req.Path, "/"),
			"fileId":   req.FileID,
			"parentId": req.ParentID,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a OpenFamilyAdapter) validateAliyunOpenAuth(profile AuthProfile, domainID string, driveID string) OperationResult {
	baseEndpoint, err := resolveOpenFamilyEndpoint(profile, domainID)
	if err != nil {
		return OperationResult{
			Status:  "invalid_provider_endpoint",
			Message: err.Error(),
			Mode:    "open_family_real_auth",
		}
	}

	userStatus, userPayload, err := postProviderJSON(context.Background(), baseEndpoint+"/v2/user/get", profile.Token, map[string]interface{}{})
	if err != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Aliyun Open user validation request failed: %v", err),
			Mode:    "open_family_real_auth",
		}
	}
	if userStatus == http.StatusUnauthorized || userStatus == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "Aliyun Open rejected the supplied access token.",
			Mode:    "open_family_real_auth",
		}
	}
	if userStatus < 200 || userStatus >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Aliyun Open user validation returned HTTP %d.", userStatus),
			Mode:    "open_family_real_auth",
			Payload: userPayload,
		}
	}

	if gotDomainID := stringMapValue(userPayload, "domain_id"); gotDomainID != "" && gotDomainID != domainID {
		return OperationResult{
			Status:  "domain_id_mismatch",
			Message: fmt.Sprintf("Aliyun Open returned domain_id %q, expected %q.", gotDomainID, domainID),
			Mode:    "open_family_real_auth",
			Payload: userPayload,
		}
	}

	driveStatus, drivePayload, err := postProviderJSON(context.Background(), baseEndpoint+"/v2/drive/get_default_drive", profile.Token, map[string]interface{}{})
	if err != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Aliyun Open default drive request failed: %v", err),
			Mode:    "open_family_real_auth",
			Payload: userPayload,
		}
	}
	if driveStatus == http.StatusUnauthorized || driveStatus == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "Aliyun Open rejected the supplied access token while reading default drive.",
			Mode:    "open_family_real_auth",
			Payload: userPayload,
		}
	}
	if driveStatus < 200 || driveStatus >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Aliyun Open default drive request returned HTTP %d.", driveStatus),
			Mode:    "open_family_real_auth",
			Payload: mergePayloads(userPayload, drivePayload),
		}
	}

	if gotDriveID := stringMapValue(drivePayload, "drive_id"); gotDriveID != "" && gotDriveID != driveID {
		return OperationResult{
			Status:  "drive_id_mismatch",
			Message: fmt.Sprintf("Aliyun Open returned drive_id %q, expected %q.", gotDriveID, driveID),
			Mode:    "open_family_real_auth",
			Payload: mergePayloads(userPayload, drivePayload),
		}
	}

	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Aliyun Open validated the supplied access token against live user and default-drive endpoints.",
		Mode:    "open_family_real_auth",
		Payload: mergePayloads(userPayload, drivePayload),
	}
}

func resolveOpenFamilyEndpoint(profile AuthProfile, domainID string) (string, error) {
	if raw := strings.TrimSpace(profile.Extra["apiEndpoint"]); raw != "" {
		parsed, err := url.Parse(raw)
		if err != nil {
			return "", fmt.Errorf("invalid apiEndpoint: %w", err)
		}
		if parsed.Scheme == "" || parsed.Host == "" {
			return "", fmt.Errorf("invalid apiEndpoint: scheme and host are required")
		}
		return strings.TrimRight(parsed.String(), "/"), nil
	}
	endpoint := fmt.Sprintf("https://%s.api.aliyunpds.com", domainID)
	return strings.TrimRight(endpoint, "/"), nil
}

func mergePayloads(first map[string]interface{}, second map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range first {
		out["user."+key] = value
	}
	for key, value := range second {
		out["drive."+key] = value
	}
	return out
}

func stringMapValue(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func (a OpenFamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Open-family adapter accepted create-dir request.",
		Mode:    "open_family_placeholder",
		Payload: map[string]interface{}{
			"parentId": req.ParentID,
			"dirName":  req.DirName,
		},
	}
}

func (a OpenFamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.MD5) != "" && req.Size > 0
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Open-family adapter evaluated fast-upload candidate.",
			Mode:    "open_family_placeholder",
		},
		Candidate: candidate,
	}
}

func (a OpenFamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "Open-family adapter refuses pending_manual items until a real fallback runtime is implemented.",
				Mode:    "open_family_placeholder",
			},
		}
	}
	if req.Strategy == "fast_upload" && strings.TrimSpace(req.MD5) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_md5",
				Message: "Fast upload requires md5 for the open-family adapter.",
				Mode:    "open_family_placeholder",
			},
		}
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Open-family adapter recorded scaffold upload success.",
			Mode:    "open_family_placeholder",
			Payload: map[string]interface{}{
				"path":     req.Path,
				"parentId": req.ParentID,
				"name":     req.Name,
				"strategy": req.Strategy,
				"provider": a.MetaInfo.Key,
			},
		},
		ConflictAction: "none",
	}
}
