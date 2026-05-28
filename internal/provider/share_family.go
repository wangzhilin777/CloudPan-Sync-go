package provider

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	quarkAPIBase      = "https://pc-api.uc.cn"
	quarkDriveAPIBase = "https://drive-pc.quark.cn"
	ucAPIBase         = "https://pc-api.uc.cn"
	ucDriveAPIBase    = "https://pc-api.uc.cn"
)

type ShareFamilyAdapter struct {
	StaticAdapter
	RequirePwdID bool
}

type shareFamilySession struct {
	APIEndpoint   string
	DriveEndpoint string
	Cookie        string
	PwdID         string
	Passcode      string
	ProviderKey   string
}

type shareFamilyUploadSession struct {
	AuthInfo  map[string]interface{}
	Bucket    string
	ObjKey    string
	UploadID  string
	UploadURL string
	Callback  map[string]interface{}
	PartSize  int64
	TaskID    string
	FileID    string
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
	session, err := a.newShareFamilySession(profile)
	if err != nil {
		return OperationResult{
			Status:  normalizeShareFamilySessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "share_family_real_auth",
		}
	}
	statusCode, payload, stoken, requestErr := getShareFamilyToken(context.Background(), session)
	if requestErr != nil {
		return OperationResult{
			Status:  normalizeShareFamilyRequestErrorStatus(requestErr),
			Message: fmt.Sprintf("%s auth validation request failed: %v", a.MetaInfo.DisplayName, requestErr),
			Mode:    "share_family_real_auth",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: fmt.Sprintf("%s rejected the supplied cookie.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_auth",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("%s auth validation returned HTTP %d.", a.MetaInfo.DisplayName, statusCode),
			Mode:    "share_family_real_auth",
			Payload: payload,
		}
	}
	if strings.TrimSpace(stoken) == "" {
		return OperationResult{
			Status:  "auth_invalid",
			Message: fmt.Sprintf("%s did not return a usable share token.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_auth",
			Payload: payload,
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: fmt.Sprintf("%s validated the supplied cookie and pwdId against the live share token endpoint.", a.MetaInfo.DisplayName),
		Mode:    "share_family_real_auth",
		Payload: map[string]interface{}{
			"pwdId":  session.PwdID,
			"stoken": stoken,
			"raw":    payload,
		},
	}
}

func (a ShareFamilyAdapter) List(req ListRequest) ListResult {
	session, validation := a.shareFamilyValidatedSession(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	targetPath := normalizeOpenFamilyPath(req.Path)
	parentID := strings.TrimSpace(req.ParentID)
	basePath := targetPath
	if parentID == "" {
		if targetPath == "/" {
			parentID = "0"
			basePath = "/"
		} else {
			resolvedParentID, resolvedEntry, found, resolveErr := a.resolveShareFamilyEntryByPath(session, targetPath, req.PageSize)
			if resolveErr != nil {
				return ListResult{
					OperationResult: OperationResult{
						Status:  normalizeShareFamilyRequestErrorStatus(resolveErr),
						Message: fmt.Sprintf("%s path resolution failed: %v", a.MetaInfo.DisplayName, resolveErr),
						Mode:    "share_family_real_directory",
					},
				}
			}
			if !found {
				return ListResult{
					OperationResult: OperationResult{
						Status:  "path_not_found",
						Message: fmt.Sprintf("%s path %q was not found.", a.MetaInfo.DisplayName, targetPath),
						Mode:    "share_family_real_directory",
					},
				}
			}
			if !boolMapValue(resolvedEntry, "isDir") {
				return ListResult{
					OperationResult: OperationResult{
						OK:      true,
						Status:  "ok",
						Message: fmt.Sprintf("%s resolved a file path directly.", a.MetaInfo.DisplayName),
						Mode:    "share_family_real_directory",
					},
					Items: []map[string]interface{}{resolvedEntry},
				}
			}
			parentID = resolvedParentID
			basePath = targetPath
		}
	}

	items, listErr := a.listShareFamilyByParent(session, parentID, basePath, req.PageSize)
	if listErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeShareFamilyRequestErrorStatus(listErr),
				Message: fmt.Sprintf("%s list request failed: %v", a.MetaInfo.DisplayName, listErr),
				Mode:    "share_family_real_directory",
			},
		}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: fmt.Sprintf("%s listed live directory entries.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_directory",
		},
		Items: items,
	}
}

func (a ShareFamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	session, validation := a.shareFamilyValidatedSession(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	if targetPath == "/" && strings.TrimSpace(req.FileID) == "" {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: fmt.Sprintf("%s root directory metadata is available.", a.MetaInfo.DisplayName),
				Mode:    "share_family_real_directory",
			},
			Entry: map[string]interface{}{
				"exists":   true,
				"isDir":    true,
				"name":     "/",
				"path":     "/",
				"fileId":   "0",
				"parentId": "",
				"provider": a.MetaInfo.Key,
			},
		}
	}

	parentID := strings.TrimSpace(req.ParentID)
	targetFileID := strings.TrimSpace(req.FileID)
	if parentID == "" && targetPath != "/" {
		parentPath := normalizeOpenFamilyPath(parentDirectory(targetPath))
		if parentPath == "." || parentPath == "" {
			parentPath = "/"
		}
		if parentPath == "/" {
			parentID = "0"
		} else {
			resolvedParentID, _, found, resolveErr := a.resolveShareFamilyEntryByPath(session, parentPath, 0)
			if resolveErr != nil {
				return MetadataResult{
					OperationResult: OperationResult{
						Status:  normalizeShareFamilyRequestErrorStatus(resolveErr),
						Message: fmt.Sprintf("%s parent path resolution failed: %v", a.MetaInfo.DisplayName, resolveErr),
						Mode:    "share_family_real_directory",
					},
				}
			}
			if !found {
				return MetadataResult{
					OperationResult: OperationResult{
						OK:      true,
						Status:  "missing",
						Message: fmt.Sprintf("%s did not find the requested parent path.", a.MetaInfo.DisplayName),
						Mode:    "share_family_real_directory",
					},
					Entry: map[string]interface{}{
						"exists":   false,
						"path":     targetPath,
						"provider": a.MetaInfo.Key,
					},
				}
			}
			parentID = resolvedParentID
		}
	}
	if parentID == "" {
		parentID = "0"
	}

	items, listErr := a.listShareFamilyByParent(session, parentID, normalizeOpenFamilyPath(parentDirectory(targetPath)), 0)
	if listErr != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeShareFamilyRequestErrorStatus(listErr),
				Message: fmt.Sprintf("%s metadata list request failed: %v", a.MetaInfo.DisplayName, listErr),
				Mode:    "share_family_real_directory",
			},
		}
	}
	targetName := inferName(targetPath, "")
	for _, item := range items {
		if targetFileID != "" && strings.TrimSpace(stringMapValue(item, "fileId")) == targetFileID {
			return a.shareFamilyMetadataResult(session, parentID, item)
		}
		if targetFileID == "" && strings.TrimSpace(stringMapValue(item, "name")) == targetName {
			return a.shareFamilyMetadataResult(session, parentID, item)
		}
	}

	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "missing",
			Message: fmt.Sprintf("%s did not find the requested path.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_directory",
		},
		Entry: map[string]interface{}{
			"exists":   false,
			"path":     targetPath,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a ShareFamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	session, validation := a.shareFamilyValidatedSession(req.Profile)
	if !validation.OK {
		return validation
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = "0"
	}
	statusCode, payload, requestErr := postShareFamilyJSON(context.Background(), session, buildShareFamilyDriveFilePath(), map[string]interface{}{
		"pdir_fid":      parentID,
		"file_name":     strings.TrimSpace(req.DirName),
		"dir_path":      "",
		"dir_init_lock": false,
	})
	if requestErr != nil {
		return OperationResult{
			Status:  normalizeShareFamilyRequestErrorStatus(requestErr),
			Message: fmt.Sprintf("%s create-dir request failed: %v", a.MetaInfo.DisplayName, requestErr),
			Mode:    "share_family_real_directory",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: fmt.Sprintf("%s rejected the supplied cookie while creating a directory.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_directory",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 || !isShareFamilySuccess(payload) {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("%s create-dir returned HTTP %d.", a.MetaInfo.DisplayName, statusCode),
			Mode:    "share_family_real_directory",
			Payload: payload,
		}
	}

	rawItem, _ := payload["data"].(map[string]interface{})
	if rawItem == nil {
		rawItem = payload
	}
	entry := a.normalizeShareFamilyEntry(rawItem, pathJoin("/", strings.TrimSpace(req.DirName)), parentID)
	entry["isDir"] = true
	entry["type"] = "dir"
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: fmt.Sprintf("%s created the requested directory.", a.MetaInfo.DisplayName),
		Mode:    "share_family_real_directory",
		Payload: entry,
	}
}

func (a ShareFamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	if strings.TrimSpace(req.Profile.Cookie) == "" {
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				Status:  "missing_cookie",
				Message: "Share-family adapter requires a cookie.",
				Mode:    "share_family_real_upload",
			},
		}
	}
	if a.RequirePwdID && firstNonEmptyExtra(req.Profile.Extra, "pwdId", "sharePwdId", "share_id") == "" {
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				Status:  "missing_pwd_id",
				Message: "Share-family adapter requires extra.pwdId.",
				Mode:    "share_family_real_upload",
			},
		}
	}
	candidate := strings.TrimSpace(req.MD5) != "" && req.Size > 0
	message := fmt.Sprintf("%s fast-upload requires md5 and size.", a.MetaInfo.DisplayName)
	if candidate {
		message = fmt.Sprintf("%s fast-upload candidate is available.", a.MetaInfo.DisplayName)
	}
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "share_family_real_upload",
			Payload: map[string]interface{}{
				"requires": []string{"md5", "size"},
			},
		},
		Candidate: candidate,
	}
}

