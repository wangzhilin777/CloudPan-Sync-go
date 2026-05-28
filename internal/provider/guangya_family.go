package provider

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	guangyaAPIHost                       = "https://api.guangyapan.com"
	guangyaListPath                      = "/nd.bizuserres.s/v1/file/get_file_list"
	guangyaDownloadMetaPath              = "/nd.bizuserres.s/v1/get_res_download_url"
	guangyaCreateDirPath                 = "/nd.bizuserres.s/v1/file/create_dir"
	guangyaResCenterTokenPath            = "/nd.bizuserres.s/v1/get_res_center_token"
	guangyaCheckCanFlashUploadPath       = "/nd.bizuserres.s/v1/check_can_flash_upload"
	guangyaDeleteUploadTaskPath          = "/nd.bizuserres.s/v1/file/delete_upload_task"
	guangyaUploadInfoPath                = "/nd.bizuserres.s/v1/file/get_info_by_task_id"
	guangyaCodeResTokenInstant     int64 = 156
	guangyaUploadChunkSize               = 5 * 1024 * 1024
)

type GuangyaFamilyAdapter struct {
	StaticAdapter
}

type guangyaSession struct {
	ListEndpoint                string
	MetadataEndpoint            string
	CreateDirEndpoint           string
	ResCenterTokenEndpoint      string
	CheckCanFlashUploadEndpoint string
	DeleteUploadTaskEndpoint    string
	UploadInfoEndpoint          string
	Authorization               string
	ParentID                    string
	FileID                      string
	DID                         string
	DT                          string
	PageSize                    int
	ProviderKey                 string
	ExtraHeaders                map[string]string
}

type guangyaFastCheckLocalResult struct {
	Supported      bool
	HashKind       string
	NormalizedHash string
	Reason         string
	RiskHint       string
}

type guangyaBinaryUploadSession struct {
	TaskID             string
	BucketName         string
	ObjectPath         string
	FullEndpoint       string
	AccessKeyID        string
	SecretAccessKey    string
	SecurityToken      string
	RequestedMD5       string
	ResolvedTargetName string
}

var guangyaBinaryUploader = uploadGuangyaBinary

func NewGuangyaFamilyAdapter(meta Provider, capability CapabilitySet) Adapter {
	return GuangyaFamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
	}
}

func (a GuangyaFamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	session, err := a.newGuangyaSession(profile)
	if err != nil {
		return OperationResult{
			Status:  normalizeGuangyaSessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "guangya_family_real_auth",
		}
	}
	if strings.TrimSpace(session.ParentID) == "" {
		return OperationResult{
			Status:  "missing_parent_id",
			Message: "Guangya live validation requires parentId in the request or auth profile extra.parentId.",
			Mode:    "guangya_family_real_auth",
		}
	}
	statusCode, payload, requestErr := guangyaPostJSON(context.Background(), session.ListEndpoint, session.headers(), map[string]interface{}{
		"parentId": session.ParentID,
		"pageSize": session.PageSize,
		"orderBy":  0,
		"sortType": 0,
	})
	if requestErr != nil {
		return OperationResult{
			Status:  normalizeGuangyaRequestErrorStatus(requestErr),
			Message: fmt.Sprintf("Guangya live list validation request failed: %v", requestErr),
			Mode:    "guangya_family_real_auth",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "Guangya rejected the supplied authorization headers.",
			Mode:    "guangya_family_real_auth",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Guangya live validation returned HTTP %d.", statusCode),
			Mode:    "guangya_family_real_auth",
			Payload: payload,
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Guangya validated the supplied authorization against the live list endpoint.",
		Mode:    "guangya_family_real_auth",
		Payload: payload,
	}
}

func (a GuangyaFamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	session, err := a.newGuangyaSession(req.Profile)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeGuangyaSessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "guangya_family_real_directory",
			},
		}
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = session.ParentID
	}
	statusCode, payload, requestErr := guangyaPostJSON(context.Background(), session.ListEndpoint, session.headers(), map[string]interface{}{
		"parentId": parentID,
		"pageSize": guangyaClampPageSize(req.PageSize, session.PageSize),
		"orderBy":  0,
		"sortType": 0,
	})
	if requestErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeGuangyaRequestErrorStatus(requestErr),
				Message: fmt.Sprintf("Guangya list request failed: %v", requestErr),
				Mode:    "guangya_family_real_directory",
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Guangya list returned HTTP %d.", statusCode),
				Mode:    "guangya_family_real_directory",
				Payload: payload,
			},
		}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Guangya listed live directory entries.",
			Mode:    "guangya_family_real_directory",
		},
		Items: extractGuangyaItems(payload),
	}
}

func (a GuangyaFamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	session, err := a.newGuangyaSession(req.Profile)
	if err != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeGuangyaSessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "guangya_family_real_directory",
			},
		}
	}
	fileID := strings.TrimSpace(req.FileID)
	if fileID == "" {
		fileID = session.FileID
	}
	if fileID == "" {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "missing_file_id",
				Message: "Guangya live metadata requires fileId in the request or auth profile extra.fileId.",
				Mode:    "guangya_family_real_directory",
			},
		}
	}
	statusCode, payload, requestErr := guangyaPostJSON(context.Background(), session.MetadataEndpoint, session.headers(), map[string]interface{}{
		"fileId": fileID,
	})
	if requestErr != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeGuangyaRequestErrorStatus(requestErr),
				Message: fmt.Sprintf("Guangya metadata request failed: %v", requestErr),
				Mode:    "guangya_family_real_directory",
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Guangya metadata returned HTTP %d.", statusCode),
				Mode:    "guangya_family_real_directory",
				Payload: payload,
			},
		}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "exists",
			Message: "Guangya returned live metadata.",
			Mode:    "guangya_family_real_directory",
		},
		Entry: extractGuangyaMetadataEntry(fileID, payload),
	}
}

func (a GuangyaFamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	session, err := a.newGuangyaSession(req.Profile)
	if err != nil {
		return OperationResult{
			Status:  normalizeGuangyaSessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "guangya_family_real_directory",
		}
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = session.ParentID
	}
	if parentID == "" {
		return OperationResult{
			Status:  "missing_parent_id",
			Message: "Guangya create_dir requires parentId in the request or auth profile extra.parentId.",
			Mode:    "guangya_family_real_directory",
		}
	}
	dirName := strings.TrimSpace(req.DirName)
	if dirName == "" {
		return OperationResult{
			Status:  "missing_dir_name",
			Message: "Guangya create_dir requires dirName in the request.",
			Mode:    "guangya_family_real_directory",
		}
	}
	statusCode, payload, requestErr := guangyaPostJSON(context.Background(), session.CreateDirEndpoint, session.headers(), map[string]interface{}{
		"dirName":  dirName,
		"parentId": parentID,
	})
	if requestErr != nil {
		return OperationResult{
			Status:  normalizeGuangyaRequestErrorStatus(requestErr),
			Message: fmt.Sprintf("Guangya create_dir request failed: %v", requestErr),
			Mode:    "guangya_family_real_directory",
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Guangya create_dir returned HTTP %d.", statusCode),
			Mode:    "guangya_family_real_directory",
			Payload: payload,
		}
	}
	createdID := findGuangyaFirstText(payload, []string{"dirid", "dir_id", "fileid", "file_id", "id"}, 0)
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Guangya created the requested directory.",
		Mode:    "guangya_family_real_directory",
		Payload: map[string]interface{}{
			"fileId":   createdID,
			"parentId": parentID,
			"name":     dirName,
			"path":     dirName,
			"type":     "dir",
			"isDir":    true,
			"size":     int64(0),
			"raw":      payload,
		},
	}
}

