package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	xunleiAPIBase  = "https://api-pan.xunlei.com"
	xunleiClientID = "Xqp0kJBXWhwaTpB6"
	pikpakAPIBase  = "https://api-drive.mypikpak.com"
)

type HashFamilyAdapter struct {
	StaticAdapter
}

type hashFamilySession struct {
	BaseEndpoint  string
	Authorization string
	DeviceID      string
	CaptchaToken  string
	ClientID      string
	ProviderKey   string
}

type hashFamilyResumableSession struct {
	Bucket            string
	Key               string
	EndpointURL       string
	EndpointHost      string
	AccessKeyID       string
	AccessKeySecret   string
	SecurityToken     string
	UsePathStyle      bool
	OriginalResumable map[string]interface{}
}

var hashFamilyResumableUploader = uploadHashFamilyResumableBinary

func NewHashFamilyAdapter(meta Provider, capability CapabilitySet) Adapter {
	return HashFamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
	}
}

func (a HashFamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	if hashFamilyAuthorizationHeader(profile) == "" {
		return OperationResult{
			Status:  "missing_access_token",
			Message: "Hash-family adapter requires a token.",
			Mode:    "hash_family_placeholder",
		}
	}
	if a.MetaInfo.Key == "xunlei" {
		return a.validateXunleiAuth(profile)
	}
	if a.MetaInfo.Key == "pikpak" {
		return a.validatePikPakAuth(profile)
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
	if a.MetaInfo.Key == "xunlei" {
		return a.listXunlei(req)
	}
	if a.MetaInfo.Key == "pikpak" {
		return a.listPikPak(req)
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
	if a.MetaInfo.Key == "xunlei" {
		return a.metadataXunlei(req)
	}
	if a.MetaInfo.Key == "pikpak" {
		return a.metadataPikPak(req)
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
	if a.MetaInfo.Key == "xunlei" {
		return a.createDirXunlei(req)
	}
	if a.MetaInfo.Key == "pikpak" {
		return a.createDirPikPak(req)
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
	if a.MetaInfo.Key == "xunlei" {
		candidate := strings.TrimSpace(req.GCID) != "" && req.Size > 0
		message := "Xunlei fast-upload requires gcid and size."
		if candidate {
			message = "Xunlei fast-upload candidate is available."
		}
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: message,
				Mode:    "hash_family_real_upload",
				Payload: map[string]interface{}{
					"requires": []string{"gcid", "size"},
				},
			},
			Candidate: candidate,
		}
	}
	if a.MetaInfo.Key == "pikpak" {
		candidate := strings.TrimSpace(req.GCID) != "" && req.Size > 0
		message := "PikPak fast-upload requires gcid and size."
		if candidate {
			message = "PikPak fast-upload candidate is available."
		}
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: message,
				Mode:    "hash_family_real_upload",
				Payload: map[string]interface{}{
					"requires": []string{"gcid", "size"},
				},
			},
			Candidate: candidate,
		}
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
	if a.MetaInfo.Key == "xunlei" {
		return a.uploadXunlei(req)
	}
	if a.MetaInfo.Key == "pikpak" {
		return a.uploadPikPak(req)
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

func (a HashFamilyAdapter) validateXunleiAuth(profile AuthProfile) OperationResult {
	session, err := a.newXunleiSession(profile)
	if err != nil {
		return OperationResult{
			Status:  "invalid_provider_endpoint",
			Message: err.Error(),
			Mode:    "hash_family_real_auth",
		}
	}

	statusCode, payload, requestErr := getXunleiJSON(context.Background(), session, "/drive/v1/files", map[string]string{
		"limit":          "1",
		"usage":          "DISPLAY",
		"with_audit":     "true",
		"thumbnail_size": "SIZE_SMALL",
		"filters":        `{"phase":{"eq":"PHASE_TYPE_COMPLETE"},"trashed":{"eq":false}}`,
	})
	if requestErr != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Xunlei auth validation request failed: %v", requestErr),
			Mode:    "hash_family_real_auth",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "Xunlei rejected the supplied access token.",
			Mode:    "hash_family_real_auth",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Xunlei auth validation returned HTTP %d.", statusCode),
			Mode:    "hash_family_real_auth",
			Payload: payload,
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Xunlei validated the supplied access token against the live list endpoint.",
		Mode:    "hash_family_real_auth",
		Payload: payload,
	}
}

func (a HashFamilyAdapter) listXunlei(req ListRequest) ListResult {
	session, err := a.newXunleiSession(req.Profile)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "hash_family_real_directory",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	parentID := strings.TrimSpace(req.ParentID)
	basePath := targetPath
	if parentID == "" {
		if targetPath == "/" {
			basePath = "/"
		} else {
			resolvedParentID, resolvedEntry, found, resolveErr := a.resolveXunleiEntryByPath(session, targetPath, req.PageSize)
			if resolveErr != nil {
				return ListResult{
					OperationResult: OperationResult{
						Status:  normalizeHashFamilyRequestErrorStatus(resolveErr),
						Message: fmt.Sprintf("Xunlei path resolution failed: %v", resolveErr),
						Mode:    "hash_family_real_directory",
					},
				}
			}
			if !found {
				return ListResult{
					OperationResult: OperationResult{
						Status:  "path_not_found",
						Message: fmt.Sprintf("Xunlei path %q was not found.", targetPath),
						Mode:    "hash_family_real_directory",
					},
				}
			}
			if !boolMapValue(resolvedEntry, "isDir") {
				return ListResult{
					OperationResult: OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "Xunlei resolved a file path directly.",
						Mode:    "hash_family_real_directory",
					},
					Items: []map[string]interface{}{resolvedEntry},
				}
			}
			parentID = resolvedParentID
			basePath = targetPath
		}
	}

	items, listErr := a.listXunleiByParent(session, parentID, basePath, req.PageSize)
	if listErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeHashFamilyRequestErrorStatus(listErr),
				Message: fmt.Sprintf("Xunlei list request failed: %v", listErr),
				Mode:    "hash_family_real_directory",
			},
		}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Xunlei listed live directory entries.",
			Mode:    "hash_family_real_directory",
		},
		Items: items,
	}
}

