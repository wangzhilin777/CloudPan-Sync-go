package provider

import "strings"

type BaiduFamilyAdapter struct {
	StaticAdapter
}

func NewBaiduFamilyAdapter(meta Provider, capability CapabilitySet) Adapter {
	return BaiduFamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
	}
}

func (a BaiduFamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	if strings.TrimSpace(profile.Token) == "" && strings.TrimSpace(profile.Cookie) == "" {
		return OperationResult{
			Status:  "missing_access_token_or_cookie",
			Message: "Baidu adapter requires a token or cookie.",
			Mode:    "baidu_family_placeholder",
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Baidu scaffold validation passed credential checks.",
		Mode:    "baidu_family_placeholder",
	}
}

func (a BaiduFamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Baidu adapter returned placeholder live list data.",
			Mode:    "baidu_family_placeholder",
		},
		Items: []map[string]interface{}{
			{
				"name":     inferName(req.Path, "baidu-remote.bin"),
				"path":     defaultPath(req.Path, "/"),
				"parentId": req.ParentID,
				"provider": a.MetaInfo.Key,
			},
		},
	}
}

func (a BaiduFamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Baidu adapter returned placeholder metadata.",
			Mode:    "baidu_family_placeholder",
		},
		Entry: map[string]interface{}{
			"name":     inferName(req.Path, "baidu-remote.bin"),
			"path":     defaultPath(req.Path, "/"),
			"fileId":   req.FileID,
			"parentId": req.ParentID,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a BaiduFamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Baidu adapter accepted create-dir request.",
		Mode:    "baidu_family_placeholder",
		Payload: map[string]interface{}{
			"parentId": req.ParentID,
			"dirName":  req.DirName,
		},
	}
}

func (a BaiduFamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.MD5) != "" && req.Size > 0
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Baidu adapter evaluated fast-upload candidate.",
			Mode:    "baidu_family_placeholder",
		},
		Candidate: candidate,
	}
}

func (a BaiduFamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "Baidu adapter refuses pending_manual items until binary fallback is implemented.",
				Mode:    "baidu_family_placeholder",
			},
		}
	}
	if req.Strategy == "fast_upload" && strings.TrimSpace(req.MD5) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_md5",
				Message: "Fast upload requires md5 for the Baidu adapter.",
				Mode:    "baidu_family_placeholder",
			},
		}
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Baidu adapter recorded scaffold upload success.",
			Mode:    "baidu_family_placeholder",
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