func (a GuangyaFamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	session, err := a.newGuangyaSession(req.Profile)
	if err != nil {
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				Status:  normalizeGuangyaSessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "guangya_family_real_upload",
			},
		}
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = session.ParentID
	}
	if parentID == "" {
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				Status:  "missing_parent_id",
				Message: "Guangya live fast check requires parentId in the request or auth profile extra.parentId.",
				Mode:    "guangya_family_real_upload",
			},
		}
	}
	local := guangyaLocalFastCheck(req)
	row := map[string]interface{}{
		"path":             req.Path,
		"size":             req.Size,
		"supported":        local.Supported,
		"hashKind":         local.HashKind,
		"normalizedHash":   local.NormalizedHash,
		"canFastUpload":    false,
		"taskId":           "",
		"status":           0,
		"error":            "",
		"note":             local.Reason,
		"cleanupAttempted": false,
		"riskHint":         local.RiskHint,
	}
	if !local.Supported {
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: local.Reason,
				Mode:    "guangya_family_real_upload",
				Payload: row,
			},
			Candidate: false,
		}
	}
	targetName := inferName(req.Path, req.Name)
	if strings.TrimSpace(req.Name) != "" {
		targetName = strings.TrimSpace(req.Name)
	}
	body := map[string]interface{}{
		"capacity": 1,
		"name":     targetName,
		"parentId": parentID,
		"res": map[string]interface{}{
			"fileSize": req.Size,
		},
	}
	if local.HashKind == "md5" {
		body["res"].(map[string]interface{})["md5"] = local.NormalizedHash
	}
	statusCode, payload, requestErr := guangyaPostJSON(context.Background(), session.ResCenterTokenEndpoint, session.headers(), body)
	row["status"] = statusCode
	if requestErr != nil {
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				Status:  normalizeGuangyaRequestErrorStatus(requestErr),
				Message: fmt.Sprintf("Guangya live fast check request failed: %v", requestErr),
				Mode:    "guangya_family_real_upload",
				Payload: row,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Guangya live fast check returned HTTP %d.", statusCode),
				Mode:    "guangya_family_real_upload",
				Payload: row,
			},
		}
	}
	taskID := findGuangyaFirstText(payload, []string{"taskid", "task_id", "taskId"}, 0)
	row["taskId"] = taskID
	codeText := findGuangyaFirstText(payload, []string{"code"}, 0)
	if codeNumber, err := strconv.ParseInt(codeText, 10, 64); err == nil && codeNumber == guangyaCodeResTokenInstant {
		row["canFastUpload"] = true
		row["note"] = "Guangya fast-upload inventory hit succeeded via get_res_center_token."
		row["riskHint"] = ""
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: row["note"].(string),
				Mode:    "guangya_family_real_upload",
				Payload: row,
			},
			Candidate: true,
		}
	}
	if local.HashKind == "gcid" && taskID != "" {
		checkStatus, checkPayload, checkErr := guangyaPostJSON(context.Background(), session.CheckCanFlashUploadEndpoint, session.headers(), map[string]interface{}{
			"taskId": taskID,
			"gcid":   local.NormalizedHash,
		})
		row["status"] = checkStatus
		if checkErr != nil {
			return FastUploadCheckResult{
				OperationResult: OperationResult{
					Status:  normalizeGuangyaRequestErrorStatus(checkErr),
					Message: fmt.Sprintf("Guangya GCID flash-upload check failed: %v", checkErr),
					Mode:    "guangya_family_real_upload",
					Payload: row,
				},
			}
		}
		data, _ := checkPayload["data"].(map[string]interface{})
		if data != nil && boolMapValue(data, "canFlashUpload") {
			if strings.TrimSpace(session.UploadInfoEndpoint) == "" {
				row["canFastUpload"] = false
				row["note"] = "Guangya GCID flash-upload candidate was found, but uploadInfoEndpoint is not configured for final confirmation."
				row["riskHint"] = "Configure extra.uploadInfoEndpoint or let runtime fall back honestly after hash_miss."
			} else {
				row["canFastUpload"] = true
				row["note"] = "Guangya GCID flash-upload check succeeded."
				row["riskHint"] = ""
			}
			return FastUploadCheckResult{
				OperationResult: OperationResult{
					OK:      true,
					Status:  "ok",
					Message: row["note"].(string),
					Mode:    "guangya_family_real_upload",
					Payload: row,
				},
				Candidate: true,
			}
		}
		row["note"] = "Guangya GCID flash-upload check did not hit provider inventory."
	}
	if taskID != "" {
		row["cleanupAttempted"] = true
		_, _, _ = guangyaPostJSON(context.Background(), session.DeleteUploadTaskEndpoint, session.headers(), map[string]interface{}{
			"taskIds": []string{taskID},
		})
	}
	if row["note"] == local.Reason {
		row["note"] = "Guangya fast-upload inventory check did not report an available instant hit."
		row["riskHint"] = "No provider-side inventory hit was reported, so runtime should fall back honestly."
	}
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: row["note"].(string),
			Mode:    "guangya_family_real_upload",
			Payload: row,
		},
		Candidate: true,
	}
}