func (a ShareFamilyAdapter) Upload(req UploadRequest) UploadResult {
	session, validation := a.shareFamilyValidatedSession(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: fmt.Sprintf("%s pending_manual items still require manual confirmation.", a.MetaInfo.DisplayName),
				Mode:    "share_family_real_upload",
			},
		}
	}

	localPath := strings.TrimSpace(req.LocalPath)
	if localPath == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("%s upload requires a readable local file.", a.MetaInfo.DisplayName),
				Mode:    "share_family_real_upload",
			},
		}
	}
	info, statErr := os.Stat(localPath)
	if statErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("%s could not stat local file: %v", a.MetaInfo.DisplayName, statErr),
				Mode:    "share_family_real_upload",
			},
		}
	}
	actualMD5, actualSHA1, hashErr := computeShareFamilyLocalHashes(localPath)
	if hashErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("%s could not compute local hashes: %v", a.MetaInfo.DisplayName, hashErr),
				Mode:    "share_family_real_upload",
			},
		}
	}
	if normalizedMD5 := strings.ToLower(strings.TrimSpace(req.MD5)); normalizedMD5 != "" && normalizedMD5 != actualMD5 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_hash_mismatch",
				Message: fmt.Sprintf("%s local md5 does not match the task entry.", a.MetaInfo.DisplayName),
				Mode:    "share_family_real_upload",
			},
		}
	}
	if resumeResult := a.resumeShareFamilyUpload(session, req, localPath, actualMD5, actualSHA1); resumeResult != nil {
		return *resumeResult
	}
	parentID := defaultShareFamilyParentID(req.ParentID)
	resolvedTargetName, conflictAction, conflictNote, conflictErr := a.resolveShareFamilyUploadName(session, parentID, inferName(req.Path, req.Name), req.ConflictPolicy)
	if conflictErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeShareFamilyRequestErrorStatus(conflictErr),
				Message: fmt.Sprintf("%s upload conflict preflight failed: %v", a.MetaInfo.DisplayName, conflictErr),
				Mode:    "share_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	preBody := map[string]interface{}{
		"ccp_hash_update": true,
		"dir_path":        "",
		"file_name":       resolvedTargetName,
		"format_type":     shareFamilyFormatType(localPath),
		"l_created_at":    time.Now().Unix(),
		"l_updated_at":    time.Now().Unix(),
		"pdir_fid":        parentID,
		"size":            info.Size(),
	}
	preStatus, prePayload, preErr := postShareFamilyJSON(context.Background(), session, buildShareFamilyUploadPrePath(), preBody)
	if preErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeShareFamilyRequestErrorStatus(preErr),
				Message: fmt.Sprintf("%s upload/pre request failed: %v", a.MetaInfo.DisplayName, preErr),
				Mode:    "share_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	if preStatus < 200 || preStatus >= 300 || !isShareFamilySuccess(prePayload) {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("%s upload/pre returned HTTP %d.", a.MetaInfo.DisplayName, preStatus),
				Mode:    "share_family_real_upload",
				Payload: prePayload,
			},
			ConflictAction: conflictAction,
		}
	}
	uploadSession := extractShareFamilyUploadSession(prePayload)
	if uploadSession.TaskID == "" || uploadSession.ObjKey == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("%s upload/pre did not return task_id or obj_key.", a.MetaInfo.DisplayName),
				Mode:    "share_family_real_upload",
				Payload: prePayload,
			},
			ConflictAction: conflictAction,
		}
	}
	hashStatus, hashPayload, hashErr := postShareFamilyJSON(context.Background(), session, buildShareFamilyUploadHashPath(), map[string]interface{}{
		"md5":     actualMD5,
		"sha1":    actualSHA1,
		"task_id": uploadSession.TaskID,
	})
	if hashErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeShareFamilyRequestErrorStatus(hashErr),
				Message: fmt.Sprintf("%s update/hash request failed: %v", a.MetaInfo.DisplayName, hashErr),
				Mode:    "share_family_real_upload",
				Payload: prePayload,
			},
			ConflictAction: conflictAction,
		}
	}
	if hashStatus < 200 || hashStatus >= 300 || !isShareFamilySuccess(hashPayload) {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("%s update/hash returned HTTP %d.", a.MetaInfo.DisplayName, hashStatus),
				Mode:    "share_family_real_upload",
				Payload: mergePayloads(prePayload, hashPayload),
			},
			ConflictAction: conflictAction,
		}
	}
	hashData, _ := hashPayload["data"].(map[string]interface{})
	usedBinaryFallback := true
	uploadFallbackPayload := map[string]interface{}{}
	if hashData != nil && boolMapValue(hashData, "finish") {
		usedBinaryFallback = false
	} else {
		var fallbackErr error
		uploadFallbackPayload, fallbackErr = completeShareFamilyBinaryUpload(session, localPath, uploadSession, nil)
		if fallbackErr != nil {
			failurePayload := shareFamilyUploadFailurePayload(uploadSession, parentID, resolvedTargetName, conflictAction, actualMD5, actualSHA1, map[string]interface{}{
				"preUploadResponse": prePayload,
				"hashResponse":      hashPayload,
				"uploadFallback":    uploadFallbackPayload,
			})
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "provider_request_failed",
					Message: fmt.Sprintf("%s binary upload fallback failed: %v", a.MetaInfo.DisplayName, fallbackErr),
					Mode:    "share_family_real_upload",
					Payload: failurePayload,
				},
				ConflictAction: conflictAction,
			}
		}
	}
	finishStatus, finishPayload, finishErr := postShareFamilyJSON(context.Background(), session, buildShareFamilyUploadFinishPath(), map[string]interface{}{
		"obj_key": uploadSession.ObjKey,
		"task_id": uploadSession.TaskID,
	})
	if finishErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeShareFamilyRequestErrorStatus(finishErr),
				Message: fmt.Sprintf("%s upload/finish request failed: %v", a.MetaInfo.DisplayName, finishErr),
				Mode:    "share_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	if finishStatus < 200 || finishStatus >= 300 || !isShareFamilySuccess(finishPayload) {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("%s upload/finish returned HTTP %d.", a.MetaInfo.DisplayName, finishStatus),
				Mode:    "share_family_real_upload",
				Payload: map[string]interface{}{
					"preUploadResponse": prePayload,
					"hashResponse":      hashPayload,
					"finishResponse":    finishPayload,
				},
			},
			ConflictAction: conflictAction,
		}
	}
	finishData, _ := finishPayload["data"].(map[string]interface{})
	fileID := firstNonEmptyString(finishData, "fid", "file_id", "fileId")
	if fileID == "" {
		fileID = uploadSession.FileID
	}
	resolvedName := firstNonEmptyString(finishData, "file_name", "fileName", "name")
	if resolvedName == "" {
		resolvedName = resolvedTargetName
	}
	commonPayload := map[string]interface{}{
		"fileId":             fileID,
		"taskId":             uploadSession.TaskID,
		"objKey":             uploadSession.ObjKey,
		"resolvedTargetName": resolvedName,
		"conflictAction":     conflictAction,
		"usedBinaryFallback": usedBinaryFallback,
		"preUploadResponse":  prePayload,
		"hashResponse":       hashPayload,
		"finishResponse":     finishPayload,
	}
	if usedBinaryFallback {
		commonPayload["uploadFallback"] = uploadFallbackPayload
	}
	verifyMode := "finish_response"
	if usedBinaryFallback {
		verifyMode = "finish_response_after_binary_upload"
	}
	verifyPayload := map[string]interface{}{
		"fileId":             fileID,
		"taskId":             uploadSession.TaskID,
		"objKey":             uploadSession.ObjKey,
		"usedBinaryFallback": usedBinaryFallback,
	}
	commonPayload["verifyMode"] = verifyMode
	commonPayload["verifyOk"] = true
	commonPayload["verifyPayload"] = verifyPayload
	message := fmt.Sprintf("%s fast upload completed through upload/pre + update/hash + upload/finish.", a.MetaInfo.DisplayName)
	if usedBinaryFallback {
		message = fmt.Sprintf("%s fast upload completed through upload/pre + update/hash + binary multipart upload + upload/finish.", a.MetaInfo.DisplayName)
	}
	if conflictNote != "" {
		message += " " + conflictNote
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "share_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: conflictAction,
	}
}

