package provider

import "strings"

type Cloud189FamilyAdapter struct {
	StaticAdapter
}

func NewCloud189FamilyAdapter(meta Provider, capability CapabilitySet) Adapter {
	return Cloud189FamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
	}
}

func (a Cloud189FamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	if strings.TrimSpace(profile.Cookie) == "" {
		return OperationResult{
			Status:  "missing_cookie",
			Message: "189Cloud adapter requires a cookie.",
			Mode:    "cloud189_family_placeholder",
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "189Cloud scaffold validation passed credential checks.",
		Mode:    "cloud189_family_placeholder",
	}
}

func (a Cloud189FamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "189Cloud adapter returned placeholder live list data.",
			Mode:    "cloud189_family_placeholder",
		},
		Items: []map[string]interface{}{
			{
				"name":     inferName(req.Path, "189cloud-remote.bin"),
				"path":     defaultPath(req.Path, "/"),
				"parentId": req.ParentID,
				"provider": a.MetaInfo.Key,
			},
		},
	}
}

func (a Cloud189FamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "189Cloud adapter returned placeholder metadata.",
			Mode:    "cloud189_family_placeholder",
		},
		Entry: map[string]interface{}{
			"name":     inferName(req.Path, "189cloud-remote.bin"),
			"path":     defaultPath(req.Path, "/"),
			"fileId":   req.FileID,
			"parentId": req.ParentID,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a Cloud189FamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "189Cloud adapter accepted create-dir request.",
		Mode:    "cloud189_family_placeholder",
		Payload: map[string]interface{}{
			"parentId": req.ParentID,
			"dirName":  req.DirName,
		},
	}
}

func (a Cloud189FamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.MD5) != "" && req.Size > 0
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "189Cloud adapter evaluated fast-upload candidate.",
			Mode:    "cloud189_family_placeholder",
		},
		Candidate: candidate,
	}
}

func (a Cloud189FamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "189Cloud adapter refuses pending_manual items until binary fallback is implemented.",
				Mode:    "cloud189_family_placeholder",
			},
		}
	}
	if req.Strategy == "fast_upload" && strings.TrimSpace(req.MD5) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_md5",
				Message: "Fast upload requires md5 for the 189Cloud adapter.",
				Mode:    "cloud189_family_placeholder",
			},
		}
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "189Cloud adapter recorded scaffold upload success.",
			Mode:    "cloud189_family_placeholder",
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