func (a GuangyaFamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "Guangya pending_manual items require manual confirmation before upload.",
				Mode:    "guangya_family_real_upload",
			},
		}
	}
	session, err := a.newGuangyaSession(req.Profile)
	if err != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeGuangyaSessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "guangya_family_real_upload",
			},
		}
	}
	payload := map[string]interface{}{
		"path":         req.Path,
		"parentId":     defaultPath(req.ParentID, session.ParentID),
		"name":         defaultPath(req.Name, inferName(req.Path, "file")),
		"strategy":     req.Strategy,
		"provider":     a.MetaInfo.Key,
		"hasToken":     strings.TrimSpace(session.Authorization) != "",
		"hasParentId":  strings.TrimSpace(defaultPath(req.ParentID, session.ParentID)) != "",
		"hasLocalPath": strings.TrimSpace(req.LocalPath) != "",
	}
	resolvedTargetName, conflictAction, conflictNote, conflictErr := a.resolveGuangyaUploadName(req.Profile, payload["parentId"].(string), payload["name"].(string), req.ConflictPolicy)
	if conflictErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeGuangyaRequestErrorStatus(conflictErr),
				Message: fmt.Sprintf("Guangya upload conflict preflight failed: %v", conflictErr),
				Mode:    "guangya_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}
	payload["name"] = resolvedTargetName
	payload["resolvedTargetName"] = resolvedTargetName
	payload["conflictAction"] = conflictAction
	if req.Strategy == "fast_upload" {
		check := a.FastUploadCheck(FastUploadCheckRequest{
			Profile:  req.Profile,
			Path:     req.Path,
			ParentID: req.ParentID,
			Name:     req.Name,
			Size:     req.Size,
			MD5:      req.MD5,
			GCID:     req.GCID,
		})
		payload["fastCheck"] = check.Payload
		if !check.OK {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  check.Status,
					Message: check.Message,
					Mode:    "guangya_family_real_upload",
					Payload: payload,
				},
				ConflictAction: conflictAction,
			}
		}
		if !check.Candidate || !boolMapValue(check.Payload, "canFastUpload") {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "hash_miss",
					Message: "Guangya fast-upload precheck did not produce a reusable provider-side hit.",
					Mode:    "guangya_family_real_upload",
					Payload: payload,
				},
				ConflictAction: conflictAction,
			}
		}
		fastPayload, fastErr := a.completeGuangyaFastUpload(req.Profile, session, resolvedTargetName, payload["parentId"].(string), check.Payload)
		if fastErr != nil {
			status := "provider_request_failed"
			if strings.Contains(fastErr.Error(), "upload_info endpoint") {
				status = "missing_upload_info_endpoint"
			}
			return UploadResult{
				OperationResult: OperationResult{
					Status:  status,
					Message: fastErr.Error(),
					Mode:    "guangya_family_real_upload",
					Payload: mergePayloads(payload, fastPayload),
				},
				ConflictAction: conflictAction,
			}
		}
		message := "Guangya fast-upload inventory hit completed successfully."
		if conflictNote != "" {
			message += " " + conflictNote
		}
		return UploadResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: message,
				Mode:    "guangya_family_real_upload",
				Payload: mergePayloads(payload, fastPayload),
			},
			ConflictAction: conflictAction,
		}
	}
	localPath := strings.TrimSpace(req.LocalPath)
	if localPath == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: "Guangya download_upload requires a readable local file path.",
				Mode:    "guangya_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}
	info, statErr := os.Stat(localPath)
	if statErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("Guangya could not stat local file: %v", statErr),
				Mode:    "guangya_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}
	fileSize := req.Size
	if fileSize <= 0 {
		fileSize = info.Size()
	}
	md5Value := decodeGuangyaMD5Token(req.MD5)
	uploadPayload, uploadErr := uploadGuangyaBinary(req, session, localPath, payload["parentId"].(string), resolvedTargetName, fileSize, md5Value)
	if uploadErr != nil {
		status := "provider_request_failed"
		if strings.Contains(uploadErr.Error(), "local_file_missing") {
			status = "local_file_missing"
		}
		if strings.Contains(uploadErr.Error(), "missing_gcid") {
			status = "missing_gcid"
		}
		if strings.Contains(uploadErr.Error(), "hash_miss") {
			status = "hash_miss"
		}
		return UploadResult{
			OperationResult: OperationResult{
				Status:  status,
				Message: uploadErr.Error(),
				Mode:    "guangya_family_real_upload",
				Payload: mergePayloads(payload, uploadPayload),
			},
			ConflictAction: conflictAction,
		}
	}
	verifyEntry, verifyMode, verifyOK := a.verifyGuangyaUploadedFile(req.Profile, payload["parentId"].(string), resolvedTargetName, uploadPayload)
	uploadPayload["verifyMode"] = verifyMode
	uploadPayload["verifyOk"] = verifyOK
	if verifyEntry != nil {
		uploadPayload["verifyEntry"] = verifyEntry
	}
	message := "Guangya binary upload completed through upload_token + provider-side flash check + OSS multipart runtime."
	if conflictNote != "" {
		message += " " + conflictNote
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "guangya_family_real_upload",
			Payload: mergePayloads(payload, uploadPayload),
		},
		ConflictAction: conflictAction,
	}
}

func (a GuangyaFamilyAdapter) newGuangyaSession(profile AuthProfile) (guangyaSession, error) {
	authorization := normalizeGuangyaAuthorization(profile.Token)
	if authorization == "" {
		authorization = normalizeGuangyaAuthorization(firstNonEmptyExtra(profile.Extra, "authorization", "Authorization", "token", "accessToken", "access_token"))
	}
	if authorization == "" {
		return guangyaSession{}, fmt.Errorf("Guangya adapter requires a token or authorization header")
	}
	listEndpoint, err := resolveGuangyaEndpoint(profile, "listEndpoint", guangyaAPIHost+guangyaListPath)
	if err != nil {
		return guangyaSession{}, err
	}
	metadataEndpoint, err := resolveGuangyaEndpoint(profile, "metadataEndpoint", guangyaAPIHost+guangyaDownloadMetaPath)
	if err != nil {
		return guangyaSession{}, err
	}
	createDirEndpoint, err := resolveGuangyaEndpoint(profile, "createDirEndpoint", guangyaAPIHost+guangyaCreateDirPath)
	if err != nil {
		return guangyaSession{}, err
	}
	resCenterTokenEndpoint, err := resolveGuangyaEndpoint(profile, "resCenterTokenEndpoint", guangyaAPIHost+guangyaResCenterTokenPath)
	if err != nil {
		return guangyaSession{}, err
	}
	checkCanFlashUploadEndpoint, err := resolveGuangyaEndpoint(profile, "checkCanFlashUploadEndpoint", guangyaAPIHost+guangyaCheckCanFlashUploadPath)
	if err != nil {
		return guangyaSession{}, err
	}
	deleteUploadTaskEndpoint, err := resolveGuangyaEndpoint(profile, "deleteUploadTaskEndpoint", guangyaAPIHost+guangyaDeleteUploadTaskPath)
	if err != nil {
		return guangyaSession{}, err
	}
	uploadInfoEndpoint, err := resolveGuangyaEndpoint(profile, "uploadInfoEndpoint", guangyaAPIHost+guangyaUploadInfoPath)
	if err != nil {
		return guangyaSession{}, err
	}
	return guangyaSession{
		ListEndpoint:                listEndpoint,
		MetadataEndpoint:            metadataEndpoint,
		CreateDirEndpoint:           createDirEndpoint,
		ResCenterTokenEndpoint:      resCenterTokenEndpoint,
		CheckCanFlashUploadEndpoint: checkCanFlashUploadEndpoint,
		DeleteUploadTaskEndpoint:    deleteUploadTaskEndpoint,
		UploadInfoEndpoint:          uploadInfoEndpoint,
		Authorization:               authorization,
		ParentID:                    pickGuangyaParentID(profile.Extra, ""),
		FileID:                      pickGuangyaFileID(profile.Extra, ""),
		DID:                         strings.TrimSpace(firstNonEmptyExtra(profile.Extra, "did", "deviceId")),
		DT:                          strings.TrimSpace(firstNonEmptyExtra(profile.Extra, "dt")),
		PageSize:                    pickGuangyaPageSize(profile.Extra, 100),
		ProviderKey:                 a.MetaInfo.Key,
		ExtraHeaders:                pickGuangyaExtraHeaders(profile.Extra),
	}, nil
}