func (a HashFamilyAdapter) metadataXunlei(req MetadataRequest) MetadataResult {
	session, err := a.newXunleiSession(req.Profile)
	if err != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "hash_family_real_directory",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	if targetPath == "/" && strings.TrimSpace(req.FileID) == "" {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: "Xunlei root directory metadata is available.",
				Mode:    "hash_family_real_directory",
			},
			Entry: map[string]interface{}{
				"exists":   true,
				"isDir":    true,
				"name":     "/",
				"path":     "/",
				"fileId":   "",
				"provider": a.MetaInfo.Key,
			},
		}
	}

	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" && targetPath != "/" {
		parentPath := normalizeOpenFamilyPath(parentDirectory(targetPath))
		if parentPath == "." || parentPath == "" {
			parentPath = "/"
		}
		if parentPath != "/" {
			resolvedParentID, _, found, resolveErr := a.resolveXunleiEntryByPath(session, parentPath, 0)
			if resolveErr != nil {
				return MetadataResult{
					OperationResult: OperationResult{
						Status:  normalizeHashFamilyRequestErrorStatus(resolveErr),
						Message: fmt.Sprintf("Xunlei parent path resolution failed: %v", resolveErr),
						Mode:    "hash_family_real_directory",
					},
				}
			}
			if !found {
				return MetadataResult{
					OperationResult: OperationResult{
						OK:      true,
						Status:  "missing",
						Message: "Xunlei did not find the requested parent path.",
						Mode:    "hash_family_real_directory",
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

	items, listErr := a.listXunleiByParent(session, parentID, normalizeOpenFamilyPath(parentDirectory(targetPath)), 0)
	if listErr != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeHashFamilyRequestErrorStatus(listErr),
				Message: fmt.Sprintf("Xunlei metadata list request failed: %v", listErr),
				Mode:    "hash_family_real_directory",
			},
		}
	}
	targetName := inferName(targetPath, "")
	targetFileID := strings.TrimSpace(req.FileID)
	for _, item := range items {
		if targetFileID != "" && strings.TrimSpace(stringMapValue(item, "fileId")) == targetFileID {
			return MetadataResult{
				OperationResult: OperationResult{
					OK:      true,
					Status:  "exists",
					Message: "Xunlei returned live metadata.",
					Mode:    "hash_family_real_directory",
				},
				Entry: item,
			}
		}
		if targetFileID == "" && strings.TrimSpace(stringMapValue(item, "name")) == targetName {
			return MetadataResult{
				OperationResult: OperationResult{
					OK:      true,
					Status:  "exists",
					Message: "Xunlei returned live metadata.",
					Mode:    "hash_family_real_directory",
				},
				Entry: item,
			}
		}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "missing",
			Message: "Xunlei did not find the requested path.",
			Mode:    "hash_family_real_directory",
		},
		Entry: map[string]interface{}{
			"exists":   false,
			"path":     targetPath,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a HashFamilyAdapter) createDirXunlei(req CreateDirRequest) OperationResult {
	session, err := a.newXunleiSession(req.Profile)
	if err != nil {
		return OperationResult{
			Status:  "invalid_provider_endpoint",
			Message: err.Error(),
			Mode:    "hash_family_real_directory",
		}
	}

	statusCode, payload, requestErr := postXunleiJSON(context.Background(), session, "/drive/v1/files", map[string]interface{}{
		"kind":      "drive#folder",
		"name":      strings.TrimSpace(req.DirName),
		"parent_id": strings.TrimSpace(req.ParentID),
	})
	if requestErr != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Xunlei create-dir request failed: %v", requestErr),
			Mode:    "hash_family_real_directory",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "Xunlei rejected the supplied access token while creating a directory.",
			Mode:    "hash_family_real_directory",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Xunlei create-dir returned HTTP %d.", statusCode),
			Mode:    "hash_family_real_directory",
			Payload: payload,
		}
	}

	entry := a.normalizeXunleiEntry(payload, pathJoin("/", strings.TrimSpace(req.DirName)), strings.TrimSpace(req.ParentID))
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Xunlei created the requested directory.",
		Mode:    "hash_family_real_directory",
		Payload: entry,
	}
}

func (a HashFamilyAdapter) uploadXunlei(req UploadRequest) UploadResult {
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "Xunlei pending_manual items still require follow-up runtime support.",
				Mode:    "hash_family_real_upload",
			},
		}
	}

	session, err := a.newXunleiSession(req.Profile)
	if err != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "hash_family_real_upload",
			},
		}
	}

	localPath := strings.TrimSpace(req.LocalPath)
	fileSize := req.Size
	if fileSize <= 0 && localPath != "" {
		info, statErr := os.Stat(localPath)
		if statErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "local_file_missing",
					Message: fmt.Sprintf("Xunlei could not stat local file: %v", statErr),
					Mode:    "hash_family_real_upload",
				},
			}
		}
		fileSize = info.Size()
	}

	gcid := strings.ToUpper(strings.TrimSpace(req.GCID))
	if gcid == "" && localPath != "" {
		computed, computeErr := computeHashFamilyGCID(localPath)
		if computeErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "local_file_missing",
					Message: fmt.Sprintf("Xunlei could not compute local gcid: %v", computeErr),
					Mode:    "hash_family_real_upload",
				},
			}
		}
		gcid = computed
	}
	if gcid == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_gcid",
				Message: "Xunlei upload requires gcid or a readable local file to compute it.",
				Mode:    "hash_family_real_upload",
			},
		}
	}
	if fileSize <= 0 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_size",
				Message: "Xunlei upload requires file size or a readable local file.",
				Mode:    "hash_family_real_upload",
			},
		}
	}

	parentID := strings.TrimSpace(req.ParentID)
	if resumed := a.resumeXunleiUpload(session, req, parentID, localPath, gcid); resumed != nil {
		return *resumed
	}
	resolvedTargetName, conflictAction, conflictNote, conflictErr := a.resolveXunleiUploadName(session, parentID, inferName(req.Path, req.Name), req.ConflictPolicy)
	if conflictErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeHashFamilyRequestErrorStatus(conflictErr),
				Message: fmt.Sprintf("Xunlei upload conflict preflight failed: %v", conflictErr),
				Mode:    "hash_family_real_upload",
			},
		}
	}

	statusCode, payload, requestErr := postXunleiJSON(context.Background(), session, "/drive/v1/files", map[string]interface{}{
		"kind":        "drive#file",
		"name":        resolvedTargetName,
		"size":        fileSize,
		"hash":        gcid,
		"upload_type": "UPLOAD_TYPE_RESUMABLE",
		"parent_id":   parentID,
	})
	if requestErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Xunlei upload create request failed: %v", requestErr),
				Mode:    "hash_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "Xunlei rejected the supplied access token while creating an upload.",
				Mode:    "hash_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Xunlei upload create returned HTTP %d.", statusCode),
				Mode:    "hash_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}

	fileInfo, _ := payload["file"].(map[string]interface{})
	resumable, hasResumable := payload["resumable"].(map[string]interface{})
	fileID := firstNonEmptyString(fileInfo, "id", "file_id", "fileId")
	uploadType := firstNonEmptyString(payload, "upload_type", "uploadType")
	resolvedName := firstNonEmptyString(fileInfo, "name", "file_name", "fileName")
	if resolvedName == "" {
		resolvedName = resolvedTargetName
	}

	commonPayload := map[string]interface{}{
		"createResponse":     payload,
		"fileId":             fileID,
		"resolvedTargetName": resolvedName,
		"conflictAction":     conflictAction,
		"uploadType":         uploadType,
		"gcid":               gcid,
	}

	usedBinaryFallback := false
	if hasResumable {
		commonPayload["resumable"] = resumable
		commonPayload["providerData"] = map[string]interface{}{
			"resumable":          cloneMap(resumable),
			"resolvedTargetName": resolvedName,
			"conflictAction":     conflictAction,
			"fileId":             fileID,
			"gcid":               gcid,
			"uploadId":           hashFamilyResumableUploadID(resumable),
		}
		if localPath == "" {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "hash_miss",
					Message: "Xunlei returned a resumable upload session after hash miss, but no local file path was available for binary fallback.",
					Mode:    "hash_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		uploadPayload, uploadErr := hashFamilyResumableUploader("xunlei", localPath, resumable)
		if uploadErr != nil {
			commonPayload["resumableUpload"] = uploadPayload
			commonPayload = mergeMaps(commonPayload, hashFamilyWholeObjectCheckpoint(resumable, fileID, uploadPayload, false))
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "provider_request_failed",
					Message: fmt.Sprintf("Xunlei resumable binary upload failed: %v", uploadErr),
					Mode:    "hash_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		usedBinaryFallback = true
		commonPayload["resumableUpload"] = uploadPayload
		commonPayload = mergeMaps(commonPayload, hashFamilyWholeObjectCheckpoint(resumable, fileID, uploadPayload, true))
	}

	verifyEntry, verifyMode, verifyOK := a.verifyXunleiUploadedFile(session, parentID, resolvedName, fileID, gcid)
	commonPayload["verifyMode"] = verifyMode
	commonPayload["verifyOk"] = verifyOK
	commonPayload["usedBinaryFallback"] = usedBinaryFallback
	if verifyEntry != nil {
		commonPayload["verifiedEntry"] = verifyEntry
	}

	message := "Xunlei rapid-upload request succeeded and did not require a follow-up resumable upload session."
	if usedBinaryFallback {
		message = "Xunlei resumed to binary upload fallback after hash miss and completed successfully."
	}
	if conflictNote != "" {
		message += " " + conflictNote
	}

	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "hash_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: conflictAction,
	}
}

