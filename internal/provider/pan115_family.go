package provider

import "strings"

type Pan115FamilyAdapter struct {
	StaticAdapter
}

func NewPan115FamilyAdapter(meta Provider, capability CapabilitySet) Adapter {
	return Pan115FamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
	}
}

func (a Pan115FamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	if strings.TrimSpace(profile.Token) == "" && strings.TrimSpace(profile.Cookie) == "" {
		return OperationResult{
			Status:  "missing_access_token_or_cookie",
			Message: "115 Open adapter requires a token or cookie.",
			Mode:    "pan115_family_placeholder",
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "115 Open scaffold validation passed credential checks.",
		Mode:    "pan115_family_placeholder",
	}
}

func (a Pan115FamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "115 Open adapter returned placeholder live list data.",
			Mode:    "pan115_family_placeholder",
		},
		Items: []map[string]interface{}{
			{
				"name":     inferName(req.Path, "115-remote.bin"),
				"path":     defaultPath(req.Path, "/"),
				"parentId": req.ParentID,
				"provider": a.MetaInfo.Key,
			},
		},
	}
}

func (a Pan115FamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "115 Open adapter returned placeholder metadata.",
			Mode:    "pan115_family_placeholder",
		},
		Entry: map[string]interface{}{
			"name":     inferName(req.Path, "115-remote.bin"),
			"path":     defaultPath(req.Path, "/"),
			"fileId":   req.FileID,
			"parentId": req.ParentID,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a Pan115FamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "115 Open adapter accepted create-dir request.",
		Mode:    "pan115_family_placeholder",
		Payload: map[string]interface{}{
			"parentId": req.ParentID,
			"dirName":  req.DirName,
		},
	}
}

func (a Pan115FamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.SHA1) != "" && req.Size > 0
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "115 Open adapter evaluated fast-upload candidate.",
			Mode:    "pan115_family_placeholder",
		},
		Candidate: candidate,
	}
}

func (a Pan115FamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "115 Open adapter refuses pending_manual items until binary fallback is implemented.",
				Mode:    "pan115_family_placeholder",
			},
		}
	}
	if req.Strategy == "fast_upload" && strings.TrimSpace(req.SHA1) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_sha1",
				Message: "Fast upload requires sha1 for the 115 Open adapter.",
				Mode:    "pan115_family_placeholder",
			},
		}
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "115 Open adapter recorded scaffold upload success.",
			Mode:    "pan115_family_placeholder",
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