func resolveGuangyaEndpoint(profile AuthProfile, key string, fallback string) (string, error) {
	raw := strings.TrimSpace(profile.Extra[key])
	if raw == "" {
		return fallback, nil
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid %s: %w", key, err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid %s: scheme and host are required", key)
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func normalizeGuangyaAuthorization(value string) string {
	text := strings.TrimSpace(value)
	if text == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(text), "bearer ") {
		return text
	}
	return "Bearer " + text
}

func pickGuangyaParentID(extra map[string]string, explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	return firstNonEmptyExtra(extra, "parentId", "parent_id", "parentFileId", "parent_file_id", "dirId", "dir_id", "pid")
}

func pickGuangyaFileID(extra map[string]string, explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	return firstNonEmptyExtra(extra, "fileId", "file_id", "resId", "res_id", "id")
}

func pickGuangyaPageSize(extra map[string]string, explicit int) int {
	if explicit > 0 {
		return explicit
	}
	if value := strings.TrimSpace(extra["pageSize"]); value != "" {
		if number, err := strconv.Atoi(value); err == nil && number > 0 {
			return number
		}
	}
	return 100
}

func pickGuangyaExtraHeaders(extra map[string]string) map[string]string {
	headers := map[string]string{}
	for key, value := range extra {
		normalized := strings.TrimSpace(key)
		if normalized == "" || strings.TrimSpace(value) == "" {
			continue
		}
		lower := strings.ToLower(normalized)
		if lower == "appid" || lower == "timestamp" || lower == "signature" || lower == "nonce" {
			headers[normalized] = strings.TrimSpace(value)
		}
	}
	return headers
}

func (s guangyaSession) headers() map[string]string {
	headers := map[string]string{
		"User-Agent":    "CloudPanSync/0.1",
		"Accept":        "application/json, text/plain, */*",
		"Content-Type":  "application/json;charset=UTF-8",
		"Authorization": s.Authorization,
	}
	if strings.TrimSpace(s.DID) != "" {
		headers["did"] = s.DID
	}
	if strings.TrimSpace(s.DT) != "" {
		headers["dt"] = s.DT
	}
	for key, value := range s.ExtraHeaders {
		headers[key] = value
	}
	return headers
}

func guangyaPostJSON(ctx context.Context, endpoint string, headers map[string]string, body interface{}) (int, map[string]interface{}, error) {
	payloadBytes, err := json.Marshal(body)
	if err != nil {
		return 0, nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(payloadBytes)))
	if err != nil {
		return 0, nil, err
	}
	for key, value := range headers {
		if strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
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

func guangyaClampPageSize(explicit int, fallback int) int {
	if explicit > 0 {
		return explicit
	}
	if fallback > 0 {
		return fallback
	}
	return 100
}

func guangyaLocalFastCheck(req FastUploadCheckRequest) guangyaFastCheckLocalResult {
	md5Value := decodeGuangyaMD5Token(defaultPath(req.MD5, ""))
	if md5Value == "" {
		md5Value = decodeGuangyaMD5Token(firstNonEmptyExtra(map[string]string{"etag": ""}, "etag"))
	}
	if md5Value != "" {
		return guangyaFastCheckLocalResult{
			Supported:      true,
			HashKind:       "md5",
			NormalizedHash: md5Value,
			Reason:         "MD5 is available and can be used for Guangya fast-upload precheck.",
			RiskHint:       "Use low concurrency and watch for anti-abuse responses.",
		}
	}
	gcid := strings.ToLower(strings.TrimSpace(req.GCID))
	if len(gcid) == 40 {
		valid := true
		for _, ch := range gcid {
			if !strings.ContainsRune("0123456789abcdef", ch) {
				valid = false
				break
			}
		}
		if valid {
			return guangyaFastCheckLocalResult{
				Supported:      true,
				HashKind:       "gcid",
				NormalizedHash: gcid,
				Reason:         "GCID looks valid and can be used for Guangya GCID precheck.",
				RiskHint:       "GCID support depends on current provider-side policy.",
			}
		}
	}
	return guangyaFastCheckLocalResult{
		Supported:      false,
		HashKind:       "missing",
		NormalizedHash: "",
		Reason:         "Neither valid MD5 nor valid 40-char GCID is available.",
		RiskHint:       "Fallback may require manual confirmation for large files.",
	}
}

func decodeGuangyaMD5Token(raw string) string {
	text := strings.Trim(strings.TrimSpace(raw), "\"")
	if text == "" {
		return ""
	}
	lower := strings.ToLower(text)
	if len(lower) == 32 {
		valid := true
		for _, ch := range lower {
			if !strings.ContainsRune("0123456789abcdef", ch) {
				valid = false
				break
			}
		}
		if valid {
			return lower
		}
	}
	normalized := strings.ReplaceAll(strings.ReplaceAll(text, "-", "+"), "_", "/")
	padding := (4 - (len(normalized) % 4)) % 4
	normalized += strings.Repeat("=", padding)
	decoded, err := base64.StdEncoding.DecodeString(normalized)
	if err != nil || len(decoded) != 16 {
		return ""
	}
	return fmt.Sprintf("%x", decoded)
}

func extractGuangyaItems(payload map[string]interface{}) []map[string]interface{} {
	rows := make([]map[string]interface{}, 0)
	scanGuangyaPossibleItems(payload, &rows, map[uintptr]bool{}, 0)
	deduped := make([]map[string]interface{}, 0, len(rows))
	seen := make(map[string]bool, len(rows))
	for _, row := range rows {
		key := fmt.Sprintf("%s::%s", stringMapValue(row, "fileId"), stringMapValue(row, "name"))
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, row)
	}
	return deduped
}