func (a HashFamilyAdapter) validatePikPakAuth(profile AuthProfile) OperationResult {
	session, err := a.newPikPakSession(profile)
	if err != nil {
		return OperationResult{
			Status:  "invalid_provider_endpoint",
			Message: err.Error(),
			Mode:    "hash_family_real_auth",
		}
	}

	statusCode, payload, requestErr := getPikPakJSON(context.Background(), session, "/drive/v1/files", map[string]string{
		"limit":      "1",
		"with_audit": "true",
		"filters":    `{"trashed":{"eq":false}}`,
	})
	if requestErr != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("PikPak auth validation request failed: %v", requestErr),
			Mode:    "hash_family_real_auth",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "PikPak rejected the supplied access token.",
			Mode:    "hash_family_real_auth",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("PikPak auth validation returned HTTP %d.", statusCode),
			Mode:    "hash_family_real_auth",
			Payload: payload,
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "PikPak validated the supplied access token against the live list endpoint.",
		Mode:    "hash_family_real_auth",
		Payload: payload,
	}
}

func (a HashFamilyAdapter) listPikPak(req ListRequest) ListResult {
	session, err := a.newPikPakSession(req.Profile)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "hash_family_real_directory",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	parentID := strings.TrimSpace(req.ParentID)
	basePath := targetPath
	if parentID == "" {
		if targetPath == "/" {
			basePath = "/"
		} else {
			resolvedParentID, resolvedEntry, found, resolveErr := a.resolvePikPakEntryByPath(session, targetPath, req.PageSize)
			if resolveErr != nil {
				return ListResult{
					OperationResult: OperationResult{
						Status:  normalizeHashFamilyRequestErrorStatus(resolveErr),
						Message: fmt.Sprintf("PikPak path resolution failed: %v", resolveErr),
						Mode:    "hash_family_real_directory",
					},
				}
			}
			if !found {
				return ListResult{
					OperationResult: OperationResult{
						Status:  "path_not_found",
						Message: fmt.Sprintf("PikPak path %q was not found.", targetPath),
						Mode:    "hash_family_real_directory",
					},
				}
			}
			if !boolMapValue(resolvedEntry, "isDir") {
				return ListResult{
					OperationResult: OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "PikPak resolved a file path directly.",
						Mode:    "hash_family_real_directory",
					},
					Items: []map[string]interface{}{resolvedEntry},
				}
			}
			parentID = resolvedParentID
			basePath = targetPath
		}
	}

	items, listErr := a.listPikPakByParent(session, parentID, basePath, req.PageSize)
	if listErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeHashFamilyRequestErrorStatus(listErr),
				Message: fmt.Sprintf("PikPak list request failed: %v", listErr),
				Mode:    "hash_family_real_directory",
			},
		}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "PikPak listed live directory entries.",
			Mode:    "hash_family_real_directory",
		},
		Items: items,
	}
}

func (a HashFamilyAdapter) metadataPikPak(req MetadataRequest) MetadataResult {
	session, err := a.newPikPakSession(req.Profile)
	if err != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "hash_family_real_directory",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	if targetPath == "/" && strings.TrimSpace(req.FileID) == "" {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: "PikPak root directory metadata is available.",
				Mode:    "hash_family_real_directory",
			},
			Entry: map[string]interface{}{
				"exists":   true,
				"isDir":    true,
				"name":     "/",
				"path":     "/",
				"fileId":   "",
				"provider": a.MetaInfo.Key,
			},
		}
	}

	if fileID := strings.TrimSpace(req.FileID); fileID != "" {
		entry, getErr := a.getPikPakEntryByID(session, fileID, targetPath)
		if getErr != nil {
			return MetadataResult{
				OperationResult: OperationResult{
					Status:  normalizeHashFamilyRequestErrorStatus(getErr),
					Message: fmt.Sprintf("PikPak metadata request failed: %v", getErr),
					Mode:    "hash_family_real_directory",
				},
			}
		}
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: "PikPak returned live metadata.",
				Mode:    "hash_family_real_directory",
			},
			Entry: entry,
		}
	}

	_, entry, found, resolveErr := a.resolvePikPakEntryByPath(session, targetPath, 0)
	if resolveErr != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeHashFamilyRequestErrorStatus(resolveErr),
				Message: fmt.Sprintf("PikPak path resolution failed: %v", resolveErr),
				Mode:    "hash_family_real_directory",
			},
		}
	}
	if !found {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "missing",
				Message: "PikPak did not find the requested path.",
				Mode:    "hash_family_real_directory",
			},
			Entry: map[string]interface{}{
				"exists":   false,
				"path":     targetPath,
				"provider": a.MetaInfo.Key,
			},
		}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "exists",
			Message: "PikPak returned live metadata.",
			Mode:    "hash_family_real_directory",
		},
		Entry: entry,
	}
}

func (a HashFamilyAdapter) createDirPikPak(req CreateDirRequest) OperationResult {
	session, err := a.newPikPakSession(req.Profile)
	if err != nil {
		return OperationResult{
			Status:  "invalid_provider_endpoint",
			Message: err.Error(),
			Mode:    "hash_family_real_directory",
		}
	}

	statusCode, payload, requestErr := postPikPakJSON(context.Background(), session, "/drive/v1/files", map[string]interface{}{
		"kind":      "drive#folder",
		"name":      strings.TrimSpace(req.DirName),
		"parent_id": strings.TrimSpace(req.ParentID),
	})
	if requestErr != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("PikPak create-dir request failed: %v", requestErr),
			Mode:    "hash_family_real_directory",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "PikPak rejected the supplied access token while creating a directory.",
			Mode:    "hash_family_real_directory",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("PikPak create-dir returned HTTP %d.", statusCode),
			Mode:    "hash_family_real_directory",
			Payload: payload,
		}
	}

	entry := a.normalizePikPakEntry(payload, pathJoin("/", strings.TrimSpace(req.DirName)), strings.TrimSpace(req.ParentID))
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "PikPak created the requested directory.",
		Mode:    "hash_family_real_directory",
		Payload: entry,
	}
}

