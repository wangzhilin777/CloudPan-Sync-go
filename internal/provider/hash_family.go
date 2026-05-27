package provider

import "strings"

type HashFamilyAdapter struct {
	StaticAdapter
}

func NewHashFamilyAdapter(meta Provider, capability CapabilitySet) Adapter {
	return HashFamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
	}
}

func (a HashFamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	if strings.TrimSpace(profile.Token) == "" {
		return OperationResult{
			Status:  "missing_access_token",
			Message: "Hash-family adapter requires a token.",
			Mode:    "hash_family_placeholder",
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Hash-family scaffold validation passed credential checks.",
		Mode:    "hash_family_placeholder",
	}
}

func (a HashFamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Hash-family adapter returned placeholder live list data.",
			Mode:    "hash_family_placeholder",
		},
		Items: []map[string]interface{}{
			{
				"name":     inferName(req.Path, "hash-remote.bin"),
				"path":     defaultPath(req.Path, "/"),
				"parentId": req.ParentID,
				"provider": a.MetaInfo.Key,
			},
		},
	}
}

func (a HashFamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Hash-family adapter returned placeholder metadata.",
			Mode:    "hash_family_placeholder",
		},
		Entry: map[string]interface{}{
			"name":     inferName(req.Path, "hash-remote.bin"),
			"path":     defaultPath(req.Path, "/"),
			"fileId":   req.FileID,
			"parentId": req.ParentID,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a HashFamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Hash-family adapter accepted create-dir request.",
		Mode:    "hash_family_placeholder",
		Payload: map[string]interface{}{
			"parentId": req.ParentID,
			"dirName":  req.DirName,
		},
	}
}

func (a HashFamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.GCID) != "" && req.Size > 0
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Hash-family adapter evaluated fast-upload candidate.",
			Mode:    "hash_family_placeholder",
		},
		Candidate: candidate,
	}
}

func (a HashFamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "Hash-family adapter refuses pending_manual items until binary fallback is implemented.",
				Mode:    "hash_family_placeholder",
			},
		}
	}
	if req.Strategy == "fast_upload" && strings.TrimSpace(req.GCID) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_gcid",
				Message: "Fast upload requires gcid for the hash-family adapter.",
				Mode:    "hash_family_placeholder",
			},
		}
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Hash-family adapter recorded scaffold upload success.",
			Mode:    "hash_family_placeholder",
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