func scanGuangyaPossibleItems(node interface{}, out *[]map[string]interface{}, seen map[uintptr]bool, depth int) {
	if node == nil || depth > 6 {
		return
	}
	switch typed := node.(type) {
	case []interface{}:
		for _, item := range typed {
			scanGuangyaPossibleItems(item, out, seen, depth+1)
		}
		return
	case map[string]interface{}:
		fileID := firstNonEmptyString(typed, "fileId", "file_id", "resId", "res_id", "id")
		name := firstNonEmptyString(typed, "fileName", "filename", "name", "resName", "title")
		isDir := false
		for _, key := range []string{"isDir", "isFolder"} {
			if boolMapValue(typed, key) {
				isDir = true
				break
			}
		}
		for _, key := range []string{"dirType", "fileType", "type", "kind"} {
			value := strings.ToLower(strings.TrimSpace(stringMapValue(typed, key)))
			if value == "1" || value == "dir" || value == "folder" || value == "directory" {
				isDir = true
				break
			}
		}
		size := int64(0)
		for _, key := range []string{"fileSize", "size"} {
			if current := int64MapValue(typed, key); current != 0 {
				size = current
				break
			}
		}
		if fileID != "" && name != "" {
			*out = append(*out, map[string]interface{}{
				"exists":   true,
				"fileId":   fileID,
				"dirId":    firstNonEmptyString(typed, "dirId", "parentId", "pid"),
				"parentId": firstNonEmptyString(typed, "parentId", "pid"),
				"name":     name,
				"path":     defaultPath(firstNonEmptyString(typed, "path"), name),
				"type":     map[bool]string{true: "dir", false: "file"}[isDir],
				"isDir":    isDir,
				"size":     size,
				"md5":      strings.TrimSpace(firstNonEmptyString(typed, "md5", "etag")),
				"gcid":     strings.TrimSpace(firstNonEmptyString(typed, "gcid")),
				"provider": "guangya",
				"raw":      typed,
			})
		}
		for _, value := range typed {
			switch nested := value.(type) {
			case map[string]interface{}, []interface{}:
				scanGuangyaPossibleItems(nested, out, seen, depth+1)
			}
		}
	}
}

func findGuangyaFirstText(node interface{}, keys []string, depth int) string {
	if node == nil || depth > 6 {
		return ""
	}
	switch typed := node.(type) {
	case map[string]interface{}:
		lowered := make(map[string]interface{}, len(typed))
		for key, value := range typed {
			lowered[strings.ToLower(strings.TrimSpace(key))] = value
		}
		for _, key := range keys {
			if value, ok := lowered[strings.ToLower(key)]; ok {
				if text := strings.TrimSpace(fmt.Sprintf("%v", value)); text != "" && text != "<nil>" {
					return text
				}
			}
		}
		for _, value := range typed {
			if found := findGuangyaFirstText(value, keys, depth+1); found != "" {
				return found
			}
		}
	case []interface{}:
		for _, item := range typed {
			if found := findGuangyaFirstText(item, keys, depth+1); found != "" {
				return found
			}
		}
	}
	return ""
}

func extractGuangyaMetadataEntry(fileID string, payload map[string]interface{}) map[string]interface{} {
	md5Value := normalizeGuangyaHash(findGuangyaFirstText(payload, []string{"md5", "etag", "hash", "digest"}, 0), "md5")
	gcidValue := normalizeGuangyaHash(findGuangyaFirstText(payload, []string{"gcid", "resource_md5", "filehash", "reshash"}, 0), "gcid")
	sizeValue := int64(0)
	for _, key := range []string{"filesize", "file_size", "size", "ressize", "resource_size", "bytes", "length"} {
		text := findGuangyaFirstText(payload, []string{key}, 0)
		if text == "" {
			continue
		}
		if parsed, err := strconv.ParseInt(text, 10, 64); err == nil {
			sizeValue = parsed
			break
		}
	}
	name := findGuangyaFirstText(payload, []string{"filename", "file_name", "name", "resname", "title"}, 0)
	return map[string]interface{}{
		"exists":   true,
		"fileId":   fileID,
		"parentId": "",
		"name":     defaultPath(name, fileID),
		"path":     defaultPath(name, fileID),
		"type":     "file",
		"isDir":    false,
		"size":     sizeValue,
		"md5":      md5Value,
		"sha1":     "",
		"sha256":   "",
		"gcid":     gcidValue,
		"etag":     findGuangyaFirstText(payload, []string{"etag"}, 0),
		"provider": "guangya",
		"raw": map[string]interface{}{
			"fileId":  fileID,
			"name":    name,
			"payload": payload,
		},
	}
}

func uploadGuangyaBinary(req UploadRequest, session guangyaSession, localPath string, parentID string, resolvedTargetName string, fileSize int64, md5Value string) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"localPath":          localPath,
		"resolvedTargetName": resolvedTargetName,
		"parentId":           parentID,
		"fileSize":           fileSize,
	}
	uploadSession, tokenPayload, err := createGuangyaBinaryUploadSession(session, resolvedTargetName, parentID, localPath, fileSize, md5Value)
	payload["uploadTokenResponse"] = tokenPayload
	if err != nil {
		return payload, err
	}
	payload["taskId"] = uploadSession.TaskID
	payload["bucketName"] = uploadSession.BucketName
	payload["objectPath"] = uploadSession.ObjectPath

	if fileSize < 1024*1024 {
		infoPayload, err := guangyaUploadInfo(session, uploadSession.TaskID)
		payload["uploadInfoResponse"] = infoPayload
		if err != nil {
			return payload, err
		}
		return payload, nil
	}

	gcidValue := strings.ToUpper(strings.TrimSpace(req.GCID))
	if gcidValue == "" {
		computedGCID, computeErr := computeHashFamilyGCID(localPath)
		if computeErr != nil {
			return payload, fmt.Errorf("missing_gcid: Guangya could not compute local gcid: %w", computeErr)
		}
		gcidValue = computedGCID
	}
	payload["gcid"] = gcidValue
	flashPayload, canFlash, err := guangyaCheckCanFlashUpload(session, uploadSession.TaskID, gcidValue)
	payload["flashCheckResponse"] = flashPayload
	if err != nil {
		return payload, err
	}
	if canFlash {
		infoPayload, err := guangyaUploadInfo(session, uploadSession.TaskID)
		payload["uploadInfoResponse"] = infoPayload
		if err != nil {
			return payload, err
		}
		payload["usedBinaryFallback"] = false
		return payload, nil
	}

	multipartPayload, err := guangyaCDNUpload(localPath, uploadSession, req.ResumeUpload)
	payload["multipartUpload"] = multipartPayload
	if err != nil {
		return payload, err
	}
	payload["usedBinaryFallback"] = true

	infoPayload := map[string]interface{}{}
	for attempt := 0; attempt < 5; attempt++ {
		currentInfo, infoErr := guangyaUploadInfo(session, uploadSession.TaskID)
		payload["uploadInfoResponse"] = currentInfo
		if infoErr != nil {
			return payload, infoErr
		}
		infoPayload = currentInfo
		if strings.TrimSpace(firstNonEmptyString(currentInfo, "msg", "message")) == "文件上传中" {
			time.Sleep(2 * time.Second)
			continue
		}
		break
	}
	payload["uploadInfoResponse"] = infoPayload
	return payload, nil
}