func (a HashFamilyAdapter) uploadPikPak(req UploadRequest) UploadResult {
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "PikPak pending_manual items still require follow-up runtime support.",
				Mode:    "hash_family_real_upload",
			},
		}
	}

	session, err := a.newPikPakSession(req.Profile)
	if err != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "hash_family_real_upload",
			},
		}
	}

	localPath := strings.TrimSpace(req.LocalPath)
	fileSize := req.Size
	if fileSize <= 0 && localPath != "" {
		info, statErr := os.Stat(localPath)
		if statErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "local_file_missing",
					Message: fmt.Sprintf("PikPak could not stat local file: %v", statErr),
					Mode:    "hash_family_real_upload",
				},
			}
		}
		fileSize = info.Size()
	}

	gcid := strings.ToUpper(strings.TrimSpace(req.GCID))
	if gcid == "" && localPath != "" {
		computed, computeErr := computeHashFamilyGCID(localPath)
		if computeErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "local_file_missing",
					Message: fmt.Sprintf("PikPak could not compute local gcid: %v", computeErr),
					Mode:    "hash_family_real_upload",
				},
			}
		}
		gcid = computed
	}
	if gcid == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_gcid",
				Message: "PikPak upload requires gcid or a readable local file to compute it.",
				Mode:    "hash_family_real_upload",
			},
		}
	}
	if fileSize <= 0 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_size",
				Message: "PikPak upload requires file size or a readable local file.",
				Mode:    "hash_family_real_upload",
			},
		}
	}

	parentID := strings.TrimSpace(req.ParentID)
	if resumed := a.resumePikPakUpload(session, req, parentID, localPath, gcid); resumed != nil {
		return *resumed
	}
	resolvedTargetName, conflictAction, conflictNote, conflictErr := a.resolvePikPakUploadName(session, parentID, inferName(req.Path, req.Name), req.ConflictPolicy)
	if conflictErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeHashFamilyRequestErrorStatus(conflictErr),
				Message: fmt.Sprintf("PikPak upload conflict preflight failed: %v", conflictErr),
				Mode:    "hash_family_real_upload",
			},
		}
	}

	statusCode, payload, requestErr := postPikPakJSON(context.Background(), session, "/drive/v1/files", map[string]interface{}{
		"kind":        "drive#file",
		"name":        resolvedTargetName,
		"size":        fileSize,
		"hash":        gcid,
		"upload_type": "UPLOAD_TYPE_RESUMABLE",
		"objProvider": map[string]interface{}{"provider": "UPLOAD_TYPE_UNKNOWN"},
		"parent_id":   parentID,
		"folder_type": "NORMAL",
	})
	if requestErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("PikPak upload create request failed: %v", requestErr),
				Mode:    "hash_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "PikPak rejected the supplied access token while creating an upload.",
				Mode:    "hash_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("PikPak upload create returned HTTP %d.", statusCode),
				Mode:    "hash_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}

	fileInfo, _ := payload["file"].(map[string]interface{})
	resumable, hasResumable := payload["resumable"].(map[string]interface{})
	fileID := firstNonEmptyString(fileInfo, "id", "file_id", "fileId")
	uploadType := firstNonEmptyString(payload, "upload_type", "uploadType")
	resolvedName := firstNonEmptyString(fileInfo, "name", "file_name", "fileName")
	if resolvedName == "" {
		resolvedName = resolvedTargetName
	}

	commonPayload := map[string]interface{}{
		"createResponse":     payload,
		"fileId":             fileID,
		"resolvedTargetName": resolvedName,
		"conflictAction":     conflictAction,
		"uploadType":         uploadType,
		"gcid":               gcid,
	}

	usedBinaryFallback := false
	if hasResumable {
		commonPayload["resumable"] = resumable
		commonPayload["providerData"] = map[string]interface{}{
			"resumable":          cloneMap(resumable),
			"resolvedTargetName": resolvedName,
			"conflictAction":     conflictAction,
			"fileId":             fileID,
			"gcid":               gcid,
			"uploadId":           hashFamilyResumableUploadID(resumable),
		}
		if localPath == "" {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "hash_miss",
					Message: "PikPak returned a resumable upload session after hash miss, but no local file path was available for binary fallback.",
					Mode:    "hash_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		uploadPayload, uploadErr := hashFamilyResumableUploader("pikpak", localPath, resumable)
		if uploadErr != nil {
			commonPayload["resumableUpload"] = uploadPayload
			commonPayload = mergeMaps(commonPayload, hashFamilyWholeObjectCheckpoint(resumable, fileID, uploadPayload, false))
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "provider_request_failed",
					Message: fmt.Sprintf("PikPak resumable binary upload failed: %v", uploadErr),
					Mode:    "hash_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		usedBinaryFallback = true
		commonPayload["resumableUpload"] = uploadPayload
		commonPayload = mergeMaps(commonPayload, hashFamilyWholeObjectCheckpoint(resumable, fileID, uploadPayload, true))
	}

	verifyEntry, verifyMode, verifyOK := a.verifyPikPakUploadedFile(session, parentID, resolvedName, fileID, gcid)
	commonPayload["verifyMode"] = verifyMode
	commonPayload["verifyOk"] = verifyOK
	commonPayload["usedBinaryFallback"] = usedBinaryFallback
	if verifyEntry != nil {
		commonPayload["verifiedEntry"] = verifyEntry
	}

	message := "PikPak rapid-upload request succeeded and did not require a follow-up resumable upload session."
	if usedBinaryFallback {
		message = "PikPak resumed to binary upload fallback after hash miss and completed successfully."
	}
	if conflictNote != "" {
		message += " " + conflictNote
	}

	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "hash_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: conflictAction,
	}
}

func (a HashFamilyAdapter) resumeXunleiUpload(session hashFamilySession, req UploadRequest, parentID string, localPath string, gcid string) *UploadResult {
	resume := req.ResumeUpload
	if resume == nil || strings.TrimSpace(localPath) == "" {
		return nil
	}
	resumable := hashFamilyResumeSessionPayload(*resume)
	if len(resumable) == 0 {
		return nil
	}
	fileID := strings.TrimSpace(resume.FileID)
	if fileID == "" {
		return nil
	}
	resolvedName := hashFamilyResumeTargetName(*resume, inferName(req.Path, req.Name))
	uploadPayload, uploadErr := hashFamilyResumableUploader("xunlei", localPath, resumable)
	commonPayload := map[string]interface{}{
		"fileId":             fileID,
		"uploadId":           firstNonEmptyValue(resume.UploadID, hashFamilyResumableUploadID(resumable)),
		"resolvedTargetName": resolvedName,
		"gcid":               gcid,
		"resumedUpload":      true,
		"resumable":          resumable,
		"providerData":       cloneMap(resume.ProviderData),
		"resumableUpload":    uploadPayload,
	}
	if uploadErr != nil {
		commonPayload = mergeMaps(commonPayload, hashFamilyWholeObjectCheckpoint(resumable, fileID, uploadPayload, false))
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Xunlei resumed resumable binary upload failed: %v", uploadErr),
				Mode:    "hash_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: stringMapValue(resume.ProviderData, "conflictAction"),
		}
	}
	commonPayload = mergeMaps(commonPayload, hashFamilyWholeObjectCheckpoint(resumable, fileID, uploadPayload, true))
	verifyEntry, verifyMode, verifyOK := a.verifyXunleiUploadedFile(session, parentID, resolvedName, fileID, gcid)
	commonPayload["verifyMode"] = verifyMode
	commonPayload["verifyOk"] = verifyOK
	commonPayload["usedBinaryFallback"] = true
	if verifyEntry != nil {
		commonPayload["verifiedEntry"] = verifyEntry
	}
	return &UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Xunlei resumed a previously created resumable upload session and verified the uploaded file afterwards.",
			Mode:    "hash_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: stringMapValue(resume.ProviderData, "conflictAction"),
	}
}

func (a HashFamilyAdapter) resumePikPakUpload(session hashFamilySession, req UploadRequest, parentID string, localPath string, gcid string) *UploadResult {
	resume := req.ResumeUpload
	if resume == nil || strings.TrimSpace(localPath) == "" {
		return nil
	}
	resumable := hashFamilyResumeSessionPayload(*resume)
	if len(resumable) == 0 {
		return nil
	}
	fileID := strings.TrimSpace(resume.FileID)
	if fileID == "" {
		return nil
	}
	resolvedName := hashFamilyResumeTargetName(*resume, inferName(req.Path, req.Name))
	uploadPayload, uploadErr := hashFamilyResumableUploader("pikpak", localPath, resumable)
	commonPayload := map[string]interface{}{
		"fileId":             fileID,
		"uploadId":           firstNonEmptyValue(resume.UploadID, hashFamilyResumableUploadID(resumable)),
		"resolvedTargetName": resolvedName,
		"gcid":               gcid,
		"resumedUpload":      true,
		"resumable":          resumable,
		"providerData":       cloneMap(resume.ProviderData),
		"resumableUpload":    uploadPayload,
	}
	if uploadErr != nil {
		commonPayload = mergeMaps(commonPayload, hashFamilyWholeObjectCheckpoint(resumable, fileID, uploadPayload, false))
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("PikPak resumed resumable binary upload failed: %v", uploadErr),
				Mode:    "hash_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: stringMapValue(resume.ProviderData, "conflictAction"),
		}
	}
	commonPayload = mergeMaps(commonPayload, hashFamilyWholeObjectCheckpoint(resumable, fileID, uploadPayload, true))
	verifyEntry, verifyMode, verifyOK := a.verifyPikPakUploadedFile(session, parentID, resolvedName, fileID, gcid)
	commonPayload["verifyMode"] = verifyMode
	commonPayload["verifyOk"] = verifyOK
	commonPayload["usedBinaryFallback"] = true
	if verifyEntry != nil {
		commonPayload["verifiedEntry"] = verifyEntry
	}
	return &UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "PikPak resumed a previously created resumable upload session and verified the uploaded file afterwards.",
			Mode:    "hash_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: stringMapValue(resume.ProviderData, "conflictAction"),
	}
}