func (a ShareFamilyAdapter) resumeShareFamilyUpload(session shareFamilySession, req UploadRequest, localPath string, actualMD5 string, actualSHA1 string) *UploadResult {
	resume := req.ResumeUpload
	if resume == nil || len(resume.ProviderData) == 0 {
		return nil
	}
	uploadSession, parentID, resolvedTargetName, conflictAction, ok := shareFamilyResumeUploadSession(*resume)
	if !ok {
		return nil
	}
	if resumedMD5 := strings.ToLower(strings.TrimSpace(stringMapValue(resume.ProviderData, "md5"))); resumedMD5 != "" {
		actualMD5 = resumedMD5
	}
	if resumedSHA1 := strings.ToLower(strings.TrimSpace(stringMapValue(resume.ProviderData, "sha1"))); resumedSHA1 != "" {
		actualSHA1 = resumedSHA1
	}
	fallbackPayload, fallbackErr := completeShareFamilyBinaryUpload(session, localPath, uploadSession, resume)
	if fallbackErr != nil {
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("%s resumed binary multipart upload failed: %v", a.MetaInfo.DisplayName, fallbackErr),
				Mode:    "share_family_real_upload",
				Payload: shareFamilyUploadFailurePayload(uploadSession, parentID, resolvedTargetName, conflictAction, actualMD5, actualSHA1, map[string]interface{}{
					"uploadFallback": fallbackPayload,
					"resumedUpload":  true,
				}),
			},
			ConflictAction: conflictAction,
		}
	}
	finishStatus, finishPayload, finishErr := postShareFamilyJSON(context.Background(), session, buildShareFamilyUploadFinishPath(), map[string]interface{}{
		"obj_key": uploadSession.ObjKey,
		"task_id": uploadSession.TaskID,
	})
	if finishErr != nil {
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeShareFamilyRequestErrorStatus(finishErr),
				Message: fmt.Sprintf("%s resumed upload/finish request failed: %v", a.MetaInfo.DisplayName, finishErr),
				Mode:    "share_family_real_upload",
				Payload: shareFamilyUploadFailurePayload(uploadSession, parentID, resolvedTargetName, conflictAction, actualMD5, actualSHA1, map[string]interface{}{
					"uploadFallback": fallbackPayload,
					"resumedUpload":  true,
				}),
			},
			ConflictAction: conflictAction,
		}
	}
	if finishStatus < 200 || finishStatus >= 300 || !isShareFamilySuccess(finishPayload) {
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("%s resumed upload/finish returned HTTP %d.", a.MetaInfo.DisplayName, finishStatus),
				Mode:    "share_family_real_upload",
				Payload: shareFamilyUploadFailurePayload(uploadSession, parentID, resolvedTargetName, conflictAction, actualMD5, actualSHA1, map[string]interface{}{
					"uploadFallback": fallbackPayload,
					"finishResponse": finishPayload,
					"resumedUpload":  true,
				}),
			},
			ConflictAction: conflictAction,
		}
	}
	finishData, _ := finishPayload["data"].(map[string]interface{})
	fileID := firstNonEmptyString(finishData, "fid", "file_id", "fileId")
	if fileID == "" {
		fileID = strings.TrimSpace(uploadSession.FileID)
		if fileID == "" {
			fileID = strings.TrimSpace(resume.FileID)
		}
	}
	finalTargetName := firstNonEmptyString(finishData, "file_name", "fileName", "name")
	if finalTargetName == "" {
		finalTargetName = strings.TrimSpace(resolvedTargetName)
	}
	if finalTargetName == "" {
		finalTargetName = inferName(req.Path, req.Name)
	}
	commonPayload := shareFamilyUploadSuccessPayload(uploadSession, parentID, finalTargetName, conflictAction, actualMD5, actualSHA1, map[string]interface{}{
		"uploadFallback":     fallbackPayload,
		"finishResponse":     finishPayload,
		"resumedUpload":      true,
		"usedBinaryFallback": true,
		"verifyMode":         "finish_response_after_binary_upload",
		"verifyOk":           true,
		"verifyPayload": map[string]interface{}{
			"fileId":             fileID,
			"taskId":             uploadSession.TaskID,
			"objKey":             uploadSession.ObjKey,
			"usedBinaryFallback": true,
		},
	})
	commonPayload["fileId"] = fileID
	return &UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: fmt.Sprintf("%s resumed a cached multipart upload session and completed upload/finish afterwards.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: conflictAction,
	}
}

