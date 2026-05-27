package provider

import "strings"

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