func (a HashFamilyAdapter) newXunleiSession(profile AuthProfile) (hashFamilySession, error) {
	baseEndpoint, err := resolveXunleiEndpoint(profile)
	if err != nil {
		return hashFamilySession{}, err
	}
	authorization := hashFamilyAuthorizationHeader(profile)
	if authorization == "" {
		return hashFamilySession{}, fmt.Errorf("missing access token")
	}
	return hashFamilySession{
		BaseEndpoint:  baseEndpoint,
		Authorization: authorization,
		DeviceID:      firstNonEmptyExtra(profile.Extra, "deviceId", "device_id", "x-device-id"),
		CaptchaToken:  firstNonEmptyExtra(profile.Extra, "captchaToken", "captcha_token", "x-captcha-token"),
		ClientID:      firstNonEmptyExtra(profile.Extra, "clientId", "client_id", "x-client-id"),
		ProviderKey:   a.MetaInfo.Key,
	}, nil
}

func (a HashFamilyAdapter) newPikPakSession(profile AuthProfile) (hashFamilySession, error) {
	baseEndpoint, err := resolvePikPakEndpoint(profile)
	if err != nil {
		return hashFamilySession{}, err
	}
	authorization := hashFamilyAuthorizationHeader(profile)
	if authorization == "" {
		return hashFamilySession{}, fmt.Errorf("missing access token")
	}
	return hashFamilySession{
		BaseEndpoint:  baseEndpoint,
		Authorization: authorization,
		DeviceID:      firstNonEmptyExtra(profile.Extra, "deviceId", "device_id", "x-device-id"),
		CaptchaToken:  firstNonEmptyExtra(profile.Extra, "captchaToken", "captcha_token", "x-captcha-token"),
		ProviderKey:   a.MetaInfo.Key,
	}, nil
}

func (a HashFamilyAdapter) resolveXunleiEntryByPath(session hashFamilySession, path string, pageSize int) (string, map[string]interface{}, bool, error) {
	normalized := normalizeOpenFamilyPath(path)
	if normalized == "/" {
		return "", map[string]interface{}{
			"exists":   true,
			"isDir":    true,
			"name":     "/",
			"path":     "/",
			"fileId":   "",
			"provider": session.ProviderKey,
		}, true, nil
	}

	currentID := ""
	currentPath := "/"
	var currentEntry map[string]interface{}
	parts := strings.Split(strings.Trim(normalized, "/"), "/")
	for _, part := range parts {
		children, err := a.listXunleiByParent(session, currentID, currentPath, pageSize)
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

func (a HashFamilyAdapter) listXunleiByParent(session hashFamilySession, parentID string, basePath string, pageSize int) ([]map[string]interface{}, error) {
	limit := pageSize
	if limit <= 0 {
		limit = 100
	}
	pageToken := ""
	items := make([]map[string]interface{}, 0)
	for {
		params := map[string]string{
			"parent_id":      parentID,
			"usage":          "DISPLAY",
			"with_audit":     "true",
			"thumbnail_size": "SIZE_SMALL",
			"limit":          strconv.Itoa(limit),
			"filters":        `{"phase":{"eq":"PHASE_TYPE_COMPLETE"},"trashed":{"eq":false}}`,
		}
		if pageToken != "" {
			params["page_token"] = pageToken
		}
		statusCode, payload, err := getXunleiJSON(context.Background(), session, "/drive/v1/files", params)
		if err != nil {
			return nil, err
		}
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			return nil, fmt.Errorf("auth_invalid")
		}
		if statusCode < 200 || statusCode >= 300 {
			return nil, fmt.Errorf("http %d", statusCode)
		}
		for _, raw := range interfaceSliceValue(payload, "files") {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			items = append(items, a.normalizeXunleiEntry(item, pathJoin(basePath, firstNonEmptyString(item, "name", "file_name")), parentID))
		}
		nextPageToken := firstNonEmptyString(payload, "next_page_token", "nextPageToken")
		if nextPageToken == "" {
			return items, nil
		}
		pageToken = nextPageToken
	}
}

func (a HashFamilyAdapter) resolvePikPakEntryByPath(session hashFamilySession, path string, pageSize int) (string, map[string]interface{}, bool, error) {
	normalized := normalizeOpenFamilyPath(path)
	if normalized == "/" {
		return "", map[string]interface{}{
			"exists":   true,
			"isDir":    true,
			"name":     "/",
			"path":     "/",
			"fileId":   "",
			"provider": session.ProviderKey,
		}, true, nil
	}

	currentID := ""
	currentPath := "/"
	var currentEntry map[string]interface{}
	parts := strings.Split(strings.Trim(normalized, "/"), "/")
	for _, part := range parts {
		children, err := a.listPikPakByParent(session, currentID, currentPath, pageSize)
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

func (a HashFamilyAdapter) listPikPakByParent(session hashFamilySession, parentID string, basePath string, pageSize int) ([]map[string]interface{}, error) {
	limit := pageSize
	if limit <= 0 {
		limit = 100
	}
	pageToken := ""
	items := make([]map[string]interface{}, 0)
	for {
		params := map[string]string{
			"parent_id":  parentID,
			"with_audit": "true",
			"limit":      strconv.Itoa(limit),
			"filters":    `{"trashed":{"eq":false}}`,
		}
		if pageToken != "" {
			params["page_token"] = pageToken
		}
		statusCode, payload, err := getPikPakJSON(context.Background(), session, "/drive/v1/files", params)
		if err != nil {
			return nil, err
		}
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			return nil, fmt.Errorf("auth_invalid")
		}
		if statusCode < 200 || statusCode >= 300 {
			return nil, fmt.Errorf("http %d", statusCode)
		}
		for _, raw := range interfaceSliceValue(payload, "files") {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			items = append(items, a.normalizePikPakEntry(item, pathJoin(basePath, firstNonEmptyString(item, "name", "file_name")), parentID))
		}
		nextPageToken := firstNonEmptyString(payload, "next_page_token", "nextPageToken")
		if nextPageToken == "" {
			return items, nil
		}
		pageToken = nextPageToken
	}
}

func (a HashFamilyAdapter) normalizeXunleiEntry(raw map[string]interface{}, path string, parentID string) map[string]interface{} {
	kind := strings.ToLower(firstNonEmptyString(raw, "kind"))
	isDir := kind == "drive#folder" || kind == "drive#dir" || kind == "drive#directory" || boolMapValue(raw, "isDir")
	gcid := strings.ToUpper(firstNonEmptyString(raw, "hash", "gcid"))
	entry := map[string]interface{}{
		"exists":   true,
		"name":     firstNonEmptyString(raw, "name", "file_name"),
		"path":     normalizeOpenFamilyPath(path),
		"fileId":   firstNonEmptyString(raw, "id", "file_id", "fileId"),
		"parentId": firstNonEmptyString(raw, "parent_id", "parentId"),
		"isDir":    isDir,
		"size":     int64MapValue(raw, "size"),
		"provider": a.MetaInfo.Key,
	}
	if stringMapValue(entry, "parentId") == "" {
		entry["parentId"] = parentID
	}
	if gcid != "" {
		entry["gcid"] = gcid
	}
	return entry
}

func (a HashFamilyAdapter) normalizePikPakEntry(raw map[string]interface{}, path string, parentID string) map[string]interface{} {
	kind := strings.ToLower(firstNonEmptyString(raw, "kind"))
	isDir := kind == "drive#folder" || kind == "drive#dir" || kind == "drive#directory" || boolMapValue(raw, "isDir")
	hashValue := strings.ToUpper(firstNonEmptyString(raw, "hash", "gcid"))
	entry := map[string]interface{}{
		"exists":   true,
		"name":     firstNonEmptyString(raw, "name", "file_name"),
		"path":     normalizeOpenFamilyPath(path),
		"fileId":   firstNonEmptyString(raw, "id", "file_id", "fileId"),
		"parentId": firstNonEmptyString(raw, "parent_id", "parentId"),
		"isDir":    isDir,
		"size":     int64MapValue(raw, "size"),
		"provider": a.MetaInfo.Key,
	}
	if stringMapValue(entry, "parentId") == "" {
		entry["parentId"] = parentID
	}
	if len(hashValue) == 32 {
		entry["md5"] = hashValue
		entry["etag"] = hashValue
	}
	if len(hashValue) == 40 {
		entry["gcid"] = hashValue
	}
	return entry
}

func (a HashFamilyAdapter) resolveXunleiUploadName(session hashFamilySession, parentID string, targetName string, policy ConflictPolicy) (string, string, string, error) {
	items, err := a.listXunleiByParent(session, parentID, "/", 200)
	if err != nil {
		return "", "", "", err
	}
	existing := make(map[string]bool, len(items))
	for _, item := range items {
		name := strings.TrimSpace(stringMapValue(item, "name"))
		if name != "" {
			existing[name] = true
		}
	}
	if !existing[targetName] {
		return targetName, "none", "", nil
	}
	index := 1
	stem, suffix := splitHashFamilyName(targetName)
	candidate := targetName
	for existing[candidate] {
		candidate = fmt.Sprintf("%s (%d)%s", stem, index, suffix)
		index++
	}
	if policy == ConflictPolicyAutoRenameNew {
		return candidate, "auto_rename_new", "A same-name file already exists under the target path, so Xunlei auto-renamed the new file.", nil
	}
	return candidate, "overwrite_downgraded_to_auto_rename", "The requested overwrite policy was downgraded because the current Xunlei upload path does not support verified in-place overwrite.", nil
}

func (a HashFamilyAdapter) resolvePikPakUploadName(session hashFamilySession, parentID string, targetName string, policy ConflictPolicy) (string, string, string, error) {
	items, err := a.listPikPakByParent(session, parentID, "/", 200)
	if err != nil {
		return "", "", "", err
	}
	existing := make(map[string]bool, len(items))
	for _, item := range items {
		name := strings.TrimSpace(stringMapValue(item, "name"))
		if name != "" {
			existing[name] = true
		}
	}
	if !existing[targetName] {
		return targetName, "none", "", nil
	}
	index := 1
	stem, suffix := splitHashFamilyName(targetName)
	candidate := targetName
	for existing[candidate] {
		candidate = fmt.Sprintf("%s (%d)%s", stem, index, suffix)
		index++
	}
	if policy == ConflictPolicyAutoRenameNew {
		return candidate, "auto_rename_new", "A same-name file already exists under the target path, so PikPak auto-renamed the new file.", nil
	}
	return candidate, "overwrite_downgraded_to_auto_rename", "The requested overwrite policy was downgraded because the current PikPak upload path does not support verified in-place overwrite.", nil
}

func splitHashFamilyName(name string) (string, string) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return "file", ""
	}
	idx := strings.LastIndex(trimmed, ".")
	if idx <= 0 || idx == len(trimmed)-1 {
		return trimmed, ""
	}
	return trimmed[:idx], trimmed[idx:]
}