func (a ShareFamilyAdapter) shareFamilyValidatedSession(profile AuthProfile) (shareFamilySession, OperationResult) {
	session, err := a.newShareFamilySession(profile)
	if err != nil {
		return shareFamilySession{}, OperationResult{
			Status:  normalizeShareFamilySessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "share_family_real_directory",
		}
	}
	statusCode, payload, stoken, requestErr := getShareFamilyToken(context.Background(), session)
	if requestErr != nil {
		return shareFamilySession{}, OperationResult{
			Status:  normalizeShareFamilyRequestErrorStatus(requestErr),
			Message: fmt.Sprintf("%s share token request failed: %v", a.MetaInfo.DisplayName, requestErr),
			Mode:    "share_family_real_directory",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return shareFamilySession{}, OperationResult{
			Status:  "auth_invalid",
			Message: fmt.Sprintf("%s rejected the supplied cookie.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_directory",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(stoken) == "" {
		return shareFamilySession{}, OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("%s share token request did not return a usable stoken.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_directory",
			Payload: payload,
		}
	}
	return session, OperationResult{OK: true}
}

func (a ShareFamilyAdapter) newShareFamilySession(profile AuthProfile) (shareFamilySession, error) {
	cookie := strings.TrimSpace(profile.Cookie)
	if cookie == "" {
		cookie = firstNonEmptyExtra(profile.Extra, "cookie", "cookie_header")
	}
	if cookie == "" {
		return shareFamilySession{}, fmt.Errorf("share-family adapter requires a cookie")
	}
	pwdID := firstNonEmptyExtra(profile.Extra, "pwdId", "sharePwdId", "share_id")
	if a.RequirePwdID && pwdID == "" {
		return shareFamilySession{}, fmt.Errorf("share-family adapter requires extra.pwdId")
	}
	apiEndpoint, driveEndpoint, err := resolveShareFamilyEndpoints(profile)
	if err != nil {
		return shareFamilySession{}, err
	}
	return shareFamilySession{
		APIEndpoint:   apiEndpoint,
		DriveEndpoint: driveEndpoint,
		Cookie:        cookie,
		PwdID:         pwdID,
		Passcode:      firstNonEmptyExtra(profile.Extra, "passcode", "accessCode", "share_passcode"),
		ProviderKey:   a.MetaInfo.Key,
	}, nil
}

func resolveShareFamilyEndpoints(profile AuthProfile) (string, string, error) {
	providerKey := strings.TrimSpace(profile.ProviderKey)
	defaultAPI := quarkAPIBase
	defaultDrive := quarkDriveAPIBase
	if providerKey == "uc" {
		defaultAPI = ucAPIBase
		defaultDrive = ucDriveAPIBase
	}
	apiEndpoint := strings.TrimSpace(profile.Extra["apiEndpoint"])
	if apiEndpoint == "" {
		apiEndpoint = defaultAPI
	}
	driveEndpoint := strings.TrimSpace(profile.Extra["driveApiEndpoint"])
	if driveEndpoint == "" {
		driveEndpoint = defaultDrive
	}
	resolvedAPI, err := normalizeShareFamilyEndpoint(apiEndpoint, "apiEndpoint")
	if err != nil {
		return "", "", err
	}
	resolvedDrive, err := normalizeShareFamilyEndpoint(driveEndpoint, "driveApiEndpoint")
	if err != nil {
		return "", "", err
	}
	return resolvedAPI, resolvedDrive, nil
}

func normalizeShareFamilyEndpoint(raw string, field string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return "", fmt.Errorf("invalid %s: %w", field, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid %s: scheme and host are required", field)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func getShareFamilyToken(ctx context.Context, session shareFamilySession) (int, map[string]interface{}, string, error) {
	statusCode, payload, err := postShareFamilyJSON(ctx, session, "/1/clouddrive/share/sharepage/token", map[string]interface{}{
		"pwd_id":   session.PwdID,
		"passcode": session.Passcode,
	})
	if err != nil {
		return 0, nil, "", err
	}
	return statusCode, payload, extractShareFamilyStoken(payload), nil
}

func extractShareFamilyStoken(payload map[string]interface{}) string {
	if data, _ := payload["data"].(map[string]interface{}); data != nil {
		if token := firstNonEmptyString(data, "stoken", "share_token", "shareToken", "token"); token != "" {
			return token
		}
	}
	return firstNonEmptyString(payload, "stoken", "share_token", "shareToken", "token")
}

func (a ShareFamilyAdapter) resolveShareFamilyEntryByPath(session shareFamilySession, path string, pageSize int) (string, map[string]interface{}, bool, error) {
	normalized := normalizeOpenFamilyPath(path)
	if normalized == "/" {
		return "0", map[string]interface{}{
			"exists":   true,
			"name":     "/",
			"path":     "/",
			"fileId":   "0",
			"parentId": "",
			"isDir":    true,
			"provider": a.MetaInfo.Key,
		}, true, nil
	}
	parts := strings.Split(strings.Trim(normalized, "/"), "/")
	currentID := "0"
	currentPath := "/"
	var currentEntry map[string]interface{}
	for _, part := range parts {
		children, err := a.listShareFamilyByParent(session, currentID, currentPath, pageSize)
		if err != nil {
			return "", nil, false, err
		}
		found := false
		for _, item := range children {
			if strings.TrimSpace(stringMapValue(item, "name")) != strings.TrimSpace(part) {
				continue
			}
			currentEntry = item
			currentID = strings.TrimSpace(stringMapValue(item, "fileId"))
			currentPath = normalizeOpenFamilyPath(pathJoin(currentPath, part))
			found = true
			break
		}
		if !found {
			return "", nil, false, nil
		}
	}
	return currentID, currentEntry, true, nil
}

func (a ShareFamilyAdapter) listShareFamilyByParent(session shareFamilySession, parentID string, basePath string, pageSize int) ([]map[string]interface{}, error) {
	limit := pageSize
	if limit <= 0 {
		limit = 200
	}
	statusCode, tokenPayload, stoken, err := getShareFamilyToken(context.Background(), session)
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return nil, fmt.Errorf("auth_invalid")
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(stoken) == "" {
		return nil, fmt.Errorf("share_token_failed")
	}

	page := 1
	items := make([]map[string]interface{}, 0)
	total := 0
	for {
		query := url.Values{}
		query.Set("pwd_id", session.PwdID)
		query.Set("stoken", stoken)
		query.Set("pdir_fid", defaultShareFamilyParentID(parentID))
		query.Set("force", "0")
		query.Set("_page", fmt.Sprintf("%d", page))
		query.Set("_size", fmt.Sprintf("%d", limit))
		query.Set("_fetch_banner", "0")
		query.Set("_fetch_share", "0")
		query.Set("_fetch_total", "1")
		query.Set("sort", "file_type:asc,file_name:asc")
		query.Set("pr", "ucpro")
		query.Set("fr", "pc")
		statusCode, payload, requestErr := getShareFamilyJSON(context.Background(), session, "/1/clouddrive/share/sharepage/detail?"+query.Encode())
		if requestErr != nil {
			return nil, requestErr
		}
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			return nil, fmt.Errorf("auth_invalid")
		}
		if statusCode < 200 || statusCode >= 300 || !isShareFamilySuccess(payload) {
			return nil, fmt.Errorf("http %d", statusCode)
		}
		pageItems := extractShareFamilyList(payload)
		for _, item := range pageItems {
			items = append(items, a.normalizeShareFamilyEntry(item, pathJoin(basePath, shareFamilyItemName(item)), defaultShareFamilyParentID(parentID)))
		}
		if total == 0 {
			total = extractShareFamilyTotal(payload)
		}
		if len(pageItems) == 0 || len(pageItems) < limit || (total > 0 && len(items) >= total) {
			_ = tokenPayload
			return items, nil
		}
		page++
	}
}

func (a ShareFamilyAdapter) shareFamilyMetadataResult(session shareFamilySession, parentID string, item map[string]interface{}) MetadataResult {
	entry := cloneMap(item)
	if boolMapValue(entry, "isDir") {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: fmt.Sprintf("%s returned live metadata.", a.MetaInfo.DisplayName),
				Mode:    "share_family_real_directory",
			},
			Entry: entry,
		}
	}

	statusCode, md5Map, rawPayload, err := a.fetchShareFamilyMD5Map(session, []map[string]interface{}{item})
	if err != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeShareFamilyRequestErrorStatus(err),
				Message: fmt.Sprintf("%s metadata MD5 request failed: %v", a.MetaInfo.DisplayName, err),
				Mode:    "share_family_real_directory",
				Payload: rawPayload,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("%s metadata MD5 returned HTTP %d.", a.MetaInfo.DisplayName, statusCode),
				Mode:    "share_family_real_directory",
				Payload: rawPayload,
			},
		}
	}
	fileID := strings.TrimSpace(stringMapValue(item, "fileId"))
	if md5Value := md5Map[fileID]; md5Value != "" {
		entry["md5"] = md5Value
		entry["etag"] = md5Value
	}
	entry["parentId"] = defaultShareFamilyParentID(parentID)
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "exists",
			Message: fmt.Sprintf("%s returned live metadata.", a.MetaInfo.DisplayName),
			Mode:    "share_family_real_directory",
		},
		Entry: entry,
	}
}

func (a ShareFamilyAdapter) fetchShareFamilyMD5Map(session shareFamilySession, items []map[string]interface{}) (int, map[string]string, map[string]interface{}, error) {
	fileItems := make([]map[string]interface{}, 0, len(items))
	fids := make([]interface{}, 0, len(items))
	fidTokens := make([]interface{}, 0, len(items))
	for _, item := range items {
		if boolMapValue(item, "isDir") || strings.TrimSpace(stringMapValue(item, "fileId")) == "" {
			continue
		}
		fileItems = append(fileItems, item)
		fids = append(fids, stringMapValue(item, "fileId"))
		fidTokens = append(fidTokens, stringMapValue(item, "fidToken"))
	}
	if len(fileItems) == 0 {
		return 200, map[string]string{}, map[string]interface{}{}, nil
	}
	statusCode, tokenPayload, stoken, err := getShareFamilyToken(context.Background(), session)
	if err != nil {
		return 0, nil, nil, err
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return statusCode, nil, tokenPayload, fmt.Errorf("auth_invalid")
	}
	if statusCode < 200 || statusCode >= 300 || strings.TrimSpace(stoken) == "" {
		return statusCode, nil, tokenPayload, fmt.Errorf("share_token_failed")
	}
	query := url.Values{}
	query.Set("pr", "ucpro")
	query.Set("fr", "pc")
	query.Set("uc_param_str", "")
	statusCode, payload, requestErr := postShareFamilyJSON(context.Background(), session, "/1/clouddrive/file/download?"+query.Encode(), map[string]interface{}{
		"fids":       fids,
		"pwd_id":     session.PwdID,
		"stoken":     stoken,
		"fids_token": fidTokens,
	})
	if requestErr != nil {
		return 0, nil, nil, requestErr
	}
	foundNodes := make([]map[string]interface{}, 0)
	collectShareFamilyDownloadInfoObjects(payload, &foundNodes, 0)
	md5Map := make(map[string]string, len(foundNodes))
	for _, node := range foundNodes {
		md5Value := shareFamilyDownloadInfoMD5(node)
		if md5Value == "" {
			continue
		}
		if fid := shareFamilyItemFID(node); fid != "" {
			md5Map[fid] = md5Value
		}
	}
	return statusCode, md5Map, payload, nil
}

func cloneMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return map[string]interface{}{}
	}
	cloned := make(map[string]interface{}, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func isShareFamilySuccess(payload map[string]interface{}) bool {
	if len(payload) == 0 {
		return false
	}
	if code := firstNonEmptyString(payload, "code", "status"); code == "0" || code == "200" {
		return true
	}
	return false
}

func extractShareFamilyList(payload map[string]interface{}) []map[string]interface{} {
	if data, _ := payload["data"].(map[string]interface{}); data != nil {
		for _, key := range []string{"list", "file_list", "fileList", "items", "records"} {
			if items := mapSliceValue(data, key); len(items) > 0 {
				return items
			}
		}
	}
	for _, key := range []string{"list", "file_list", "fileList", "items", "records"} {
		if items := mapSliceValue(payload, key); len(items) > 0 {
			return items
		}
	}
	return []map[string]interface{}{}
}

func mapSliceValue(values map[string]interface{}, key string) []map[string]interface{} {
	rawItems := interfaceSliceValue(values, key)
	items := make([]map[string]interface{}, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		items = append(items, item)
	}
	return items
}

func extractShareFamilyTotal(payload map[string]interface{}) int {
	candidates := make([]interface{}, 0, 8)
	if data, _ := payload["data"].(map[string]interface{}); data != nil {
		candidates = append(candidates, data["total"], data["totalCount"], data["total_count"], data["count"])
	}
	candidates = append(candidates, payload["total"], payload["totalCount"], payload["total_count"], payload["count"])
	for _, value := range candidates {
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "" || text == "<nil>" {
			continue
		}
		if parsed, err := strconv.Atoi(text); err == nil {
			return parsed
		}
	}
	return 0
}

func shareFamilyItemName(item map[string]interface{}) string {
	return firstNonEmptyString(item, "file_name", "fileName", "name", "title", "server_filename", "filename")
}

func shareFamilyItemFID(item map[string]interface{}) string {
	return firstNonEmptyString(item, "fid", "file_id", "fileId", "share_fid", "shareFileId", "id")
}

func shareFamilyItemFIDToken(item map[string]interface{}) string {
	return firstNonEmptyString(item, "share_fid_token", "fid_token", "fidToken", "file_token", "fileToken", "token")
}

func shareFamilyItemSize(item map[string]interface{}) int64 {
	for _, key := range []string{"size", "file_size", "fileSize", "bytes"} {
		if value := int64MapValue(item, key); value > 0 {
			return value
		}
	}
	return 0
}

func shareFamilyItemIsDir(item map[string]interface{}) bool {
	for _, key := range []string{"dir", "isdir", "is_dir", "isDir", "folder", "is_folder", "isFolder"} {
		if boolMapValue(item, key) {
			return true
		}
		text := strings.ToLower(strings.TrimSpace(stringMapValue(item, key)))
		if text == "0" || text == "false" {
			return false
		}
	}
	typeText := strings.ToLower(firstNonEmptyString(item, "type", "kind", "file_type", "fileType", "obj_category", "category"))
	if strings.Contains(typeText, "dir") || strings.Contains(typeText, "folder") || strings.Contains(typeText, "directory") {
		return true
	}
	if strings.Contains(typeText, "file") || strings.Contains(typeText, "video") || strings.Contains(typeText, "audio") || strings.Contains(typeText, "image") || strings.Contains(typeText, "doc") {
		return false
	}
	return false
}

func (a ShareFamilyAdapter) normalizeShareFamilyEntry(raw map[string]interface{}, path string, parentID string) map[string]interface{} {
	isDir := shareFamilyItemIsDir(raw)
	entry := map[string]interface{}{
		"exists":   true,
		"name":     shareFamilyItemName(raw),
		"path":     normalizeOpenFamilyPath(path),
		"fileId":   shareFamilyItemFID(raw),
		"fidToken": shareFamilyItemFIDToken(raw),
		"parentId": defaultShareFamilyParentID(parentID),
		"type":     "file",
		"isDir":    isDir,
		"size":     shareFamilyItemSize(raw),
		"md5":      "",
		"etag":     "",
		"provider": a.MetaInfo.Key,
		"raw":      raw,
	}
	if isDir {
		entry["type"] = "dir"
	}
	return entry
}

func collectShareFamilyDownloadInfoObjects(node interface{}, out *[]map[string]interface{}, depth int) {
	if node == nil || depth > 8 {
		return
	}
	switch typed := node.(type) {
	case []interface{}:
		for _, item := range typed {
			collectShareFamilyDownloadInfoObjects(item, out, depth+1)
		}
	case map[string]interface{}:
		hasFID := false
		for _, key := range []string{"fid", "file_id", "fileId", "share_fid", "shareFileId", "id"} {
			if strings.TrimSpace(stringMapValue(typed, key)) != "" {
				hasFID = true
				break
			}
		}
		hasHash := shareFamilyDownloadInfoMD5(typed) != ""
		if hasFID || hasHash {
			*out = append(*out, typed)
		}
		for _, value := range typed {
			switch value.(type) {
			case map[string]interface{}, []interface{}:
				collectShareFamilyDownloadInfoObjects(value, out, depth+1)
			}
		}
	}
}

func shareFamilyDownloadInfoMD5(item map[string]interface{}) string {
	for _, key := range []string{"md5", "file_md5", "fileMd5", "hash", "etag", "content_hash", "contentHash"} {
		value := strings.ToLower(strings.TrimSpace(stringMapValue(item, key)))
		if len(value) == 32 {
			valid := true
			for _, ch := range value {
				if !strings.ContainsRune("0123456789abcdef", ch) {
					valid = false
					break
				}
			}
			if valid {
				return value
			}
		}
	}
	return ""
}

func defaultShareFamilyParentID(parentID string) string {
	if strings.TrimSpace(parentID) == "" {
		return "0"
	}
	return strings.TrimSpace(parentID)
}

func buildShareFamilyDriveFilePath() string {
	query := url.Values{}
	query.Set("pr", "ucpro")
	query.Set("fr", "pc")
	query.Set("uc_param_str", "")
	return "/1/clouddrive/file?" + query.Encode()
}

func buildShareFamilyUploadPrePath() string {
	query := url.Values{}
	query.Set("pr", "ucpro")
	query.Set("fr", "pc")
	query.Set("uc_param_str", "")
	return "/1/clouddrive/file/upload/pre?" + query.Encode()
}

func buildShareFamilyUploadHashPath() string {
	query := url.Values{}
	query.Set("pr", "ucpro")
	query.Set("fr", "pc")
	query.Set("uc_param_str", "")
	return "/1/clouddrive/file/update/hash?" + query.Encode()
}

func buildShareFamilyUploadFinishPath() string {
	query := url.Values{}
	query.Set("pr", "ucpro")
	query.Set("fr", "pc")
	query.Set("uc_param_str", "")
	return "/1/clouddrive/file/upload/finish?" + query.Encode()
}

func buildShareFamilyUploadAuthPath() string {
	query := url.Values{}
	query.Set("pr", "ucpro")
	query.Set("fr", "pc")
	query.Set("uc_param_str", "")
	return "/1/clouddrive/file/upload/auth?" + query.Encode()
}

func getShareFamilyJSON(ctx context.Context, session shareFamilySession, requestPath string) (int, map[string]interface{}, error) {
	endpoint := strings.TrimRight(session.APIEndpoint, "/") + requestPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	applyShareFamilyHeaders(req, session, false)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeShareFamilyJSON(resp)
}

func postShareFamilyJSON(ctx context.Context, session shareFamilySession, requestPath string, body interface{}) (int, map[string]interface{}, error) {
	payload := []byte("{}")
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		payload = raw
	}
	base := strings.TrimRight(session.APIEndpoint, "/")
	if strings.HasPrefix(requestPath, "/1/clouddrive/file") {
		base = strings.TrimRight(session.DriveEndpoint, "/")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+requestPath, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	applyShareFamilyHeaders(req, session, true)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeShareFamilyJSON(resp)
}

func decodeShareFamilyJSON(resp *http.Response) (int, map[string]interface{}, error) {
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	if len(bodyBytes) == 0 {
		return resp.StatusCode, map[string]interface{}{}, nil
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("decode provider json: %w", err)
	}
	return resp.StatusCode, payload, nil
}

func applyShareFamilyHeaders(req *http.Request, session shareFamilySession, jsonBody bool) {
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Cookie", session.Cookie)
	if session.ProviderKey == "uc" {
		req.Header.Set("Referer", "https://drive.uc.cn/")
		req.Header.Set("Origin", "https://drive.uc.cn")
		req.Header.Set("User-Agent", "Mozilla/5.0 UcCloudDrivePC/1.0")
	} else {
		req.Header.Set("Referer", "https://pan.quark.cn/")
		req.Header.Set("Origin", "https://pan.quark.cn")
		req.Header.Set("User-Agent", "Mozilla/5.0 QuarkCloudDrivePC/1.0")
	}
	if jsonBody {
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
	}
}

func normalizeShareFamilySessionErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "cookie") {
		return "missing_cookie"
	}
	if strings.Contains(err.Error(), "pwdId") {
		return "missing_pwd_id"
	}
	if strings.Contains(err.Error(), "invalid apiEndpoint") || strings.Contains(err.Error(), "invalid driveApiEndpoint") {
		return "invalid_provider_endpoint"
	}
	return "provider_request_failed"
}

func normalizeShareFamilyRequestErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "auth_invalid") || strings.Contains(err.Error(), "share_token_failed") {
		return "auth_invalid"
	}
	if strings.Contains(err.Error(), "parent_path_not_found") {
		return "path_not_found"
	}
	return "provider_request_failed"
}

func computeShareFamilyLocalHashes(localPath string) (string, string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = file.Close() }()
	md5Hasher := md5.New()
	sha1Hasher := sha1.New()
	buffer := make([]byte, 1024*1024)
	for {
		readBytes, readErr := file.Read(buffer)
		if readBytes > 0 {
			chunk := buffer[:readBytes]
			_, _ = md5Hasher.Write(chunk)
			_, _ = sha1Hasher.Write(chunk)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", "", readErr
		}
	}
	return fmt.Sprintf("%x", md5Hasher.Sum(nil)), fmt.Sprintf("%x", sha1Hasher.Sum(nil)), nil
}

func shareFamilyFormatType(localPath string) string {
	switch strings.ToLower(strings.TrimSpace(strings.TrimPrefix(filepath.Ext(localPath), "."))) {
	case "mp4", "mkv", "avi", "mov", "flv", "wmv", "m4v":
		return "video"
	case "mp3", "flac", "wav", "aac", "m4a", "ogg":
		return "audio"
	case "jpg", "jpeg", "png", "gif", "webp", "bmp", "heic":
		return "image"
	case "txt", "md", "pdf", "doc", "docx", "xls", "xlsx", "ppt", "pptx":
		return "doc"
	default:
		return "file"
	}
}

func (a ShareFamilyAdapter) resolveShareFamilyUploadName(session shareFamilySession, parentID string, targetName string, policy ConflictPolicy) (string, string, string, error) {
	items, err := a.listShareFamilyByParent(session, parentID, "/", 200)
	if err != nil {
		return targetName, "conflict_check_unavailable", fmt.Sprintf("Could not verify same-name conflicts before %s upload, so the original file name was kept.", a.MetaInfo.DisplayName), err
	}
	existing := map[string]bool{}
	for _, item := range items {
		name := strings.TrimSpace(stringMapValue(item, "name"))
		if name != "" {
			existing[name] = true
		}
	}
	if !existing[targetName] {
		return targetName, "none", "", nil
	}
	stem, suffix := splitShareFamilyName(targetName)
	candidate := targetName
	index := 1
	for existing[candidate] {
		candidate = fmt.Sprintf("%s (%d)%s", stem, index, suffix)
		index++
	}
	if policy == ConflictPolicyAutoRenameNew {
		return candidate, "auto_rename_new", fmt.Sprintf("A same-name file already exists under the target path, so %s auto-renamed the new file.", a.MetaInfo.DisplayName), nil
	}
	return candidate, "overwrite_downgraded_to_auto_rename", fmt.Sprintf("The requested overwrite policy was downgraded because the current %s upload path does not support verified in-place overwrite.", a.MetaInfo.DisplayName), nil
}

func splitShareFamilyName(name string) (string, string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "file", ""
	}
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	if stem == "" {
		stem = "file"
	}
	return stem, ext
}

func extractShareFamilyUploadSession(prePayload map[string]interface{}) shareFamilyUploadSession {
	data, _ := prePayload["data"].(map[string]interface{})
	meta, _ := data["metadata"].(map[string]interface{})
	authInfo, _ := data["auth_info"].(map[string]interface{})
	if len(authInfo) == 0 {
		authInfo, _ = meta["auth_info"].(map[string]interface{})
	}
	callback, _ := data["callback"].(map[string]interface{})
	if len(callback) == 0 {
		callback, _ = meta["callback"].(map[string]interface{})
	}
	return shareFamilyUploadSession{
		AuthInfo:  authInfo,
		Bucket:    firstNonEmptyString(data, "bucket"),
		ObjKey:    firstNonEmptyString(data, "obj_key", "objKey"),
		UploadID:  firstNonEmptyString(data, "upload_id", "uploadId"),
		UploadURL: firstNonEmptyString(data, "upload_url", "uploadUrl"),
		Callback:  callback,
		PartSize:  int64MapValue(data, "part_size"),
		TaskID:    firstNonEmptyString(data, "task_id", "taskId"),
		FileID:    firstNonEmptyString(data, "fid", "file_id", "fileId"),
	}
}

func completeShareFamilyBinaryUpload(session shareFamilySession, localPath string, uploadSession shareFamilyUploadSession, resume *ResumeUpload) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"uploadSession": map[string]interface{}{
			"bucket":    uploadSession.Bucket,
			"objKey":    uploadSession.ObjKey,
			"uploadId":  uploadSession.UploadID,
			"uploadUrl": uploadSession.UploadURL,
		},
	}
	if len(uploadSession.AuthInfo) == 0 || uploadSession.Bucket == "" || uploadSession.ObjKey == "" || uploadSession.UploadID == "" || uploadSession.UploadURL == "" || len(uploadSession.Callback) == 0 {
		return payload, fmt.Errorf("missing upload session")
	}
	uploadBase := normalizeShareFamilyUploadBase(uploadSession.UploadURL, uploadSession.Bucket)
	contentType := shareFamilyGuessContentType(localPath)
	partSize := uploadSession.PartSize
	if partSize <= 0 {
		info, err := os.Stat(localPath)
		if err != nil {
			return payload, err
		}
		partSize = info.Size()
	}
	file, err := os.Open(localPath)
	if err != nil {
		return payload, err
	}
	defer func() { _ = file.Close() }()
	totalParts := shareFamilyTotalPartCount(localPath, partSize)
	if totalParts <= 0 {
		totalParts = 1
	}
	uploadedParts := shareFamilyUploadedPartsForResume(resume)
	uploadedPartsMap := shareFamilyUploadedPartsMap(uploadedParts)
	startPartNumber := shareFamilyResumeStartPart(resume, uploadedParts)
	if startPartNumber <= 0 {
		startPartNumber = 1
	}
	partEtags := shareFamilyPartEtags(totalParts, uploadedPartsMap)
	for partNumber := 1; partNumber <= totalParts; partNumber++ {
		if partNumber < startPartNumber && uploadedPartsMap[partNumber] != "" {
			continue
		}
		partBytes, readErr := shareFamilyReadPart(file, partSize, partNumber)
		if readErr != nil {
			return payload, readErr
		}
		if len(partBytes) == 0 {
			break
		}
		ossDate := time.Now().UTC().Format(http.TimeFormat)
		authMeta := buildShareFamilyPartAuthMeta(contentType, ossDate, uploadSession.Bucket, uploadSession.ObjKey, partNumber, uploadSession.UploadID)
		authStatus, authPayload, authErr := postShareFamilyJSON(context.Background(), session, buildShareFamilyUploadAuthPath(), map[string]interface{}{
			"auth_info": uploadSession.AuthInfo,
			"auth_meta": authMeta,
			"task_id":   uploadSession.TaskID,
		})
		if authErr != nil {
			shareFamilyApplyUploadProgressPayload(payload, totalParts, uploadedParts, partNumber)
			return payload, authErr
		}
		if authStatus < 200 || authStatus >= 300 || !isShareFamilySuccess(authPayload) {
			shareFamilyApplyUploadProgressPayload(payload, totalParts, uploadedParts, partNumber)
			return payload, fmt.Errorf("part upload auth failed")
		}
		authData, _ := authPayload["data"].(map[string]interface{})
		authKey := firstNonEmptyString(authData, "auth_key", "authKey")
		if authKey == "" {
			shareFamilyApplyUploadProgressPayload(payload, totalParts, uploadedParts, partNumber)
			return payload, fmt.Errorf("part upload missing auth key")
		}
		etag, putErr := putShareFamilyOSSPart(uploadBase, uploadSession.ObjKey, uploadSession.UploadID, partNumber, authKey, contentType, ossDate, partBytes, session)
		if putErr != nil {
			shareFamilyApplyUploadProgressPayload(payload, totalParts, uploadedParts, partNumber)
			return payload, putErr
		}
		partEtags[partNumber-1] = etag
		uploadedParts = shareFamilyUpsertUploadedPart(uploadedParts, partNumber, etag)
		uploadedPartsMap[partNumber] = etag
	}
	partEtags = shareFamilyPartEtags(totalParts, uploadedPartsMap)
	if shareFamilyHasMissingPartEtags(partEtags) {
		shareFamilyApplyUploadProgressPayload(payload, totalParts, uploadedParts, shareFamilyFirstMissingPart(partEtags))
		return payload, fmt.Errorf("missing multipart etag before commit")
	}
	payload["partCount"] = len(partEtags)
	payload["partEtags"] = partEtags
	payload["uploadedPartCount"] = len(uploadedParts)
	payload["uploadedParts"] = uploadedParts
	payload["nextPartNumber"] = len(partEtags) + 1
	if resume != nil {
		payload["resumedUpload"] = true
	}
	commitXML := buildShareFamilyCommitXML(partEtags)
	commitMD5Raw := md5.Sum([]byte(commitXML))
	commitMD5 := base64.StdEncoding.EncodeToString(commitMD5Raw[:])
	callbackBytes, _ := json.Marshal(uploadSession.Callback)
	callbackB64 := base64.StdEncoding.EncodeToString(callbackBytes)
	commitDate := time.Now().UTC().Format(http.TimeFormat)
	commitMeta := buildShareFamilyCommitAuthMeta(commitMD5, callbackB64, commitDate, uploadSession.Bucket, uploadSession.ObjKey, uploadSession.UploadID)
	commitAuthStatus, commitAuthPayload, commitAuthErr := postShareFamilyJSON(context.Background(), session, buildShareFamilyUploadAuthPath(), map[string]interface{}{
		"auth_info": uploadSession.AuthInfo,
		"auth_meta": commitMeta,
		"task_id":   uploadSession.TaskID,
	})
	payload["commitAuthResponse"] = commitAuthPayload
	if commitAuthErr != nil {
		return payload, commitAuthErr
	}
	if commitAuthStatus < 200 || commitAuthStatus >= 300 || !isShareFamilySuccess(commitAuthPayload) {
		return payload, fmt.Errorf("commit auth failed")
	}
	commitAuthData, _ := commitAuthPayload["data"].(map[string]interface{})
	commitAuthKey := firstNonEmptyString(commitAuthData, "auth_key", "authKey")
	if commitAuthKey == "" {
		return payload, fmt.Errorf("commit missing auth key")
	}
	commitStatus, commitErr := postShareFamilyOSSCommit(uploadBase, uploadSession.ObjKey, uploadSession.UploadID, commitAuthKey, commitMD5, callbackB64, commitDate, commitXML, session)
	payload["commitStatus"] = commitStatus
	if commitErr != nil {
		return payload, commitErr
	}
	return payload, nil
}