func createGuangyaBinaryUploadSession(session guangyaSession, targetName string, parentID string, localPath string, fileSize int64, md5Value string) (guangyaBinaryUploadSession, map[string]interface{}, error) {
	body := map[string]interface{}{
		"capacity": 2,
		"name":     targetName,
		"parentId": parentID,
		"res": map[string]interface{}{
			"fileSize": fileSize,
		},
	}
	if fileSize < 1024*1024 {
		if strings.TrimSpace(md5Value) == "" {
			computedMD5, err := computeGuangyaLocalMD5(localPath)
			if err != nil {
				return guangyaBinaryUploadSession{}, nil, fmt.Errorf("provider_request_failed: Guangya could not compute local md5: %w", err)
			}
			md5Value = computedMD5
		}
		body["res"].(map[string]interface{})["md5"] = strings.ToLower(strings.TrimSpace(md5Value))
	}
	statusCode, payload, requestErr := guangyaPostJSON(context.Background(), session.ResCenterTokenEndpoint, session.headers(), body)
	if requestErr != nil {
		return guangyaBinaryUploadSession{}, payload, fmt.Errorf("provider_request_failed: Guangya upload_token request failed: %w", requestErr)
	}
	if statusCode < 200 || statusCode >= 300 {
		return guangyaBinaryUploadSession{}, payload, fmt.Errorf("provider_request_failed: Guangya upload_token returned HTTP %d", statusCode)
	}
	data, _ := payload["data"].(map[string]interface{})
	taskID := firstNonEmptyString(data, "taskId", "task_id")
	if taskID == "" {
		taskID = firstNonEmptyString(payload, "taskId", "task_id")
	}
	if taskID == "" {
		return guangyaBinaryUploadSession{}, payload, fmt.Errorf("provider_request_failed: Guangya upload_token did not return taskId")
	}
	creds, _ := data["creds"].(map[string]interface{})
	uploadSession := guangyaBinaryUploadSession{
		TaskID:             taskID,
		BucketName:         firstNonEmptyString(data, "bucketName", "bucket_name"),
		ObjectPath:         firstNonEmptyString(data, "objectPath", "object_path"),
		FullEndpoint:       firstNonEmptyString(data, "fullEndPoint", "fullEndpoint", "full_endpoint"),
		AccessKeyID:        firstNonEmptyString(creds, "accessKeyID", "accessKeyId", "access_key_id"),
		SecretAccessKey:    firstNonEmptyString(creds, "secretAccessKey", "secret_access_key"),
		SecurityToken:      firstNonEmptyString(creds, "sessionToken", "securityToken", "security_token"),
		RequestedMD5:       strings.ToLower(strings.TrimSpace(md5Value)),
		ResolvedTargetName: targetName,
	}
	return uploadSession, payload, nil
}

func guangyaCheckCanFlashUpload(session guangyaSession, taskID string, gcid string) (map[string]interface{}, bool, error) {
	statusCode, payload, requestErr := guangyaPostJSON(context.Background(), session.CheckCanFlashUploadEndpoint, session.headers(), map[string]interface{}{
		"taskId": taskID,
		"gcid":   gcid,
	})
	if requestErr != nil {
		return payload, false, fmt.Errorf("provider_request_failed: Guangya GCID flash-upload check failed: %w", requestErr)
	}
	if statusCode < 200 || statusCode >= 300 {
		return payload, false, fmt.Errorf("provider_request_failed: Guangya GCID flash-upload check returned HTTP %d", statusCode)
	}
	data, _ := payload["data"].(map[string]interface{})
	return payload, data != nil && boolMapValue(data, "canFlashUpload"), nil
}

func guangyaUploadInfo(session guangyaSession, taskID string) (map[string]interface{}, error) {
	statusCode, payload, requestErr := guangyaPostJSON(context.Background(), session.UploadInfoEndpoint, session.headers(), map[string]interface{}{
		"taskId": taskID,
	})
	if requestErr != nil {
		return payload, fmt.Errorf("provider_request_failed: Guangya upload_info request failed: %w", requestErr)
	}
	if statusCode < 200 || statusCode >= 300 {
		return payload, fmt.Errorf("provider_request_failed: Guangya upload_info returned HTTP %d", statusCode)
	}
	return payload, nil
}

func cloneGuangyaUploadedParts(parts []map[string]interface{}) []map[string]interface{} {
	if len(parts) == 0 {
		return nil
	}
	cloned := make([]map[string]interface{}, 0, len(parts))
	for _, item := range parts {
		copied := make(map[string]interface{}, len(item))
		for key, value := range item {
			copied[key] = value
		}
		cloned = append(cloned, copied)
	}
	return cloned
}

func guangyaCDNUpload(localPath string, uploadSession guangyaBinaryUploadSession, resume *ResumeUpload) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"bucketName":   uploadSession.BucketName,
		"objectPath":   uploadSession.ObjectPath,
		"fullEndpoint": uploadSession.FullEndpoint,
	}
	if uploadSession.BucketName == "" || uploadSession.ObjectPath == "" || uploadSession.FullEndpoint == "" || uploadSession.AccessKeyID == "" || uploadSession.SecretAccessKey == "" || uploadSession.SecurityToken == "" {
		return payload, fmt.Errorf("provider_request_failed: Guangya upload_token response did not include a complete OSS upload session")
	}
	uploadID := ""
	existingParts := []map[string]interface{}{}
	startPartNumber := 1
	if resume != nil {
		uploadID = strings.TrimSpace(resume.UploadID)
		existingParts = cloneGuangyaUploadedParts(resume.UploadedParts)
		if resume.NextPartNumber > 0 {
			startPartNumber = resume.NextPartNumber
		}
		if resume.FailedPartNumber > 0 {
			startPartNumber = resume.FailedPartNumber
		}
		if uploadID != "" {
			payload["uploadId"] = uploadID
			payload["uploadedParts"] = existingParts
			payload["uploadedPartCount"] = len(existingParts)
			payload["resumedUpload"] = true
		}
	}
	if uploadID == "" {
		initiateXML, initiatedUploadID, err := guangyaInitiateMultipartUpload(uploadSession)
		payload["initiateResponse"] = initiateXML
		payload["uploadId"] = initiatedUploadID
		if err != nil {
			return payload, err
		}
		uploadID = initiatedUploadID
	}
	completedParts, failedPartNumber, nextPartNumber, err := guangyaUploadMultipartParts(localPath, uploadSession, uploadID, existingParts, startPartNumber)
	payload["uploadedParts"] = completedParts
	payload["uploadedPartCount"] = len(completedParts)
	if failedPartNumber > 0 {
		payload["failedPartNumber"] = failedPartNumber
	}
	if nextPartNumber > 0 {
		payload["nextPartNumber"] = nextPartNumber
	}
	if err != nil {
		return payload, err
	}
	completeXML, completeETag, err := guangyaCompleteMultipartUpload(uploadSession, uploadID, completedParts)
	payload["completeResponse"] = completeXML
	payload["etag"] = completeETag
	if err != nil {
		return payload, err
	}
	return payload, nil
}

type guangyaMultipartPart struct {
	PartNumber int
	ETag       string
}