func (a HashFamilyAdapter) verifyXunleiUploadedFile(session hashFamilySession, parentID string, targetName string, fileID string, expectedGCID string) (map[string]interface{}, string, bool) {
	items, err := a.listXunleiByParent(session, parentID, "/", 200)
	if err != nil {
		return nil, "verify_unavailable", false
	}
	normalizedGCID := strings.ToUpper(strings.TrimSpace(expectedGCID))
	if strings.TrimSpace(fileID) != "" {
		for _, item := range items {
			if strings.TrimSpace(stringMapValue(item, "fileId")) != strings.TrimSpace(fileID) {
				continue
			}
			itemGCID := strings.ToUpper(strings.TrimSpace(stringMapValue(item, "gcid")))
			return item, "metadata_by_file_id", normalizedGCID == "" || itemGCID == "" || itemGCID == normalizedGCID
		}
	}
	for _, item := range items {
		if strings.TrimSpace(stringMapValue(item, "name")) != strings.TrimSpace(targetName) {
			continue
		}
		itemGCID := strings.ToUpper(strings.TrimSpace(stringMapValue(item, "gcid")))
		return item, "list_by_parent_name", normalizedGCID == "" || itemGCID == "" || itemGCID == normalizedGCID
	}
	return nil, "list_by_parent_name", false
}

func (a HashFamilyAdapter) verifyPikPakUploadedFile(session hashFamilySession, parentID string, targetName string, fileID string, expectedGCID string) (map[string]interface{}, string, bool) {
	if strings.TrimSpace(fileID) != "" {
		entry, err := a.getPikPakEntryByID(session, fileID, "")
		if err == nil {
			itemGCID := strings.ToUpper(strings.TrimSpace(stringMapValue(entry, "gcid")))
			normalizedGCID := strings.ToUpper(strings.TrimSpace(expectedGCID))
			return entry, "metadata_by_file_id", normalizedGCID == "" || itemGCID == "" || itemGCID == normalizedGCID
		}
	}
	items, err := a.listPikPakByParent(session, parentID, "/", 200)
	if err != nil {
		return nil, "verify_unavailable", false
	}
	normalizedGCID := strings.ToUpper(strings.TrimSpace(expectedGCID))
	for _, item := range items {
		if strings.TrimSpace(stringMapValue(item, "name")) != strings.TrimSpace(targetName) {
			continue
		}
		itemGCID := strings.ToUpper(strings.TrimSpace(stringMapValue(item, "gcid")))
		return item, "list_by_parent_name", normalizedGCID == "" || itemGCID == "" || itemGCID == normalizedGCID
	}
	return nil, "list_by_parent_name", false
}

func (a HashFamilyAdapter) getPikPakEntryByID(session hashFamilySession, fileID string, path string) (map[string]interface{}, error) {
	statusCode, payload, err := getPikPakJSON(context.Background(), session, "/drive/v1/files/"+strings.TrimSpace(fileID), map[string]string{
		"thumbnail_size": "SIZE_MEDIUM",
	})
	if err != nil {
		return nil, err
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return nil, fmt.Errorf("auth_invalid")
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, fmt.Errorf("http %d", statusCode)
	}
	resolvedPath := path
	if strings.TrimSpace(resolvedPath) == "" {
		resolvedPath = normalizeOpenFamilyPath(pathJoin("/", firstNonEmptyString(payload, "name", "file_name")))
	}
	return a.normalizePikPakEntry(payload, resolvedPath, firstNonEmptyString(payload, "parent_id", "parentId")), nil
}

func hashFamilyAuthorizationHeader(profile AuthProfile) string {
	token := strings.TrimSpace(profile.Token)
	if token == "" {
		token = strings.TrimSpace(profile.Extra["authorization"])
	}
	if token == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return token
	}
	return "Bearer " + token
}