func shareFamilyResumeUploadSession(resume ResumeUpload) (shareFamilyUploadSession, string, string, string, bool) {
	if len(resume.ProviderData) == 0 {
		return shareFamilyUploadSession{}, "", "", "", false
	}
	authInfo, _ := resume.ProviderData["authInfo"].(map[string]interface{})
	callback, _ := resume.ProviderData["callback"].(map[string]interface{})
	uploadSession := shareFamilyUploadSession{
		AuthInfo:  copyPayloadMap(authInfo),
		Bucket:    stringMapValue(resume.ProviderData, "bucket"),
		ObjKey:    stringMapValue(resume.ProviderData, "objKey"),
		UploadID:  firstNonEmptyValue(stringMapValue(resume.ProviderData, "uploadId"), resume.UploadID),
		UploadURL: stringMapValue(resume.ProviderData, "uploadUrl"),
		Callback:  copyPayloadMap(callback),
		PartSize:  int64MapValue(resume.ProviderData, "partSize"),
		TaskID:    stringMapValue(resume.ProviderData, "taskId"),
		FileID:    firstNonEmptyValue(stringMapValue(resume.ProviderData, "fileId"), resume.FileID),
	}
	if len(uploadSession.AuthInfo) == 0 || len(uploadSession.Callback) == 0 || uploadSession.Bucket == "" || uploadSession.ObjKey == "" || uploadSession.UploadID == "" || uploadSession.UploadURL == "" || uploadSession.TaskID == "" {
		return shareFamilyUploadSession{}, "", "", "", false
	}
	return uploadSession,
		stringMapValue(resume.ProviderData, "parentId"),
		stringMapValue(resume.ProviderData, "resolvedTargetName"),
		stringMapValue(resume.ProviderData, "conflictAction"),
		true
}

func shareFamilyUploadFailurePayload(uploadSession shareFamilyUploadSession, parentID string, resolvedTargetName string, conflictAction string, actualMD5 string, actualSHA1 string, extra map[string]interface{}) map[string]interface{} {
	return shareFamilyUploadSuccessPayload(uploadSession, parentID, resolvedTargetName, conflictAction, actualMD5, actualSHA1, extra)
}

func shareFamilyUploadSuccessPayload(uploadSession shareFamilyUploadSession, parentID string, resolvedTargetName string, conflictAction string, actualMD5 string, actualSHA1 string, extra map[string]interface{}) map[string]interface{} {
	base := map[string]interface{}{
		"fileId":             uploadSession.FileID,
		"taskId":             uploadSession.TaskID,
		"objKey":             uploadSession.ObjKey,
		"uploadId":           uploadSession.UploadID,
		"resolvedTargetName": resolvedTargetName,
		"conflictAction":     conflictAction,
		"providerData":       shareFamilyResumeProviderData(uploadSession, parentID, resolvedTargetName, conflictAction, actualMD5, actualSHA1),
	}
	if fallbackPayload, ok := extra["uploadFallback"].(map[string]interface{}); ok {
		base = mergeMaps(base, fallbackPayload)
	}
	return mergeMaps(base, extra)
}

func shareFamilyResumeProviderData(uploadSession shareFamilyUploadSession, parentID string, resolvedTargetName string, conflictAction string, actualMD5 string, actualSHA1 string) map[string]interface{} {
	return map[string]interface{}{
		"authInfo":           copyPayloadMap(uploadSession.AuthInfo),
		"bucket":             uploadSession.Bucket,
		"objKey":             uploadSession.ObjKey,
		"uploadId":           uploadSession.UploadID,
		"uploadUrl":          uploadSession.UploadURL,
		"callback":           copyPayloadMap(uploadSession.Callback),
		"partSize":           uploadSession.PartSize,
		"taskId":             uploadSession.TaskID,
		"fileId":             uploadSession.FileID,
		"parentId":           parentID,
		"resolvedTargetName": resolvedTargetName,
		"conflictAction":     conflictAction,
		"md5":                actualMD5,
		"sha1":               actualSHA1,
	}
}

func shareFamilyTotalPartCount(localPath string, partSize int64) int {
	info, err := os.Stat(localPath)
	if err != nil {
		return 0
	}
	size := info.Size()
	if size <= 0 {
		return 1
	}
	if partSize <= 0 {
		partSize = size
	}
	return int((size + partSize - 1) / partSize)
}

func shareFamilyUploadedPartsForResume(resume *ResumeUpload) []map[string]interface{} {
	if resume == nil || len(resume.UploadedParts) == 0 {
		return nil
	}
	out := make([]map[string]interface{}, 0, len(resume.UploadedParts))
	for _, item := range resume.UploadedParts {
		out = append(out, copyPayloadMap(item))
	}
	return out
}

func shareFamilyUploadedPartsMap(parts []map[string]interface{}) map[int]string {
	out := make(map[int]string, len(parts))
	for _, item := range parts {
		partNumber := intMapValue(item, "partNumber")
		if partNumber <= 0 {
			partNumber = intMapValue(item, "part_number")
		}
		etag := firstNonEmptyValue(stringMapValue(item, "etag"), stringMapValue(item, "eTag"))
		if partNumber > 0 && etag != "" {
			out[partNumber] = etag
		}
	}
	return out
}

func shareFamilyResumeStartPart(resume *ResumeUpload, uploadedParts []map[string]interface{}) int {
	if resume == nil {
		return 1
	}
	if resume.FailedPartNumber > 0 {
		return resume.FailedPartNumber
	}
	if resume.NextPartNumber > 0 {
		return resume.NextPartNumber
	}
	highest := 0
	for _, item := range uploadedParts {
		partNumber := intMapValue(item, "partNumber")
		if partNumber <= 0 {
			partNumber = intMapValue(item, "part_number")
		}
		if partNumber > highest {
			highest = partNumber
		}
	}
	if highest > 0 {
		return highest + 1
	}
	return 1
}

