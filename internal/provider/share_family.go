package provider

import "strings"

type ShareFamilyAdapter struct {
	StaticAdapter
	RequirePwdID bool
}

func NewShareFamilyAdapter(meta Provider, capability CapabilitySet, requirePwdID bool) Adapter {
	return ShareFamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
		RequirePwdID: requirePwdID,
	}
}

func (a ShareFamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	if strings.TrimSpace(profile.Cookie) == "" {
		return OperationResult{
			Status:  "missing_cookie",
			Message: "Share-family adapter requires a cookie.",
			Mode:    "share_family_placeholder",
		}
	}
	if a.RequirePwdID && strings.TrimSpace(profile.Extra["pwdId"]) == "" {
		return OperationResult{
			Status:  "missing_pwd_id",
			Message: "Share-family adapter requires extra.pwdId.",
			Mode:    "share_family_placeholder",
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Share-family scaffold validation passed credential checks.",
		Mode:    "share_family_placeholder",
	}
}

func (a ShareFamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Share-family adapter returned placeholder live list data.",
			Mode:    "share_family_placeholder",
		},
		Items: []map[string]interface{}{
			{
				"name":     inferName(req.Path, "share-remote.bin"),
				"path":     defaultPath(req.Path, "/"),
				"parentId": req.ParentID,
				"provider": a.MetaInfo.Key,
				"pwdId":    req.Profile.Extra["pwdId"],
			},
		},
	}
}

func (a ShareFamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Share-family adapter returned placeholder metadata.",
			Mode:    "share_family_placeholder",
		},
		Entry: map[string]interface{}{
			"name":     inferName(req.Path, "share-remote.bin"),
			"path":     defaultPath(req.Path, "/"),
			"fileId":   req.FileID,
			"parentId": req.ParentID,
			"provider": a.MetaInfo.Key,
			"pwdId":    req.Profile.Extra["pwdId"],
		},
	}
}

func (a ShareFamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Share-family adapter accepted create-dir request.",
		Mode:    "share_family_placeholder",
		Payload: map[string]interface{}{
			"parentId": req.ParentID,
			"dirName":  req.DirName,
			"pwdId":    req.Profile.Extra["pwdId"],
		},
	}
}

func (a ShareFamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.MD5) != "" && req.Size > 0
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Share-family adapter evaluated fast-upload candidate.",
			Mode:    "share_family_placeholder",
		},
		Candidate: candidate,
	}
}

func (a ShareFamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "Share-family adapter refuses pending_manual items until multipart fallback is implemented.",
				Mode:    "share_family_placeholder",
			},
		}
	}
	if req.Strategy == "fast_upload" && strings.TrimSpace(req.MD5) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_md5",
				Message: "Fast upload requires md5 for the share-family adapter.",
				Mode:    "share_family_placeholder",
			},
		}
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Share-family adapter recorded scaffold upload success.",
			Mode:    "share_family_placeholder",
			Payload: map[string]interface{}{
				"path":     req.Path,
				"parentId": req.ParentID,
				"name":     req.Name,
				"strategy": req.Strategy,
				"provider": a.MetaInfo.Key,
				"pwdId":    req.Profile.Extra["pwdId"],
			},
		},
		ConflictAction: "none",
	}
}