type guangyaMultipartUploadIDXML struct {
	UploadID string `xml:"UploadId"`
}

type guangyaMultipartCompleteXML struct {
	ETag string `xml:"ETag"`
}

func guangyaInitiateMultipartUpload(uploadSession guangyaBinaryUploadSession) (string, string, error) {
	requestURL := strings.TrimRight(uploadSession.FullEndpoint, "/") + "/" + strings.TrimLeft(uploadSession.ObjectPath, "/")
	responseBody, _, err := guangyaOSSRequest(http.MethodPost, requestURL, uploadSession, nil, "", "", map[string]string{"uploads": ""})
	if err != nil {
		return "", "", err
	}
	var parsed guangyaMultipartUploadIDXML
	if err := xml.Unmarshal([]byte(responseBody), &parsed); err != nil {
		return responseBody, "", fmt.Errorf("provider_request_failed: Guangya multipart init XML decode failed: %w", err)
	}
	if strings.TrimSpace(parsed.UploadID) == "" {
		return responseBody, "", fmt.Errorf("provider_request_failed: Guangya multipart init did not return uploadId")
	}
	return responseBody, parsed.UploadID, nil
}

func guangyaUploadMultipartParts(localPath string, uploadSession guangyaBinaryUploadSession, uploadID string, existingParts []map[string]interface{}, startPartNumber int) ([]map[string]interface{}, int, int, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return existingParts, 0, 0, fmt.Errorf("local_file_missing: Guangya could not open local file: %w", err)
	}
	defer func() { _ = file.Close() }()
	parts := cloneGuangyaUploadedParts(existingParts)
	if startPartNumber <= 0 {
		startPartNumber = 1
	}
	startOffset := int64(startPartNumber-1) * guangyaUploadChunkSize
	if _, err := file.Seek(startOffset, io.SeekStart); err != nil {
		return parts, startPartNumber, startPartNumber, fmt.Errorf("provider_request_failed: Guangya could not seek local upload chunk: %w", err)
	}
	partNumber := startPartNumber
	for {
		chunk := make([]byte, guangyaUploadChunkSize)
		readBytes, readErr := io.ReadFull(file, chunk)
		if readErr == io.EOF {
			break
		}
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return parts, partNumber, partNumber, fmt.Errorf("provider_request_failed: Guangya could not read local upload chunk: %w", readErr)
		}
		chunk = chunk[:readBytes]
		if len(chunk) == 0 {
			break
		}
		chunkSum := md5.Sum(chunk)
		chunkMD5 := base64.StdEncoding.EncodeToString(chunkSum[:])
		subResources := map[string]string{
			"partNumber": strconv.Itoa(partNumber),
			"uploadId":   uploadID,
		}
		_, headers, reqErr := guangyaOSSRequest(http.MethodPut, strings.TrimRight(uploadSession.FullEndpoint, "/")+"/"+strings.TrimLeft(uploadSession.ObjectPath, "/"), uploadSession, chunk, "application/octet-stream", chunkMD5, subResources)
		if reqErr != nil {
			return parts, partNumber, partNumber, reqErr
		}
		etag := strings.Trim(strings.TrimSpace(headers.Get("ETag")), "\"")
		if etag == "" {
			return parts, partNumber, partNumber, fmt.Errorf("provider_request_failed: Guangya multipart upload did not return an ETag for part %d", partNumber)
		}
		parts = append(parts, map[string]interface{}{
			"partNumber": partNumber,
			"etag":       etag,
		})
		partNumber++
		if readErr == io.ErrUnexpectedEOF {
			break
		}
	}
	return parts, 0, partNumber, nil
}

func guangyaCompleteMultipartUpload(uploadSession guangyaBinaryUploadSession, uploadID string, completedParts []map[string]interface{}) (string, string, error) {
	type completePart struct {
		PartNumber int    `xml:"PartNumber"`
		ETag       string `xml:"ETag"`
	}
	type completeBody struct {
		XMLName xml.Name       `xml:"CompleteMultipartUpload"`
		Parts   []completePart `xml:"Part"`
	}
	body := completeBody{Parts: make([]completePart, 0, len(completedParts))}
	for _, part := range completedParts {
		body.Parts = append(body.Parts, completePart{
			PartNumber: int(int64MapValue(part, "partNumber")),
			ETag:       `"` + strings.TrimSpace(stringMapValue(part, "etag")) + `"`,
		})
	}
	xmlBody, err := xml.Marshal(body)
	if err != nil {
		return "", "", fmt.Errorf("provider_request_failed: Guangya multipart complete body encode failed: %w", err)
	}
	xmlBytes := append([]byte(xml.Header), xmlBody...)
	xmlSum := md5.Sum(xmlBytes)
	xmlMD5 := base64.StdEncoding.EncodeToString(xmlSum[:])
	responseBody, _, reqErr := guangyaOSSRequest(http.MethodPost, strings.TrimRight(uploadSession.FullEndpoint, "/")+"/"+strings.TrimLeft(uploadSession.ObjectPath, "/"), uploadSession, xmlBytes, "application/xml", xmlMD5, map[string]string{"uploadId": uploadID})
	if reqErr != nil {
		return responseBody, "", reqErr
	}
	var parsed guangyaMultipartCompleteXML
	if err := xml.Unmarshal([]byte(responseBody), &parsed); err != nil {
		return responseBody, "", fmt.Errorf("provider_request_failed: Guangya multipart complete XML decode failed: %w", err)
	}
	return responseBody, strings.TrimSpace(parsed.ETag), nil
}