func shareFamilyReadPart(file *os.File, partSize int64, partNumber int) ([]byte, error) {
	if partSize <= 0 || partNumber <= 0 {
		return nil, nil
	}
	offset := int64(partNumber-1) * partSize
	partBytes := make([]byte, partSize)
	readBytes, err := file.ReadAt(partBytes, offset)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return nil, err
	}
	return partBytes[:readBytes], nil
}

func shareFamilyUpsertUploadedPart(parts []map[string]interface{}, partNumber int, etag string) []map[string]interface{} {
	replaced := false
	out := make([]map[string]interface{}, 0, len(parts)+1)
	for _, item := range parts {
		currentPart := intMapValue(item, "partNumber")
		if currentPart <= 0 {
			currentPart = intMapValue(item, "part_number")
		}
		if currentPart == partNumber {
			out = append(out, map[string]interface{}{
				"partNumber": partNumber,
				"etag":       etag,
			})
			replaced = true
			continue
		}
		out = append(out, copyPayloadMap(item))
	}
	if !replaced {
		out = append(out, map[string]interface{}{
			"partNumber": partNumber,
			"etag":       etag,
		})
	}
	return shareFamilySortUploadedParts(out)
}

func shareFamilySortUploadedParts(parts []map[string]interface{}) []map[string]interface{} {
	if len(parts) <= 1 {
		return parts
	}
	out := make([]map[string]interface{}, len(parts))
	copy(out, parts)
	for i := 0; i < len(out)-1; i++ {
		for j := i + 1; j < len(out); j++ {
			left := intMapValue(out[i], "partNumber")
			if left <= 0 {
				left = intMapValue(out[i], "part_number")
			}
			right := intMapValue(out[j], "partNumber")
			if right <= 0 {
				right = intMapValue(out[j], "part_number")
			}
			if right < left {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func shareFamilyPartEtags(totalParts int, uploadedParts map[int]string) []string {
	partEtags := make([]string, totalParts)
	for partNumber, etag := range uploadedParts {
		if partNumber <= 0 || partNumber > totalParts {
			continue
		}
		partEtags[partNumber-1] = etag
	}
	return partEtags
}

func shareFamilyHasMissingPartEtags(partEtags []string) bool {
	for _, item := range partEtags {
		if strings.TrimSpace(item) == "" {
			return true
		}
	}
	return false
}

func shareFamilyFirstMissingPart(partEtags []string) int {
	for idx, item := range partEtags {
		if strings.TrimSpace(item) == "" {
			return idx + 1
		}
	}
	return len(partEtags) + 1
}

func shareFamilyApplyUploadProgressPayload(payload map[string]interface{}, totalParts int, uploadedParts []map[string]interface{}, failedPartNumber int) {
	payload["partCount"] = totalParts
	payload["uploadedPartCount"] = len(uploadedParts)
	payload["uploadedParts"] = uploadedParts
	payload["failedPartNumber"] = failedPartNumber
	payload["nextPartNumber"] = failedPartNumber
}

func copyPayloadMap(values map[string]interface{}) map[string]interface{} {
	if values == nil {
		return nil
	}
	out := make(map[string]interface{}, len(values))
	for key, value := range values {
		if nested, ok := value.(map[string]interface{}); ok {
			out[key] = copyPayloadMap(nested)
			continue
		}
		out[key] = value
	}
	return out
}

func intMapValue(values map[string]interface{}, key string) int {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return value
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err == nil {
			return parsed
		}
	}
	return 0
}

func firstNonEmptyValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeShareFamilyUploadBase(uploadURL string, bucket string) string {
	parsed, err := url.Parse(strings.TrimSpace(uploadURL))
	if err == nil && parsed.Host != "" {
		scheme := parsed.Scheme
		if scheme == "" {
			scheme = "https"
		}
		if strings.Contains(parsed.Host, "127.0.0.1") || strings.Contains(parsed.Host, "localhost") {
			return scheme + "://" + parsed.Host
		}
		if strings.HasPrefix(parsed.Host, bucket+".") {
			return scheme + "://" + parsed.Host
		}
		return scheme + "://" + bucket + "." + parsed.Host
	}
	scheme := "https"
	if strings.HasPrefix(strings.TrimSpace(uploadURL), "http://") {
		scheme = "http"
	}
	host := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(uploadURL, "https://"), "http://"))
	host = strings.Trim(host, "/")
	if strings.Contains(host, "127.0.0.1") || strings.Contains(host, "localhost") {
		return scheme + "://" + host
	}
	if strings.HasPrefix(host, bucket+".") {
		return scheme + "://" + host
	}
	return scheme + "://" + bucket + "." + host
}

func shareFamilyGuessContentType(localPath string) string {
	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(localPath)))
	if contentType == "" {
		return "application/octet-stream"
	}
	return contentType
}

func buildShareFamilyPartAuthMeta(contentType string, ossDate string, bucket string, objKey string, partNumber int, uploadID string) string {
	return "PUT\n\n" + contentType + "\n" + ossDate + "\n" +
		"x-oss-date:" + ossDate + "\n" +
		"x-oss-user-agent:aliyun-sdk-js/6.6.1 Chrome 98.0.4758.80 on Windows 10 64-bit\n" +
		"/" + bucket + "/" + strings.TrimLeft(objKey, "/") + "?partNumber=" + strconv.Itoa(partNumber) + "&uploadId=" + uploadID
}

func putShareFamilyOSSPart(uploadBase string, objKey string, uploadID string, partNumber int, authorization string, contentType string, ossDate string, partBytes []byte, session shareFamilySession) (string, error) {
	query := url.Values{}
	query.Set("partNumber", strconv.Itoa(partNumber))
	query.Set("uploadId", uploadID)
	requestURL := strings.TrimRight(uploadBase, "/") + "/" + strings.TrimLeft(objKey, "/") + "?" + query.Encode()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, requestURL, bytes.NewReader(partBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-oss-date", ossDate)
	req.Header.Set("x-oss-user-agent", "aliyun-sdk-js/6.6.1 Chrome 98.0.4758.80 on Windows 10 64-bit")
	if session.ProviderKey == "uc" {
		req.Header.Set("Referer", "https://drive.uc.cn/")
	} else {
		req.Header.Set("Referer", "https://pan.quark.cn/")
	}
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("oss part upload returned HTTP %d", resp.StatusCode)
	}
	return strings.Trim(strings.TrimSpace(resp.Header.Get("ETag")), "\""), nil
}

func buildShareFamilyCommitXML(partEtags []string) string {
	lines := []string{`<?xml version="1.0" encoding="UTF-8"?>`, "<CompleteMultipartUpload>"}
	for index, etag := range partEtags {
		lines = append(lines,
			"<Part>",
			fmt.Sprintf("<PartNumber>%d</PartNumber>", index+1),
			fmt.Sprintf("<ETag>%s</ETag>", etag),
			"</Part>",
		)
	}
	lines = append(lines, "</CompleteMultipartUpload>")
	return strings.Join(lines, "\n")
}

func buildShareFamilyCommitAuthMeta(contentMD5 string, callbackB64 string, ossDate string, bucket string, objKey string, uploadID string) string {
	return "POST\n" + contentMD5 + "\napplication/xml\n" + ossDate + "\n" +
		"x-oss-callback:" + callbackB64 + "\n" +
		"x-oss-date:" + ossDate + "\n" +
		"x-oss-user-agent:aliyun-sdk-js/6.6.1 Chrome 98.0.4758.80 on Windows 10 64-bit\n" +
		"/" + bucket + "/" + strings.TrimLeft(objKey, "/") + "?uploadId=" + uploadID
}

func postShareFamilyOSSCommit(uploadBase string, objKey string, uploadID string, authorization string, contentMD5 string, callbackB64 string, ossDate string, body string, session shareFamilySession) (int, error) {
	query := url.Values{}
	query.Set("uploadId", uploadID)
	requestURL := strings.TrimRight(uploadBase, "/") + "/" + strings.TrimLeft(objKey, "/") + "?" + query.Encode()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, requestURL, strings.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("Content-MD5", contentMD5)
	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("x-oss-callback", callbackB64)
	req.Header.Set("x-oss-date", ossDate)
	req.Header.Set("x-oss-user-agent", "aliyun-sdk-js/6.6.1 Chrome 98.0.4758.80 on Windows 10 64-bit")
	if session.ProviderKey == "uc" {
		req.Header.Set("Referer", "https://drive.uc.cn/")
	} else {
		req.Header.Set("Referer", "https://pan.quark.cn/")
	}
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, nil
}