func resolveXunleiEndpoint(profile AuthProfile) (string, error) {
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
	return xunleiAPIBase, nil
}

func resolvePikPakEndpoint(profile AuthProfile) (string, error) {
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
	return pikpakAPIBase, nil
}

func firstNonEmptyExtra(values map[string]string, keys ...string) string {
	for _, key := range keys {
		if strings.TrimSpace(values[key]) != "" {
			return strings.TrimSpace(values[key])
		}
	}
	return ""
}

func getXunleiJSON(ctx context.Context, session hashFamilySession, requestPath string, params map[string]string) (int, map[string]interface{}, error) {
	endpoint := strings.TrimRight(session.BaseEndpoint, "/") + requestPath
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return 0, nil, err
	}
	query := parsed.Query()
	for key, value := range params {
		if strings.TrimSpace(key) == "" {
			continue
		}
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return 0, nil, err
	}
	applyXunleiHeaders(req, session)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	payload, err := decodeProviderJSONResponse(resp.StatusCode, bodyBytes)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, payload, nil
}

func getPikPakJSON(ctx context.Context, session hashFamilySession, requestPath string, params map[string]string) (int, map[string]interface{}, error) {
	endpoint := strings.TrimRight(session.BaseEndpoint, "/") + requestPath
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return 0, nil, err
	}
	query := parsed.Query()
	for key, value := range params {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	if err != nil {
		return 0, nil, err
	}
	applyPikPakHeaders(req, session, false)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	payload, err := decodeProviderJSONResponse(resp.StatusCode, bodyBytes)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, payload, nil
}

func postXunleiJSON(ctx context.Context, session hashFamilySession, requestPath string, body interface{}) (int, map[string]interface{}, error) {
	payload := []byte("{}")
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		payload = raw
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(session.BaseEndpoint, "/")+requestPath, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	applyXunleiHeaders(req, session)
	req.Header.Set("Content-Type", "application/json")
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

func postPikPakJSON(ctx context.Context, session hashFamilySession, requestPath string, body interface{}) (int, map[string]interface{}, error) {
	payload := []byte("{}")
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		payload = raw
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(session.BaseEndpoint, "/")+requestPath, bytes.NewReader(payload))
	if err != nil {
		return 0, nil, err
	}
	applyPikPakHeaders(req, session, true)
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

func applyXunleiHeaders(req *http.Request, session hashFamilySession) {
	req.Header.Set("User-Agent", "CloudPanSync/0.1")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://pan.xunlei.com/")
	req.Header.Set("Authorization", session.Authorization)
	if strings.TrimSpace(session.ClientID) != "" {
		req.Header.Set("x-client-id", session.ClientID)
	} else {
		req.Header.Set("x-client-id", xunleiClientID)
	}
	if strings.TrimSpace(session.DeviceID) != "" {
		req.Header.Set("x-device-id", session.DeviceID)
	}
	if strings.TrimSpace(session.CaptchaToken) != "" {
		req.Header.Set("x-captcha-token", session.CaptchaToken)
	}
}

func applyPikPakHeaders(req *http.Request, session hashFamilySession, jsonBody bool) {
	req.Header.Set("User-Agent", "CloudPanSync/0.1")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://mypikpak.com/")
	req.Header.Set("Origin", "https://mypikpak.com")
	req.Header.Set("Authorization", session.Authorization)
	if strings.TrimSpace(session.DeviceID) != "" {
		req.Header.Set("x-device-id", session.DeviceID)
	}
	if strings.TrimSpace(session.CaptchaToken) != "" {
		req.Header.Set("x-captcha-token", session.CaptchaToken)
	}
	if jsonBody {
		req.Header.Set("Content-Type", "application/json;charset=utf-8")
	}
}

func computeHashFamilyGCID(localPath string) (string, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}
	blockSize := int64(0x40000)
	for info.Size()/blockSize > 0x200 && blockSize < 0x200000 {
		blockSize <<= 1
	}
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	outer := sha1.New()
	buffer := make([]byte, blockSize)
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			inner := sha1.Sum(buffer[:n])
			_, _ = outer.Write(inner[:])
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	return strings.ToUpper(fmt.Sprintf("%x", outer.Sum(nil))), nil
}

func uploadHashFamilyResumableBinary(providerKey string, localPath string, resumable map[string]interface{}) (map[string]interface{}, error) {
	session, payload, err := parseHashFamilyResumableSession(localPath, resumable)
	if err != nil {
		return payload, err
	}

	file, err := os.Open(localPath)
	if err != nil {
		return payload, err
	}
	defer func() { _ = file.Close() }()
	putPayload, err := putHashFamilyResumableObject(context.Background(), providerKey, session, file)
	payload = mergeMaps(payload, putPayload)
	if err != nil {
		hashFamilyApplyWholeObjectProgress(payload, false)
		return payload, err
	}
	hashFamilyApplyWholeObjectProgress(payload, true)
	return payload, nil
}

func hashFamilyResumableUploadID(resumable map[string]interface{}) string {
	if len(resumable) == 0 {
		return ""
	}
	if value := firstNonEmptyString(resumable, "upload_id", "uploadId", "id"); value != "" {
		return value
	}
	params, _ := resumable["params"].(map[string]interface{})
	return firstNonEmptyString(params, "upload_id", "uploadId", "key")
}

func hashFamilyWholeObjectCheckpoint(resumable map[string]interface{}, fileID string, uploadPayload map[string]interface{}, completed bool) map[string]interface{} {
	out := map[string]interface{}{
		"fileId":            fileID,
		"uploadId":          hashFamilyResumableUploadID(resumable),
		"partCount":         1,
		"uploadedPartCount": 0,
		"failedPartNumber":  1,
		"nextPartNumber":    1,
	}
	if completed {
		out["uploadedPartCount"] = 1
		out["uploadedParts"] = []map[string]interface{}{
			{
				"partNumber": 1,
				"etag":       stringMapValue(uploadPayload, "etag"),
				"size":       int64MapValue(uploadPayload, "objectSize"),
			},
		}
		out["failedPartNumber"] = 0
		out["nextPartNumber"] = 2
	}
	return out
}

func hashFamilyApplyWholeObjectProgress(payload map[string]interface{}, completed bool) {
	payload["partCount"] = 1
	if completed {
		payload["uploadedPartCount"] = 1
		payload["uploadedParts"] = []map[string]interface{}{
			{
				"partNumber": 1,
				"etag":       stringMapValue(payload, "etag"),
				"size":       int64MapValue(payload, "objectSize"),
			},
		}
		payload["failedPartNumber"] = 0
		payload["nextPartNumber"] = 2
		return
	}
	payload["uploadedPartCount"] = 0
	payload["failedPartNumber"] = 1
	payload["nextPartNumber"] = 1
}

func hashFamilyResumeSessionPayload(resume ResumeUpload) map[string]interface{} {
	if len(resume.ProviderData) == 0 {
		return nil
	}
	resumable, _ := resume.ProviderData["resumable"].(map[string]interface{})
	if len(resumable) == 0 {
		return nil
	}
	return cloneMap(resumable)
}

func hashFamilyResumeTargetName(resume ResumeUpload, fallback string) string {
	if resolved := stringMapValue(resume.ProviderData, "resolvedTargetName"); resolved != "" {
		return resolved
	}
	return fallback
}