func guangyaOSSRequest(method string, requestURL string, uploadSession guangyaBinaryUploadSession, content []byte, contentType string, contentMD5 string, subResources map[string]string) (string, http.Header, error) {
	headers := guangyaOSSSignHeaders(method, uploadSession, contentType, contentMD5, subResources)
	parsedURL, err := url.Parse(requestURL)
	if err != nil {
		return "", nil, fmt.Errorf("provider_request_failed: invalid Guangya OSS URL: %w", err)
	}
	query := parsedURL.Query()
	for key, value := range subResources {
		query.Set(key, value)
	}
	parsedURL.RawQuery = query.Encode()
	req, err := http.NewRequestWithContext(context.Background(), method, parsedURL.String(), bytes.NewReader(content))
	if err != nil {
		return "", nil, fmt.Errorf("provider_request_failed: build Guangya OSS request failed: %w", err)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("provider_request_failed: Guangya OSS request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(bodyBytes), resp.Header, fmt.Errorf("provider_request_failed: Guangya OSS request returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	return string(bodyBytes), resp.Header, nil
}

func guangyaOSSSignHeaders(method string, uploadSession guangyaBinaryUploadSession, contentType string, contentMD5 string, subResources map[string]string) map[string]string {
	date := time.Now().UTC().Format(http.TimeFormat)
	ossCanonicalHeaders := map[string]string{
		"x-oss-date":           date,
		"x-oss-security-token": strings.TrimSpace(uploadSession.SecurityToken),
	}
	resource := "/" + strings.TrimLeft(uploadSession.BucketName, "/") + "/" + strings.TrimLeft(uploadSession.ObjectPath, "/")
	if len(subResources) > 0 {
		keys := make([]string, 0, len(subResources))
		for key := range subResources {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			value := subResources[key]
			if value == "" {
				parts = append(parts, key)
			} else {
				parts = append(parts, key+"="+value)
			}
		}
		resource += "?" + strings.Join(parts, "&")
	}
	headersList := make([]string, 0, len(ossCanonicalHeaders))
	for key, value := range ossCanonicalHeaders {
		headersList = append(headersList, key+":"+value)
	}
	sort.Strings(headersList)
	canonical := strings.Join([]string{
		strings.ToUpper(method),
		contentMD5,
		contentType,
		date,
		strings.Join(headersList, "\n"),
		resource,
	}, "\n")
	mac := hmac.New(sha1.New, []byte(uploadSession.SecretAccessKey))
	_, _ = mac.Write([]byte(canonical))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	headers := map[string]string{
		"authorization":        "OSS " + uploadSession.AccessKeyID + ":" + signature,
		"x-oss-date":           date,
		"x-oss-security-token": uploadSession.SecurityToken,
	}
	if contentType != "" {
		headers["content-type"] = contentType
	}
	if contentMD5 != "" {
		headers["content-md5"] = contentMD5
	}
	return headers
}

func computeGuangyaLocalMD5(localPath string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func normalizeGuangyaHash(value string, kind string) string {
	text := strings.ToLower(strings.Trim(strings.TrimSpace(value), "\""))
	if text == "" {
		return ""
	}
	if kind == "md5" && len(text) == 32 {
		for _, ch := range text {
			if !strings.ContainsRune("0123456789abcdef", ch) {
				return ""
			}
		}
		return text
	}
	if kind == "gcid" && len(text) == 40 {
		for _, ch := range text {
			if !strings.ContainsRune("0123456789abcdef", ch) {
				return ""
			}
		}
		return text
	}
	return ""
}

func normalizeGuangyaSessionErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "token or authorization") {
		return "missing_access_token"
	}
	if strings.Contains(err.Error(), "invalid ") {
		return "invalid_provider_endpoint"
	}
	return "provider_request_failed"
}

func normalizeGuangyaRequestErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "auth_invalid") {
		return "auth_invalid"
	}
	return "provider_request_failed"
}

func (a GuangyaFamilyAdapter) resolveGuangyaUploadName(profile AuthProfile, parentID string, targetName string, policy ConflictPolicy) (string, string, string, error) {
	listResult := a.List(ListRequest{
		Profile:  profile,
		ParentID: parentID,
		PageSize: 200,
	})
	if !listResult.OK {
		return targetName, "conflict_check_unavailable", "Guangya same-name conflict precheck was unavailable, so the original file name was kept.", fmt.Errorf("%s", listResult.Message)
	}
	existingNames := map[string]bool{}
	for _, item := range listResult.Items {
		name := strings.TrimSpace(stringMapValue(item, "name"))
		if name != "" {
			existingNames[name] = true
		}
	}
	if !existingNames[targetName] {
		return targetName, "none", "", nil
	}
	index := 1
	candidate := guangyaRenameCandidate(targetName, index)
	for existingNames[candidate] {
		index++
		candidate = guangyaRenameCandidate(targetName, index)
	}
	if policy == ConflictPolicyAutoRenameNew {
		return candidate, "auto_rename_new", "A same-name file already exists under the target parentId, so Guangya auto-renamed the new upload.", nil
	}
	return candidate, "overwrite_downgraded_to_auto_rename", "The requested overwrite policy was downgraded because the current Guangya upload path does not support verified in-place overwrite.", nil
}

func (a GuangyaFamilyAdapter) completeGuangyaFastUpload(profile AuthProfile, session guangyaSession, targetName string, parentID string, fastCheckPayload map[string]interface{}) (map[string]interface{}, error) {
	payload := map[string]interface{}{
		"fastUpload": fastCheckPayload,
	}
	taskID := strings.TrimSpace(stringMapValue(fastCheckPayload, "taskId"))
	if taskID != "" && strings.TrimSpace(session.UploadInfoEndpoint) != "" {
		statusCode, infoPayload, requestErr := guangyaPostJSON(context.Background(), session.UploadInfoEndpoint, session.headers(), map[string]interface{}{
			"taskId": taskID,
		})
		payload["uploadInfoStatus"] = statusCode
		payload["uploadInfoResponse"] = infoPayload
		if requestErr != nil {
			return payload, fmt.Errorf("Guangya upload_info request failed after fast-upload hit: %w", requestErr)
		}
		if statusCode < 200 || statusCode >= 300 {
			return payload, fmt.Errorf("Guangya upload_info returned HTTP %d after fast-upload hit", statusCode)
		}
	} else if taskID != "" && boolMapValue(fastCheckPayload, "canFastUpload") && strings.TrimSpace(session.UploadInfoEndpoint) == "" {
		return payload, fmt.Errorf("Guangya fast-upload hit still needs upload_info endpoint for final confirmation")
	}
	verifyEntry, verifyMode, verifyOK := a.verifyGuangyaUploadedFile(profile, parentID, targetName, payload)
	payload["verifyMode"] = verifyMode
	payload["verifyOk"] = verifyOK
	if verifyEntry != nil {
		payload["verifyEntry"] = verifyEntry
	}
	return payload, nil
}

func (a GuangyaFamilyAdapter) verifyGuangyaUploadedFile(profile AuthProfile, parentID string, targetName string, payload map[string]interface{}) (map[string]interface{}, string, bool) {
	fileID := findGuangyaFirstText(payload, []string{"fileid", "file_id", "resid", "res_id", "id"}, 0)
	if fileID != "" {
		metadataResult := a.Metadata(MetadataRequest{
			Profile: profile,
			FileID:  fileID,
		})
		if metadataResult.OK && metadataResult.Entry != nil {
			return metadataResult.Entry, "metadata_by_file_id", true
		}
	}
	listResult := a.List(ListRequest{
		Profile:  profile,
		ParentID: parentID,
		PageSize: 200,
	})
	if listResult.OK {
		for _, item := range listResult.Items {
			if strings.TrimSpace(stringMapValue(item, "name")) == targetName {
				return item, "list_by_parent_name", true
			}
		}
		return nil, "list_by_parent_name", false
	}
	return nil, "verify_unavailable", false
}

func guangyaRenameCandidate(name string, index int) string {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "file"
	}
	dot := strings.LastIndex(name, ".")
	if dot <= 0 || dot == len(name)-1 {
		return fmt.Sprintf("%s (%d)", name, index)
	}
	return fmt.Sprintf("%s (%d)%s", name[:dot], index, name[dot:])
}