func parseHashFamilyResumableSession(localPath string, resumable map[string]interface{}) (hashFamilyResumableSession, map[string]interface{}, error) {
	payload := map[string]interface{}{
		"resumable": resumable,
		"localPath": localPath,
	}
	params, _ := resumable["params"].(map[string]interface{})
	if len(params) == 0 {
		return hashFamilyResumableSession{}, payload, fmt.Errorf("resumable payload missing params")
	}

	accessKeyID := firstNonEmptyString(params, "access_key_id", "accessKeyId")
	accessKeySecret := firstNonEmptyString(params, "access_key_secret", "accessKeySecret")
	securityToken := firstNonEmptyString(params, "security_token", "securityToken")
	bucket := firstNonEmptyString(params, "bucket")
	endpoint := firstNonEmptyString(params, "endpoint")
	key := firstNonEmptyString(params, "key")
	if accessKeyID == "" || accessKeySecret == "" || securityToken == "" || bucket == "" || endpoint == "" || key == "" {
		return hashFamilyResumableSession{}, payload, fmt.Errorf("resumable payload missing s3-compatible session fields")
	}

	endpointURL, endpointHost, usePathStyle, err := resolveHashFamilyResumableEndpoint(endpoint, bucket)
	if err != nil {
		return hashFamilyResumableSession{}, payload, err
	}
	payload["provider"] = firstNonEmptyString(resumable, "provider")
	payload["bucket"] = bucket
	payload["endpoint"] = endpointHost
	payload["key"] = key
	payload["endpointURL"] = endpointURL
	payload["usePathStyle"] = usePathStyle
	return hashFamilyResumableSession{
		Bucket:            bucket,
		Key:               key,
		EndpointURL:       endpointURL,
		EndpointHost:      endpointHost,
		AccessKeyID:       accessKeyID,
		AccessKeySecret:   accessKeySecret,
		SecurityToken:     securityToken,
		UsePathStyle:      usePathStyle,
		OriginalResumable: resumable,
	}, payload, nil
}

func trimBucketFromEndpoint(endpoint string, bucket string) string {
	resolved := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://"))
	prefix := bucket + "."
	if strings.HasPrefix(resolved, prefix) {
		return strings.TrimPrefix(resolved, prefix)
	}
	return resolved
}

func resolveHashFamilyResumableEndpoint(endpoint string, bucket string) (string, string, bool, error) {
	resolvedEndpoint := strings.TrimSpace(endpoint)
	if resolvedEndpoint == "" {
		return "", "", false, fmt.Errorf("resumable payload missing endpoint")
	}
	if !strings.Contains(resolvedEndpoint, "://") {
		resolvedEndpoint = "https://" + resolvedEndpoint
	}
	parsed, err := url.Parse(resolvedEndpoint)
	if err != nil {
		return "", "", false, fmt.Errorf("invalid resumable endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", "", false, fmt.Errorf("invalid resumable endpoint: scheme and host are required")
	}
	endpointHost := trimBucketFromEndpoint(parsed.Host, bucket)
	baseURL := parsed.Scheme + "://" + endpointHost
	usePathStyle := shouldUseHashFamilyPathStyle(parsed.Hostname(), endpointHost)
	return baseURL, endpointHost, usePathStyle, nil
}

func shouldUseHashFamilyPathStyle(hostname string, endpointHost string) bool {
	host := strings.TrimSpace(hostname)
	if host == "" {
		host = strings.TrimSpace(endpointHost)
	}
	host = strings.TrimPrefix(strings.TrimPrefix(host, "["), "]")
	if strings.EqualFold(host, "localhost") || strings.HasSuffix(strings.ToLower(host), ".localhost") {
		return true
	}
	return net.ParseIP(host) != nil || strings.Contains(endpointHost, ":")
}

func putHashFamilyResumableObject(ctx context.Context, region string, session hashFamilyResumableSession, file *os.File) (map[string]interface{}, error) {
	payload := map[string]interface{}{}
	info, err := file.Stat()
	if err != nil {
		return payload, fmt.Errorf("stat resumable local file: %w", err)
	}
	payload["objectSize"] = info.Size()
	payloadHash, err := hashHashFamilyFileSHA256(file)
	if err != nil {
		return payload, fmt.Errorf("hash resumable local file: %w", err)
	}
	payload["sha256"] = payloadHash
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return payload, fmt.Errorf("rewind resumable local file: %w", err)
	}

	requestURL, canonicalURI, host, err := buildHashFamilyResumableObjectURL(session)
	if err != nil {
		return payload, err
	}
	payload["requestURL"] = requestURL
	amzTime := time.Now().UTC()
	amzDate := amzTime.Format("20060102T150405Z")
	shortDate := amzTime.Format("20060102")
	signedHeaders := "host;x-amz-content-sha256;x-amz-date;x-amz-security-token"
	canonicalHeaders := strings.Join([]string{
		"host:" + host,
		"x-amz-content-sha256:" + payloadHash,
		"x-amz-date:" + amzDate,
		"x-amz-security-token:" + session.SecurityToken,
		"",
	}, "\n")
	canonicalRequest := strings.Join([]string{
		http.MethodPut,
		canonicalURI,
		"",
		canonicalHeaders,
		signedHeaders,
		payloadHash,
	}, "\n")
	scope := shortDate + "/" + region + "/s3/aws4_request"
	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hashHashFamilyStringSHA256(canonicalRequest),
	}, "\n")
	signingKey := deriveHashFamilySigV4Key(session.AccessKeySecret, shortDate, region, "s3")
	signature := hex.EncodeToString(hashFamilyHMACSHA256(signingKey, stringToSign))
	authorization := fmt.Sprintf(
		"AWS4-HMAC-SHA256 Credential=%s/%s, SignedHeaders=%s, Signature=%s",
		session.AccessKeyID,
		scope,
		signedHeaders,
		signature,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, requestURL, file)
	if err != nil {
		return payload, fmt.Errorf("build resumable upload request: %w", err)
	}
	req.Header.Set("Authorization", authorization)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Security-Token", session.SecurityToken)
	req.Header.Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return payload, fmt.Errorf("put resumable object: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload["statusCode"] = resp.StatusCode
	payload["etag"] = strings.Trim(strings.TrimSpace(resp.Header.Get("ETag")), "\"")
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		payload["responseBody"] = strings.TrimSpace(string(bodyBytes))
		return payload, fmt.Errorf("put resumable object returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	return payload, nil
}

func buildHashFamilyResumableObjectURL(session hashFamilyResumableSession) (string, string, string, error) {
	baseURL, err := url.Parse(session.EndpointURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid resumable endpoint URL: %w", err)
	}
	objectPath := encodeHashFamilyS3Key(session.Key)
	if session.UsePathStyle {
		baseURL.Path = "/" + url.PathEscape(session.Bucket) + "/" + objectPath
		return baseURL.String(), baseURL.EscapedPath(), baseURL.Host, nil
	}
	baseURL.Host = session.Bucket + "." + baseURL.Host
	baseURL.Path = "/" + objectPath
	return baseURL.String(), baseURL.EscapedPath(), baseURL.Host, nil
}

func encodeHashFamilyS3Key(key string) string {
	parts := strings.Split(strings.TrimLeft(strings.TrimSpace(key), "/"), "/")
	escaped := make([]string, 0, len(parts))
	for _, part := range parts {
		if part == "" {
			continue
		}
		escaped = append(escaped, url.PathEscape(part))
	}
	return strings.Join(escaped, "/")
}

func hashHashFamilyFileSHA256(file *os.File) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func hashHashFamilyStringSHA256(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func deriveHashFamilySigV4Key(secret string, date string, region string, service string) []byte {
	dateKey := hashFamilyHMACSHA256([]byte("AWS4"+secret), date)
	regionKey := hashFamilyHMACSHA256(dateKey, region)
	serviceKey := hashFamilyHMACSHA256(regionKey, service)
	return hashFamilyHMACSHA256(serviceKey, "aws4_request")
}

func hashFamilyHMACSHA256(key []byte, value string) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func normalizeHashFamilyRequestErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "auth_invalid") {
		return "auth_invalid"
	}
	if strings.Contains(err.Error(), "parent_path_not_found") {
		return "path_not_found"
	}
	return "provider_request_failed"
}
