package provider

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type OpenFamilyAdapter struct {
	StaticAdapter
	RequireDomainDrive bool
}

type aliyunOpenSession struct {
	BaseEndpoint string
	DriveID      string
	Token        string
	ProviderKey  string
}

type pan123OpenSession struct {
	BaseEndpoint  string
	Authorization string
	ProfileID     string
	ProviderKey   string
}

var aliyunOpenDefaultPartSize int64 = 16 * 1024 * 1024

const pan123OpenBaseEndpoint = "https://open-api.123pan.com"

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
	token := strings.TrimSpace(profile.Token)
	if a.MetaInfo.Key == "123_open" && token == "" {
		token = strings.TrimSpace(profile.Extra["authorization"])
	}
	if token == "" {
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
	if a.MetaInfo.Key == "123_open" {
		return a.validatePan123OpenAuth(profile)
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
	if a.RequireDomainDrive {
		return a.listAliyunOpen(req)
	}
	if a.MetaInfo.Key == "123_open" {
		return a.listPan123Open(req)
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
	if a.RequireDomainDrive {
		return a.metadataAliyunOpen(req)
	}
	if a.MetaInfo.Key == "123_open" {
		return a.metadataPan123Open(req)
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

func (a OpenFamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	if a.RequireDomainDrive {
		return a.createDirAliyunOpen(req)
	}
	if a.MetaInfo.Key == "123_open" {
		return a.createDirPan123Open(req)
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
	if a.RequireDomainDrive {
		return a.fastUploadCheckAliyunOpen(req)
	}
	if a.MetaInfo.Key == "123_open" {
		candidate := strings.TrimSpace(req.MD5) != "" && req.Size > 0
		message := "123Pan Open fast-upload requires md5 and size."
		if candidate {
			message = "123Pan Open fast-upload candidate is available."
		}
		return FastUploadCheckResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: message,
				Mode:    "open_family_real_upload",
				Payload: map[string]interface{}{
					"requires": []string{"md5", "size"},
				},
			},
			Candidate: candidate,
		}
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
	if a.RequireDomainDrive {
		return a.uploadAliyunOpen(req)
	}
	if a.MetaInfo.Key == "123_open" {
		return a.uploadPan123Open(req)
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

func (a OpenFamilyAdapter) fastUploadCheckAliyunOpen(req FastUploadCheckRequest) FastUploadCheckResult {
	candidate := strings.TrimSpace(req.SHA1) != "" && req.Size > 0
	message := "Aliyun Open fast-upload requires sha1 and size."
	if candidate {
		message = "Aliyun Open fast-upload candidate is available."
	}
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "open_family_real_upload",
			Payload: map[string]interface{}{
				"requires": []string{"sha1", "size"},
			},
		},
		Candidate: candidate,
	}
}

func (a OpenFamilyAdapter) uploadAliyunOpen(req UploadRequest) UploadResult {
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "Aliyun Open pending_manual items still require follow-up runtime support.",
				Mode:    "open_family_real_upload",
			},
		}
	}

	session, err := a.newAliyunOpenSession(req.Profile)
	if err != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "open_family_real_upload",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	parentID, parentPath, parentErr := a.resolveAliyunOpenUploadParent(session, req)
	if parentErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeAliyunOpenRequestErrorStatus(parentErr),
				Message: fmt.Sprintf("Aliyun Open upload parent resolution failed: %v", parentErr),
				Mode:    "open_family_real_upload",
			},
		}
	}

	createBody := map[string]interface{}{
		"drive_id":        session.DriveID,
		"parent_file_id":  parentID,
		"name":            inferAliyunOpenUploadName(req, targetPath),
		"type":            "file",
		"size":            req.Size,
		"check_name_mode": aliyunOpenCheckNameMode(req.ConflictPolicy),
	}

	if req.ConflictPolicy == ConflictPolicyOverwriteExisting {
		if existingID, _, found, resolveErr := a.resolveAliyunOpenFileByPath(session, targetPath, 0); resolveErr == nil && found && existingID != "" {
			createBody["file_id"] = existingID
		}
	}

	var rapidAttempt bool
	if req.Strategy == "fast_upload" {
		if strings.TrimSpace(req.SHA1) == "" {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "missing_sha1",
					Message: "Aliyun Open fast upload requires sha1.",
					Mode:    "open_family_real_upload",
				},
			}
		}
		rapidAttempt = true
		createBody["content_hash"] = strings.TrimSpace(req.SHA1)
		createBody["content_hash_name"] = "sha1"
	}

	partInfoList := buildAliyunOpenPartInfoList(req.Size, aliyunOpenDefaultPartSize)
	createBody["part_info_list"] = partInfoList
	payload, resumed, resumeFallback := a.prepareAliyunOpenUploadPayload(session, req, createBody, partInfoList)
	if resumeFallback != nil {
		return *resumeFallback
	}

	if rapidAttempt && boolMapValue(payload, "rapid_upload") {
		resultPayload := a.normalizeAliyunOpenEntry(payload, pathJoin(parentPath, inferAliyunOpenUploadName(req, targetPath)))
		resultPayload["rapidUpload"] = true
		return UploadResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: "Aliyun Open rapid upload succeeded.",
				Mode:    "open_family_real_upload",
				Payload: resultPayload,
			},
			ConflictAction: "none",
		}
	}
	if rapidAttempt {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "hash_miss",
				Message: "Aliyun Open did not match the supplied sha1 for rapid upload.",
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}

	if strings.TrimSpace(req.LocalPath) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: "Aliyun Open binary upload requires a local file path.",
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}
	content, readErr := os.ReadFile(req.LocalPath)
	if readErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("Aliyun Open could not read local file: %v", readErr),
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}

	partItems := partInfoMapSlice(payload, "part_info_list")
	if len(partItems) == 0 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: "Aliyun Open upload create did not return part upload URLs.",
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}
	knownUploadedParts := []map[string]interface{}(nil)
	totalPartCount := len(partItems)
	if req.ResumeUpload != nil {
		knownUploadedParts = req.ResumeUpload.UploadedParts
		if req.ResumeUpload.PartCount > totalPartCount {
			totalPartCount = req.ResumeUpload.PartCount
		}
	}
	partPayloads, putHeaders, uploadPartsErr := a.uploadAliyunOpenParts(partItems, content, int64(len(content)), aliyunOpenDefaultPartSize, knownUploadedParts, totalPartCount)
	if uploadPartsErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Aliyun Open upload part request failed: %v", uploadPartsErr),
				Mode:    "open_family_real_upload",
				Payload: mergeMaps(payload, partPayloads),
			},
		}
	}

	fileID := firstNonEmptyString(payload, "file_id", "fileId")
	uploadID := firstNonEmptyString(payload, "upload_id", "uploadId")
	completeStatus, completePayload, completeErr := postProviderJSON(context.Background(), session.BaseEndpoint+"/v2/file/complete", session.Token, map[string]interface{}{
		"drive_id":  session.DriveID,
		"file_id":   fileID,
		"upload_id": uploadID,
	})
	if completeErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Aliyun Open complete upload request failed: %v", completeErr),
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}
	if completeStatus == http.StatusUnauthorized || completeStatus == http.StatusForbidden {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "Aliyun Open rejected the supplied access token while completing an upload.",
				Mode:    "open_family_real_upload",
				Payload: completePayload,
			},
		}
	}
	if completeStatus < 200 || completeStatus >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Aliyun Open complete upload returned HTTP %d.", completeStatus),
				Mode:    "open_family_real_upload",
				Payload: completePayload,
			},
		}
	}

	resultPayload := a.normalizeAliyunOpenEntry(completePayload, pathJoin(parentPath, inferAliyunOpenUploadName(req, targetPath)))
	if etag := strings.TrimSpace(putHeaders.Get("ETag")); etag != "" {
		resultPayload["etag"] = strings.Trim(etag, "\"")
	}
	resultPayload["uploadId"] = uploadID
	resultPayload["fileId"] = fileID
	resultPayload["rapidUpload"] = false
	resultPayload["partCount"] = len(partItems)
	if resumed {
		resultPayload["resumedUpload"] = true
	}
	for key, value := range partPayloads {
		resultPayload[key] = value
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Aliyun Open upload succeeded.",
			Mode:    "open_family_real_upload",
			Payload: resultPayload,
		},
		ConflictAction: "none",
	}
}

func (a OpenFamilyAdapter) uploadAliyunOpenParts(partItems []map[string]interface{}, content []byte, totalSize int64, partSize int64, knownUploadedParts []map[string]interface{}, totalPartCount int) (map[string]interface{}, http.Header, error) {
	headers := http.Header{}
	if totalPartCount <= 0 {
		totalPartCount = len(partItems)
	}
	partPayloads := map[string]interface{}{
		"partCount": totalPartCount,
	}
	if len(partItems) == 0 {
		return partPayloads, headers, nil
	}
	uploadedParts := cloneAliyunOpenUploadedParts(knownUploadedParts)
	for idx, item := range partItems {
		partNumber := idx + 1
		if explicit := int(int64MapValue(item, "part_number")); explicit > 0 {
			partNumber = explicit
		}
		uploadURL := firstNonEmptyString(item, "upload_url", "internal_upload_url")
		if uploadURL == "" {
			partPayloads["failedPartNumber"] = partNumber
			partPayloads["nextPartNumber"] = partNumber
			partPayloads["uploadedParts"] = uploadedParts
			return partPayloads, headers, fmt.Errorf("part %d missing upload_url", partNumber)
		}
		start, end := aliyunOpenContentRange(totalSize, partSize, partNumber)
		if start < 0 || end < start || end > len(content) {
			partPayloads["failedPartNumber"] = partNumber
			partPayloads["nextPartNumber"] = partNumber
			partPayloads["uploadedParts"] = uploadedParts
			return partPayloads, headers, fmt.Errorf("part %d resolved invalid content range %d:%d", partNumber, start, end)
		}
		chunk := content[start:end]
		putStatus, putHeaders, putErr := putProviderBytes(context.Background(), uploadURL, chunk, map[string]string{
			"Content-Length": strconv.FormatInt(int64(len(chunk)), 10),
		})
		if putErr != nil {
			partPayloads["failedPartNumber"] = partNumber
			partPayloads["nextPartNumber"] = partNumber
			partPayloads["uploadedParts"] = uploadedParts
			return partPayloads, headers, putErr
		}
		if putStatus < 200 || putStatus >= 300 {
			partPayloads["failedPartNumber"] = partNumber
			partPayloads["nextPartNumber"] = partNumber
			partPayloads["failedPartStatus"] = putStatus
			partPayloads["uploadedParts"] = uploadedParts
			return partPayloads, putHeaders, fmt.Errorf("part %d returned HTTP %d", partNumber, putStatus)
		}
		partMeta := map[string]interface{}{
			"partNumber": partNumber,
			"start":      start,
			"end":        end,
			"size":       len(chunk),
		}
		if etag := strings.TrimSpace(putHeaders.Get("ETag")); etag != "" {
			partMeta["etag"] = strings.Trim(etag, "\"")
		}
		uploadedParts = append(uploadedParts, partMeta)
		if idx == len(partItems)-1 {
			headers = putHeaders
		}
	}
	partPayloads["uploadedPartCount"] = len(uploadedParts)
	partPayloads["uploadedParts"] = uploadedParts
	partPayloads["nextPartNumber"] = highestUploadedAliyunOpenPartNumber(uploadedParts) + 1
	return partPayloads, headers, nil
}

func highestUploadedAliyunOpenPartNumber(parts []map[string]interface{}) int {
	maxPartNumber := 0
	for _, item := range parts {
		partNumber := int(int64MapValue(item, "partNumber"))
		if partNumber <= 0 {
			partNumber = int(int64MapValue(item, "part_number"))
		}
		if partNumber > maxPartNumber {
			maxPartNumber = partNumber
		}
	}
	return maxPartNumber
}

func cloneAliyunOpenUploadedParts(parts []map[string]interface{}) []map[string]interface{} {
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

func (a OpenFamilyAdapter) prepareAliyunOpenUploadPayload(session aliyunOpenSession, req UploadRequest, createBody map[string]interface{}, partInfoList []map[string]interface{}) (map[string]interface{}, bool, *UploadResult) {
	if resume := req.ResumeUpload; resume != nil {
		payload, ok, result := a.resumeAliyunOpenUpload(session, *resume, partInfoList)
		if result != nil {
			return nil, false, result
		}
		if ok {
			return payload, true, nil
		}
	}
	statusCode, payload, createErr := postProviderJSON(context.Background(), session.BaseEndpoint+"/adrive/v1.0/openFile/create", session.Token, createBody)
	if createErr != nil {
		return nil, false, &UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Aliyun Open upload create request failed: %v", createErr),
				Mode:    "open_family_real_upload",
			},
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return nil, false, &UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "Aliyun Open rejected the supplied access token while creating an upload.",
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, false, &UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Aliyun Open upload create returned HTTP %d.", statusCode),
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}
	return payload, false, nil
}

func (a OpenFamilyAdapter) resumeAliyunOpenUpload(session aliyunOpenSession, resume ResumeUpload, partInfoList []map[string]interface{}) (map[string]interface{}, bool, *UploadResult) {
	if strings.TrimSpace(resume.FileID) == "" || strings.TrimSpace(resume.UploadID) == "" {
		return nil, false, nil
	}
	remaining := remainingAliyunOpenPartInfoList(resume, partInfoList)
	if len(remaining) == 0 {
		return nil, false, nil
	}
	statusCode, payload, err := postProviderJSON(context.Background(), session.BaseEndpoint+"/v2/file/get_upload_url", session.Token, map[string]interface{}{
		"drive_id":       session.DriveID,
		"file_id":        resume.FileID,
		"upload_id":      resume.UploadID,
		"part_info_list": remaining,
	})
	if err != nil {
		return nil, false, nil
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return nil, false, &UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "Aliyun Open rejected the supplied access token while refreshing upload URLs.",
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return nil, false, nil
	}
	if firstNonEmptyString(payload, "file_id", "fileId") == "" {
		payload["file_id"] = resume.FileID
	}
	if firstNonEmptyString(payload, "upload_id", "uploadId") == "" {
		payload["upload_id"] = resume.UploadID
	}
	payload["resumedUpload"] = true
	return payload, true, nil
}

func buildAliyunOpenPartInfoList(size int64, partSize int64) []map[string]interface{} {
	if partSize <= 0 {
		partSize = aliyunOpenDefaultPartSize
	}
	count := 1
	if size > 0 {
		count = int(math.Ceil(float64(size) / float64(partSize)))
		if count < 1 {
			count = 1
		}
	}
	items := make([]map[string]interface{}, 0, count)
	for i := 1; i <= count; i++ {
		items = append(items, map[string]interface{}{
			"part_number": i,
		})
	}
	return items
}

func aliyunOpenContentRange(totalSize int64, partSize int64, partNumber int) (int, int) {
	if partNumber <= 0 || totalSize < 0 {
		return -1, -1
	}
	if partSize <= 0 {
		partSize = aliyunOpenDefaultPartSize
	}
	start := int64(partNumber-1) * partSize
	if start > totalSize {
		return -1, -1
	}
	end := start + partSize
	if end > totalSize {
		end = totalSize
	}
	return int(start), int(end)
}

func remainingAliyunOpenPartInfoList(resume ResumeUpload, partInfoList []map[string]interface{}) []map[string]interface{} {
	if len(partInfoList) == 0 {
		return nil
	}
	uploaded := make(map[int]bool, len(resume.UploadedParts))
	for _, item := range resume.UploadedParts {
		partNumber := int(int64MapValue(item, "partNumber"))
		if partNumber <= 0 {
			partNumber = int(int64MapValue(item, "part_number"))
		}
		if partNumber > 0 {
			uploaded[partNumber] = true
		}
	}
	startPart := resume.NextPartNumber
	if startPart <= 0 {
		startPart = resume.FailedPartNumber
	}
	remaining := make([]map[string]interface{}, 0, len(partInfoList))
	for _, item := range partInfoList {
		partNumber := int(int64MapValue(item, "part_number"))
		if partNumber <= 0 {
			continue
		}
		if uploaded[partNumber] {
			continue
		}
		if startPart > 0 && partNumber < startPart {
			continue
		}
		remaining = append(remaining, map[string]interface{}{"part_number": partNumber})
	}
	return remaining
}

func (a OpenFamilyAdapter) listAliyunOpen(req ListRequest) ListResult {
	session, err := a.newAliyunOpenSession(req.Profile)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "open_family_real_directory",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	fileID := strings.TrimSpace(req.ParentID)
	if fileID == "" {
		fileID = "root"
		if targetPath != "/" {
			_, resolvedEntry, found, resolveErr := a.resolveAliyunOpenFileByPath(session, targetPath, req.PageSize)
			if resolveErr != nil {
				return ListResult{
					OperationResult: OperationResult{
						Status:  "provider_request_failed",
						Message: fmt.Sprintf("Aliyun Open path resolution failed: %v", resolveErr),
						Mode:    "open_family_real_directory",
					},
				}
			}
			if !found {
				return ListResult{
					OperationResult: OperationResult{
						Status:  "path_not_found",
						Message: fmt.Sprintf("Aliyun Open path %q was not found.", targetPath),
						Mode:    "open_family_real_directory",
					},
				}
			}
			if !boolMapValue(resolvedEntry, "isDir") {
				return ListResult{
					OperationResult: OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "Aliyun Open resolved a file path directly.",
						Mode:    "open_family_real_directory",
					},
					Items: []map[string]interface{}{resolvedEntry},
				}
			}
			fileID = strings.TrimSpace(stringMapValue(resolvedEntry, "fileId"))
		}
	}

	items, err := a.listAliyunOpenByParent(session, fileID, targetPath, req.PageSize)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeAliyunOpenRequestErrorStatus(err),
				Message: fmt.Sprintf("Aliyun Open list request failed: %v", err),
				Mode:    "open_family_real_directory",
			},
		}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Aliyun Open listed live directory entries.",
			Mode:    "open_family_real_directory",
		},
		Items: items,
	}
}

func (a OpenFamilyAdapter) metadataAliyunOpen(req MetadataRequest) MetadataResult {
	session, err := a.newAliyunOpenSession(req.Profile)
	if err != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "open_family_real_directory",
			},
		}
	}

	if fileID := strings.TrimSpace(req.FileID); fileID != "" {
		entry, getErr := a.getAliyunOpenEntryByID(session, fileID, normalizeOpenFamilyPath(req.Path))
		if getErr != nil {
			return MetadataResult{
				OperationResult: OperationResult{
					Status:  normalizeAliyunOpenRequestErrorStatus(getErr),
					Message: fmt.Sprintf("Aliyun Open metadata request failed: %v", getErr),
					Mode:    "open_family_real_directory",
				},
			}
		}
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: "Aliyun Open returned live metadata.",
				Mode:    "open_family_real_directory",
			},
			Entry: entry,
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	if targetPath == "/" {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: "Aliyun Open root directory metadata is available.",
				Mode:    "open_family_real_directory",
			},
			Entry: map[string]interface{}{
				"exists":   true,
				"isDir":    true,
				"name":     "/",
				"path":     "/",
				"fileId":   "root",
				"provider": a.MetaInfo.Key,
			},
		}
	}

	_, entry, found, resolveErr := a.resolveAliyunOpenFileByPath(session, targetPath, 0)
	if resolveErr != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeAliyunOpenRequestErrorStatus(resolveErr),
				Message: fmt.Sprintf("Aliyun Open path resolution failed: %v", resolveErr),
				Mode:    "open_family_real_directory",
			},
		}
	}
	if !found {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "missing",
				Message: "Aliyun Open did not find the requested path.",
				Mode:    "open_family_real_directory",
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
			Message: "Aliyun Open returned live metadata.",
			Mode:    "open_family_real_directory",
		},
		Entry: entry,
	}
}

func (a OpenFamilyAdapter) createDirAliyunOpen(req CreateDirRequest) OperationResult {
	session, err := a.newAliyunOpenSession(req.Profile)
	if err != nil {
		return OperationResult{
			Status:  "invalid_provider_endpoint",
			Message: err.Error(),
			Mode:    "open_family_real_directory",
		}
	}

	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = "root"
	}
	statusCode, payload, requestErr := postProviderJSON(context.Background(), session.BaseEndpoint+"/adrive/v1.0/openFile/create", session.Token, map[string]interface{}{
		"drive_id":        session.DriveID,
		"parent_file_id":  parentID,
		"name":            strings.TrimSpace(req.DirName),
		"type":            "folder",
		"check_name_mode": "auto_rename",
	})
	if requestErr != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Aliyun Open create-dir request failed: %v", requestErr),
			Mode:    "open_family_real_directory",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "Aliyun Open rejected the supplied access token while creating a directory.",
			Mode:    "open_family_real_directory",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Aliyun Open create-dir returned HTTP %d.", statusCode),
			Mode:    "open_family_real_directory",
			Payload: payload,
		}
	}

	entry := a.normalizeAliyunOpenEntry(payload, pathJoin("/", strings.TrimSpace(req.DirName)))
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Aliyun Open created the requested directory.",
		Mode:    "open_family_real_directory",
		Payload: entry,
	}
}

func (a OpenFamilyAdapter) newAliyunOpenSession(profile AuthProfile) (aliyunOpenSession, error) {
	domainID := strings.TrimSpace(profile.Extra["domainId"])
	baseEndpoint, err := resolveOpenFamilyEndpoint(profile, domainID)
	if err != nil {
		return aliyunOpenSession{}, err
	}
	return aliyunOpenSession{
		BaseEndpoint: baseEndpoint,
		DriveID:      strings.TrimSpace(profile.Extra["driveId"]),
		Token:        strings.TrimSpace(profile.Token),
		ProviderKey:  a.MetaInfo.Key,
	}, nil
}

func (a OpenFamilyAdapter) resolveAliyunOpenFileByPath(session aliyunOpenSession, path string, pageSize int) (string, map[string]interface{}, bool, error) {
	normalized := normalizeOpenFamilyPath(path)
	if normalized == "/" {
		return "root", map[string]interface{}{
			"exists":   true,
			"isDir":    true,
			"name":     "/",
			"path":     "/",
			"fileId":   "root",
			"provider": session.ProviderKey,
		}, true, nil
	}

	currentID := "root"
	currentPath := "/"
	var currentEntry map[string]interface{}
	parts := strings.Split(strings.Trim(normalized, "/"), "/")
	for _, part := range parts {
		children, err := a.listAliyunOpenByParent(session, currentID, currentPath, pageSize)
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

func (a OpenFamilyAdapter) listAliyunOpenByParent(session aliyunOpenSession, parentFileID string, basePath string, pageSize int) ([]map[string]interface{}, error) {
	limit := pageSize
	if limit <= 0 {
		limit = 200
	}
	marker := ""
	items := make([]map[string]interface{}, 0)
	for {
		body := map[string]interface{}{
			"drive_id":       session.DriveID,
			"parent_file_id": parentFileID,
			"limit":          limit,
		}
		if marker != "" {
			body["marker"] = marker
		}
		statusCode, payload, err := postProviderJSON(context.Background(), session.BaseEndpoint+"/adrive/v1.0/openFile/list", session.Token, body)
		if err != nil {
			return nil, err
		}
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			return nil, fmt.Errorf("auth_invalid")
		}
		if statusCode < 200 || statusCode >= 300 {
			return nil, fmt.Errorf("http %d", statusCode)
		}
		for _, raw := range interfaceSliceValue(payload, "items") {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			items = append(items, a.normalizeAliyunOpenEntry(item, pathJoin(basePath, stringMapValue(item, "name"))))
		}
		nextMarker := firstNonEmptyString(payload, "next_marker", "nextMarker")
		if nextMarker == "" {
			return items, nil
		}
		marker = nextMarker
	}
}

func (a OpenFamilyAdapter) getAliyunOpenEntryByID(session aliyunOpenSession, fileID string, path string) (map[string]interface{}, error) {
	statusCode, payload, err := postProviderJSON(context.Background(), session.BaseEndpoint+"/adrive/v1.0/openFile/get", session.Token, map[string]interface{}{
		"drive_id": session.DriveID,
		"file_id":  strings.TrimSpace(fileID),
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
	return a.normalizeAliyunOpenEntry(payload, path), nil
}

func (a OpenFamilyAdapter) normalizeAliyunOpenEntry(raw map[string]interface{}, path string) map[string]interface{} {
	entryType := strings.ToLower(firstNonEmptyString(raw, "type"))
	isDir := entryType == "folder" || entryType == "dir" || entryType == "directory" || boolMapValue(raw, "isDir")
	md5 := firstNonEmptyString(raw, "md5")
	sha1 := firstNonEmptyString(raw, "sha1")
	contentHash := firstNonEmptyString(raw, "content_hash", "contentHash")
	switch strings.ToLower(firstNonEmptyString(raw, "content_hash_name", "contentHashName")) {
	case "md5":
		if md5 == "" {
			md5 = contentHash
		}
	case "sha1":
		if sha1 == "" {
			sha1 = contentHash
		}
	}

	entry := map[string]interface{}{
		"exists":   true,
		"name":     firstNonEmptyString(raw, "name"),
		"path":     normalizeOpenFamilyPath(path),
		"fileId":   firstNonEmptyString(raw, "file_id", "fileId"),
		"parentId": firstNonEmptyString(raw, "parent_file_id", "parentId"),
		"isDir":    isDir,
		"size":     int64MapValue(raw, "size"),
		"provider": a.MetaInfo.Key,
	}
	if md5 != "" {
		entry["md5"] = md5
	}
	if sha1 != "" {
		entry["sha1"] = sha1
	}
	if etag := firstNonEmptyString(raw, "etag"); etag != "" {
		entry["etag"] = etag
	}
	if gcid := firstNonEmptyString(raw, "gcid"); gcid != "" {
		entry["gcid"] = gcid
	}
	return entry
}

func (a OpenFamilyAdapter) resolveAliyunOpenUploadParent(session aliyunOpenSession, req UploadRequest) (string, string, error) {
	if parentID := strings.TrimSpace(req.ParentID); parentID != "" {
		parentPath := normalizeOpenFamilyPath(parentDirectory(req.Path))
		if parentPath == "." || parentPath == "" {
			parentPath = "/"
		}
		return parentID, parentPath, nil
	}

	parentPath := normalizeOpenFamilyPath(parentDirectory(req.Path))
	if parentPath == "." || parentPath == "" {
		parentPath = "/"
	}
	if parentPath == "/" {
		return "root", "/", nil
	}
	parentID, _, found, err := a.resolveAliyunOpenFileByPath(session, parentPath, 0)
	if err != nil {
		return "", "", err
	}
	if !found || parentID == "" {
		return "", "", fmt.Errorf("parent_path_not_found")
	}
	return parentID, parentPath, nil
}

func inferAliyunOpenUploadName(req UploadRequest, targetPath string) string {
	if name := strings.TrimSpace(req.Name); name != "" {
		return name
	}
	return inferName(targetPath, "remote.bin")
}

func aliyunOpenCheckNameMode(policy ConflictPolicy) string {
	switch policy {
	case ConflictPolicyOverwriteExisting:
		return "overwrite"
	case ConflictPolicyAutoRenameNew:
		return "auto_rename"
	default:
		return "auto_rename"
	}
}

func partInfoMapSlice(values map[string]interface{}, key string) []map[string]interface{} {
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

func parentDirectory(path string) string {
	normalized := normalizeOpenFamilyPath(path)
	if normalized == "/" {
		return "/"
	}
	idx := strings.LastIndex(normalized, "/")
	if idx <= 0 {
		return "/"
	}
	return normalized[:idx]
}

func normalizeOpenFamilyPath(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "/"
	}
	trimmed = strings.ReplaceAll(trimmed, "\\", "/")
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	for strings.Contains(trimmed, "//") {
		trimmed = strings.ReplaceAll(trimmed, "//", "/")
	}
	if len(trimmed) > 1 && strings.HasSuffix(trimmed, "/") {
		trimmed = strings.TrimRight(trimmed, "/")
	}
	return trimmed
}

func pathJoin(base string, name string) string {
	base = normalizeOpenFamilyPath(base)
	name = strings.Trim(strings.TrimSpace(name), "/")
	if name == "" {
		return base
	}
	if base == "/" {
		return "/" + name
	}
	return base + "/" + name
}

func interfaceSliceValue(values map[string]interface{}, key string) []interface{} {
	if values == nil {
		return nil
	}
	raw, ok := values[key]
	if !ok {
		return nil
	}
	items, _ := raw.([]interface{})
	return items
}

func firstNonEmptyString(values map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if value := stringMapValue(values, key); value != "" {
			return value
		}
	}
	return ""
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

func mergeMaps(base map[string]interface{}, overlay map[string]interface{}) map[string]interface{} {
	out := map[string]interface{}{}
	for key, value := range base {
		out[key] = value
	}
	for key, value := range overlay {
		out[key] = value
	}
	return out
}

func stringMapValue(values map[string]interface{}, key string) string {
	if values == nil {
		return ""
	}
	switch value := values[key].(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	case float64:
		return strings.TrimSpace(strconv.FormatInt(int64(value), 10))
	case int:
		return strings.TrimSpace(strconv.Itoa(value))
	case int64:
		return strings.TrimSpace(strconv.FormatInt(value, 10))
	default:
		return ""
	}
}

func int64MapValue(values map[string]interface{}, key string) int64 {
	if values == nil {
		return 0
	}
	switch value := values[key].(type) {
	case int:
		return int64(value)
	case int32:
		return int64(value)
	case int64:
		return value
	case float32:
		return int64(value)
	case float64:
		return int64(value)
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err == nil {
			return parsed
		}
	}
	return 0
}

func boolMapValue(values map[string]interface{}, key string) bool {
	if values == nil {
		return false
	}
	switch value := values[key].(type) {
	case bool:
		return value
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		return err == nil && parsed
	default:
		return false
	}
}

func (a OpenFamilyAdapter) validatePan123OpenAuth(profile AuthProfile) OperationResult {
	session, err := a.newPan123OpenSession(profile)
	if err != nil {
		return OperationResult{
			Status:  "invalid_provider_endpoint",
			Message: err.Error(),
			Mode:    "open_family_real_auth",
		}
	}

	statusCode, payload, requestErr := getPan123OpenJSON(context.Background(), session, "/api/v2/file/list", map[string]string{
		"parentFileId": "0",
		"limit":        "1",
		"lastFileId":   "0",
	})
	if requestErr != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("123Pan Open auth validation request failed: %v", requestErr),
			Mode:    "open_family_real_auth",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "123Pan Open rejected the supplied access token.",
			Mode:    "open_family_real_auth",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("123Pan Open auth validation returned HTTP %d.", statusCode),
			Mode:    "open_family_real_auth",
			Payload: payload,
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "123Pan Open validated the supplied access token against the live list endpoint.",
		Mode:    "open_family_real_auth",
		Payload: payload,
	}
}

func (a OpenFamilyAdapter) listPan123Open(req ListRequest) ListResult {
	session, err := a.newPan123OpenSession(req.Profile)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "open_family_real_directory",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	parentID := strings.TrimSpace(req.ParentID)
	basePath := targetPath
	if parentID == "" {
		if targetPath == "/" {
			parentID = "0"
			basePath = "/"
		} else {
			resolvedParentID, resolvedEntry, found, resolveErr := a.resolvePan123OpenEntryByPath(session, targetPath, req.PageSize)
			if resolveErr != nil {
				return ListResult{
					OperationResult: OperationResult{
						Status:  normalizePan123OpenRequestErrorStatus(resolveErr),
						Message: fmt.Sprintf("123Pan Open path resolution failed: %v", resolveErr),
						Mode:    "open_family_real_directory",
					},
				}
			}
			if !found {
				return ListResult{
					OperationResult: OperationResult{
						Status:  "path_not_found",
						Message: fmt.Sprintf("123Pan Open path %q was not found.", targetPath),
						Mode:    "open_family_real_directory",
					},
				}
			}
			if !boolMapValue(resolvedEntry, "isDir") {
				return ListResult{
					OperationResult: OperationResult{
						OK:      true,
						Status:  "ok",
						Message: "123Pan Open resolved a file path directly.",
						Mode:    "open_family_real_directory",
					},
					Items: []map[string]interface{}{resolvedEntry},
				}
			}
			parentID = resolvedParentID
			basePath = targetPath
		}
	}

	items, listErr := a.listPan123OpenByParent(session, parentID, basePath, req.PageSize)
	if listErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizePan123OpenRequestErrorStatus(listErr),
				Message: fmt.Sprintf("123Pan Open list request failed: %v", listErr),
				Mode:    "open_family_real_directory",
			},
		}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "123Pan Open listed live directory entries.",
			Mode:    "open_family_real_directory",
		},
		Items: items,
	}
}

func (a OpenFamilyAdapter) metadataPan123Open(req MetadataRequest) MetadataResult {
	session, err := a.newPan123OpenSession(req.Profile)
	if err != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "open_family_real_directory",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	if targetPath == "/" && strings.TrimSpace(req.FileID) == "" {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: "123Pan Open root directory metadata is available.",
				Mode:    "open_family_real_directory",
			},
			Entry: map[string]interface{}{
				"exists":   true,
				"isDir":    true,
				"name":     "/",
				"path":     "/",
				"fileId":   "0",
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
		if parentPath == "/" {
			parentID = "0"
		} else {
			resolvedParentID, _, found, resolveErr := a.resolvePan123OpenEntryByPath(session, parentPath, 0)
			if resolveErr != nil {
				return MetadataResult{
					OperationResult: OperationResult{
						Status:  normalizePan123OpenRequestErrorStatus(resolveErr),
						Message: fmt.Sprintf("123Pan Open parent path resolution failed: %v", resolveErr),
						Mode:    "open_family_real_directory",
					},
				}
			}
			if !found {
				return MetadataResult{
					OperationResult: OperationResult{
						OK:      true,
						Status:  "missing",
						Message: "123Pan Open did not find the requested parent path.",
						Mode:    "open_family_real_directory",
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

	items, listErr := a.listPan123OpenByParent(session, parentID, normalizeOpenFamilyPath(parentDirectory(targetPath)), 0)
	if listErr != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizePan123OpenRequestErrorStatus(listErr),
				Message: fmt.Sprintf("123Pan Open metadata list request failed: %v", listErr),
				Mode:    "open_family_real_directory",
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
					Message: "123Pan Open returned live metadata.",
					Mode:    "open_family_real_directory",
				},
				Entry: item,
			}
		}
		if targetFileID == "" && strings.TrimSpace(stringMapValue(item, "name")) == targetName {
			return MetadataResult{
				OperationResult: OperationResult{
					OK:      true,
					Status:  "exists",
					Message: "123Pan Open returned live metadata.",
					Mode:    "open_family_real_directory",
				},
				Entry: item,
			}
		}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "missing",
			Message: "123Pan Open did not find the requested path.",
			Mode:    "open_family_real_directory",
		},
		Entry: map[string]interface{}{
			"exists":   false,
			"path":     targetPath,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a OpenFamilyAdapter) createDirPan123Open(req CreateDirRequest) OperationResult {
	session, err := a.newPan123OpenSession(req.Profile)
	if err != nil {
		return OperationResult{
			Status:  "invalid_provider_endpoint",
			Message: err.Error(),
			Mode:    "open_family_real_directory",
		}
	}

	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = "0"
	}
	statusCode, payload, requestErr := postPan123OpenJSON(context.Background(), session, "/upload/v1/file/mkdir", map[string]interface{}{
		"name":     strings.TrimSpace(req.DirName),
		"parentID": parentID,
	})
	if requestErr != nil {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("123Pan Open create-dir request failed: %v", requestErr),
			Mode:    "open_family_real_directory",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "123Pan Open rejected the supplied access token while creating a directory.",
			Mode:    "open_family_real_directory",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("123Pan Open create-dir returned HTTP %d.", statusCode),
			Mode:    "open_family_real_directory",
			Payload: payload,
		}
	}

	data, _ := payload["data"].(map[string]interface{})
	entry := map[string]interface{}{
		"exists":   true,
		"name":     strings.TrimSpace(req.DirName),
		"path":     pathJoin("/", strings.TrimSpace(req.DirName)),
		"fileId":   firstNonEmptyString(data, "dirID", "dirId", "fileID", "fileId"),
		"parentId": parentID,
		"isDir":    true,
		"size":     int64(0),
		"provider": a.MetaInfo.Key,
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "123Pan Open created the requested directory.",
		Mode:    "open_family_real_directory",
		Payload: entry,
	}
}

func (a OpenFamilyAdapter) uploadPan123Open(req UploadRequest) UploadResult {
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "123Pan Open pending_manual items still require follow-up runtime support.",
				Mode:    "open_family_real_upload",
			},
		}
	}

	session, err := a.newPan123OpenSession(req.Profile)
	if err != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "invalid_provider_endpoint",
				Message: err.Error(),
				Mode:    "open_family_real_upload",
			},
		}
	}
	if req.Strategy == "fast_upload" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "hash_miss",
				Message: "123Pan Open fast-upload check is available, but provider-side fast upload execution is not implemented yet.",
				Mode:    "open_family_real_upload",
			},
			ConflictAction: "none",
		}
	}
	if strings.TrimSpace(req.LocalPath) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: "123Pan Open binary upload requires a local file path.",
				Mode:    "open_family_real_upload",
			},
		}
	}

	content, readErr := os.ReadFile(req.LocalPath)
	if readErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("123Pan Open could not read local file: %v", readErr),
				Mode:    "open_family_real_upload",
			},
		}
	}
	if req.Size <= 0 {
		req.Size = int64(len(content))
	}

	parentID, parentPath, parentErr := a.resolvePan123OpenUploadParent(session, req)
	if parentErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizePan123OpenRequestErrorStatus(parentErr),
				Message: fmt.Sprintf("123Pan Open upload parent resolution failed: %v", parentErr),
				Mode:    "open_family_real_upload",
			},
		}
	}

	targetPath := normalizeOpenFamilyPath(req.Path)
	resolvedTargetName, conflictAction, conflictNote, conflictErr := a.resolvePan123OpenUploadName(session, parentID, inferAliyunOpenUploadName(req, targetPath), req.ConflictPolicy)
	if conflictErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizePan123OpenRequestErrorStatus(conflictErr),
				Message: fmt.Sprintf("123Pan Open upload conflict preflight failed: %v", conflictErr),
				Mode:    "open_family_real_upload",
			},
		}
	}

	md5sum := fmt.Sprintf("%x", md5.Sum(content))
	createStatus, createPayload, createErr := postPan123OpenJSON(context.Background(), session, "/upload/v1/oss/file/create", map[string]interface{}{
		"parentFileID": parentID,
		"filename":     resolvedTargetName,
		"etag":         md5sum,
		"size":         req.Size,
		"type":         1,
	})
	if createErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("123Pan Open upload create request failed: %v", createErr),
				Mode:    "open_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	if createStatus == http.StatusUnauthorized || createStatus == http.StatusForbidden {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "123Pan Open rejected the supplied access token while creating an upload.",
				Mode:    "open_family_real_upload",
				Payload: createPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	if createStatus < 200 || createStatus >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("123Pan Open upload create returned HTTP %d.", createStatus),
				Mode:    "open_family_real_upload",
				Payload: createPayload,
			},
			ConflictAction: conflictAction,
		}
	}

	createData, _ := createPayload["data"].(map[string]interface{})
	preuploadID := firstNonEmptyString(createData, "preuploadID", "preUploadID", "preuploadId")
	fileID := firstNonEmptyString(createData, "fileID", "fileId")
	if boolMapValue(createData, "reuse") && fileID != "" {
		verifyEntry, verifyMode, verifyOK := a.verifyPan123OpenUploadedFile(session, parentID, parentPath, resolvedTargetName, fileID)
		payload := map[string]interface{}{
			"createResponse":     createPayload,
			"fileId":             fileID,
			"preuploadID":        preuploadID,
			"resolvedTargetName": resolvedTargetName,
			"md5":                md5sum,
			"verifyMode":         verifyMode,
			"verifyOk":           verifyOK,
		}
		if verifyEntry != nil {
			payload["verifiedEntry"] = verifyEntry
		}
		return UploadResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: joinPan123OpenNotes("123Pan Open upload reused an existing provider-side file.", conflictNote),
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}
	if preuploadID == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: "123Pan Open upload create did not return preuploadID.",
				Mode:    "open_family_real_upload",
				Payload: createPayload,
			},
			ConflictAction: conflictAction,
		}
	}

	uploadURLStatus, uploadURLPayload, uploadURLErr := postPan123OpenJSON(context.Background(), session, "/upload/v1/oss/file/get_upload_url", map[string]interface{}{
		"preuploadID": preuploadID,
		"sliceNo":     1,
	})
	if uploadURLErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("123Pan Open get_upload_url request failed: %v", uploadURLErr),
				Mode:    "open_family_real_upload",
				Payload: createPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	if uploadURLStatus == http.StatusUnauthorized || uploadURLStatus == http.StatusForbidden {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "123Pan Open rejected the supplied access token while fetching an upload URL.",
				Mode:    "open_family_real_upload",
				Payload: uploadURLPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	if uploadURLStatus < 200 || uploadURLStatus >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("123Pan Open get_upload_url returned HTTP %d.", uploadURLStatus),
				Mode:    "open_family_real_upload",
				Payload: uploadURLPayload,
			},
			ConflictAction: conflictAction,
		}
	}

	uploadURLData, _ := uploadURLPayload["data"].(map[string]interface{})
	presignedURL := firstNonEmptyString(uploadURLData, "presignedURL", "presignedUrl", "url")
	if presignedURL == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: "123Pan Open get_upload_url did not return presignedURL.",
				Mode:    "open_family_real_upload",
				Payload: uploadURLPayload,
			},
			ConflictAction: conflictAction,
		}
	}

	putStatus, putHeaders, putErr := putPan123OpenBytes(context.Background(), presignedURL, content)
	if putErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("123Pan Open PUT upload request failed: %v", putErr),
				Mode:    "open_family_real_upload",
				Payload: mergeMaps(createPayload, uploadURLPayload),
			},
			ConflictAction: conflictAction,
		}
	}
	if putStatus < 200 || putStatus >= 300 {
		payload := mergeMaps(createPayload, uploadURLPayload)
		payload["putStatus"] = putStatus
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("123Pan Open PUT upload returned HTTP %d.", putStatus),
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
			ConflictAction: conflictAction,
		}
	}

	completeStatus, completePayload, completeErr := postPan123OpenJSON(context.Background(), session, "/upload/v1/oss/file/upload_complete", map[string]interface{}{
		"preuploadID": preuploadID,
	})
	if completeErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("123Pan Open upload_complete request failed: %v", completeErr),
				Mode:    "open_family_real_upload",
				Payload: mergeMaps(createPayload, uploadURLPayload),
			},
			ConflictAction: conflictAction,
		}
	}
	if completeStatus == http.StatusUnauthorized || completeStatus == http.StatusForbidden {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "123Pan Open rejected the supplied access token while completing an upload.",
				Mode:    "open_family_real_upload",
				Payload: completePayload,
			},
			ConflictAction: conflictAction,
		}
	}
	if completeStatus < 200 || completeStatus >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("123Pan Open upload_complete returned HTTP %d.", completeStatus),
				Mode:    "open_family_real_upload",
				Payload: completePayload,
			},
			ConflictAction: conflictAction,
		}
	}

	asyncPayload := map[string]interface{}{}
	completeData, _ := completePayload["data"].(map[string]interface{})
	if boolMapValue(completeData, "async") && !boolMapValue(completeData, "completed") {
		_, asyncPayload, _ = pollPan123OpenUploadAsyncResult(session, preuploadID, 4)
	}
	asyncData, _ := asyncPayload["data"].(map[string]interface{})
	if resolvedFileID := firstNonEmptyString(asyncData, "fileID", "fileId"); resolvedFileID != "" {
		fileID = resolvedFileID
	} else if resolvedFileID := firstNonEmptyString(completeData, "fileID", "fileId"); resolvedFileID != "" {
		fileID = resolvedFileID
	}
	verifyEntry, verifyMode, verifyOK := a.verifyPan123OpenUploadedFile(session, parentID, parentPath, resolvedTargetName, fileID)

	resultPayload := map[string]interface{}{
		"createResponse":            createPayload,
		"getUploadUrlResponse":      uploadURLPayload,
		"uploadCompleteResponse":    completePayload,
		"uploadAsyncResultResponse": asyncPayload,
		"preuploadID":               preuploadID,
		"fileId":                    fileID,
		"putStatus":                 putStatus,
		"resolvedTargetName":        resolvedTargetName,
		"md5":                       md5sum,
		"verifyMode":                verifyMode,
		"verifyOk":                  verifyOK,
	}
	if etag := strings.TrimSpace(putHeaders.Get("ETag")); etag != "" {
		resultPayload["etag"] = strings.Trim(etag, "\"")
	}
	if verifyEntry != nil {
		resultPayload["verifiedEntry"] = verifyEntry
	}

	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: joinPan123OpenNotes("123Pan Open binary upload completed through create + get_upload_url + single-part PUT + upload_complete.", conflictNote),
			Mode:    "open_family_real_upload",
			Payload: resultPayload,
		},
		ConflictAction: conflictAction,
	}
}

func (a OpenFamilyAdapter) newPan123OpenSession(profile AuthProfile) (pan123OpenSession, error) {
	baseEndpoint, err := resolvePan123OpenEndpoint(profile)
	if err != nil {
		return pan123OpenSession{}, err
	}
	authorization := pan123OpenAuthorizationHeader(profile)
	if authorization == "" {
		return pan123OpenSession{}, fmt.Errorf("missing access token")
	}
	return pan123OpenSession{
		BaseEndpoint:  baseEndpoint,
		Authorization: authorization,
		ProfileID:     strings.TrimSpace(profile.ID),
		ProviderKey:   a.MetaInfo.Key,
	}, nil
}

func (a OpenFamilyAdapter) resolvePan123OpenEntryByPath(session pan123OpenSession, path string, pageSize int) (string, map[string]interface{}, bool, error) {
	normalized := normalizeOpenFamilyPath(path)
	if normalized == "/" {
		return "0", map[string]interface{}{
			"exists":   true,
			"isDir":    true,
			"name":     "/",
			"path":     "/",
			"fileId":   "0",
			"provider": session.ProviderKey,
		}, true, nil
	}

	currentID := "0"
	currentPath := "/"
	var currentEntry map[string]interface{}
	parts := strings.Split(strings.Trim(normalized, "/"), "/")
	for _, part := range parts {
		children, err := a.listPan123OpenByParent(session, currentID, currentPath, pageSize)
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

func (a OpenFamilyAdapter) listPan123OpenByParent(session pan123OpenSession, parentFileID string, basePath string, pageSize int) ([]map[string]interface{}, error) {
	limit := pageSize
	if limit <= 0 {
		limit = 100
	}
	lastFileID := "0"
	items := make([]map[string]interface{}, 0)
	for {
		statusCode, payload, err := getPan123OpenJSON(context.Background(), session, "/api/v2/file/list", map[string]string{
			"parentFileId": strings.TrimSpace(parentFileID),
			"limit":        strconv.Itoa(limit),
			"lastFileId":   lastFileID,
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
		data, _ := payload["data"].(map[string]interface{})
		fileList := interfaceSliceValue(data, "fileList")
		if len(fileList) == 0 {
			return items, nil
		}
		lastItemID := lastFileID
		for _, raw := range fileList {
			item, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			entry := a.normalizePan123OpenEntry(item, pathJoin(basePath, firstNonEmptyString(item, "filename", "fileName", "name")))
			items = append(items, entry)
			if currentID := strings.TrimSpace(stringMapValue(entry, "fileId")); currentID != "" {
				lastItemID = currentID
			}
		}
		if len(fileList) < limit || lastItemID == lastFileID {
			return items, nil
		}
		lastFileID = lastItemID
	}
}

func (a OpenFamilyAdapter) normalizePan123OpenEntry(raw map[string]interface{}, path string) map[string]interface{} {
	itemType := int(int64MapValue(raw, "type"))
	isDir := itemType == 1 || boolMapValue(raw, "isDir")
	entry := map[string]interface{}{
		"exists":   true,
		"name":     firstNonEmptyString(raw, "filename", "fileName", "name"),
		"path":     normalizeOpenFamilyPath(path),
		"fileId":   firstNonEmptyString(raw, "fileId", "fileID"),
		"parentId": firstNonEmptyString(raw, "parentFileId", "parentFileID"),
		"isDir":    isDir,
		"size":     int64MapValue(raw, "size"),
		"provider": a.MetaInfo.Key,
	}
	if md5Value := firstNonEmptyString(raw, "etag", "md5"); md5Value != "" {
		entry["md5"] = md5Value
		entry["etag"] = md5Value
	}
	return entry
}

func (a OpenFamilyAdapter) resolvePan123OpenUploadParent(session pan123OpenSession, req UploadRequest) (string, string, error) {
	if parentID := strings.TrimSpace(req.ParentID); parentID != "" {
		parentPath := normalizeOpenFamilyPath(parentDirectory(req.Path))
		if parentPath == "." || parentPath == "" {
			parentPath = "/"
		}
		return parentID, parentPath, nil
	}

	parentPath := normalizeOpenFamilyPath(parentDirectory(req.Path))
	if parentPath == "." || parentPath == "" {
		parentPath = "/"
	}
	if parentPath == "/" {
		return "0", "/", nil
	}
	parentID, _, found, err := a.resolvePan123OpenEntryByPath(session, parentPath, 0)
	if err != nil {
		return "", "", err
	}
	if !found || parentID == "" {
		return "", "", fmt.Errorf("parent_path_not_found")
	}
	return parentID, parentPath, nil
}

func (a OpenFamilyAdapter) resolvePan123OpenUploadName(session pan123OpenSession, parentID string, targetName string, policy ConflictPolicy) (string, string, string, error) {
	items, err := a.listPan123OpenByParent(session, parentID, "/", 200)
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
	stem, suffix := splitPan123OpenName(targetName)
	candidate := targetName
	for existing[candidate] {
		candidate = fmt.Sprintf("%s (%d)%s", stem, index, suffix)
		index++
	}
	if policy == ConflictPolicyAutoRenameNew {
		return candidate, "auto_rename_new", "A same-name file already exists under the target path, so 123Pan Open auto-renamed the new file.", nil
	}
	return candidate, "overwrite_downgraded_to_auto_rename", "The requested overwrite policy was downgraded because the current 123Pan Open upload path does not support verified in-place overwrite.", nil
}

func splitPan123OpenName(name string) (string, string) {
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

func (a OpenFamilyAdapter) verifyPan123OpenUploadedFile(session pan123OpenSession, parentID string, parentPath string, targetName string, fileID string) (map[string]interface{}, string, bool) {
	items, err := a.listPan123OpenByParent(session, parentID, parentPath, 200)
	if err != nil {
		return nil, "verify_unavailable", false
	}
	if strings.TrimSpace(fileID) != "" {
		for _, item := range items {
			if strings.TrimSpace(stringMapValue(item, "fileId")) == strings.TrimSpace(fileID) {
				return item, "metadata_by_file_id", true
			}
		}
	}
	for _, item := range items {
		if strings.TrimSpace(stringMapValue(item, "name")) == strings.TrimSpace(targetName) {
			return item, "list_by_parent_name", true
		}
	}
	return nil, "list_by_parent_name", false
}

func resolvePan123OpenEndpoint(profile AuthProfile) (string, error) {
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
	return pan123OpenBaseEndpoint, nil
}

func pan123OpenAuthorizationHeader(profile AuthProfile) string {
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

func getPan123OpenJSON(ctx context.Context, session pan123OpenSession, requestPath string, params map[string]string) (int, map[string]interface{}, error) {
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
	applyPan123OpenHeaders(req, session, false)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
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

func postPan123OpenJSON(ctx context.Context, session pan123OpenSession, requestPath string, body interface{}) (int, map[string]interface{}, error) {
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
	applyPan123OpenHeaders(req, session, true)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	if len(bodyBytes) == 0 {
		return resp.StatusCode, map[string]interface{}{}, nil
	}
	var payloadMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &payloadMap); err != nil {
		return resp.StatusCode, nil, fmt.Errorf("decode provider json: %w", err)
	}
	return resp.StatusCode, payloadMap, nil
}

func putPan123OpenBytes(ctx context.Context, endpoint string, body []byte) (int, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	req.Header.Set("User-Agent", "CloudPanSync/0.1")
	req.Header.Set("Content-Type", "application/octet-stream")
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
	return resp.StatusCode, resp.Header.Clone(), nil
}

func applyPan123OpenHeaders(req *http.Request, session pan123OpenSession, withJSONContentType bool) {
	req.Header.Set("User-Agent", "CloudPanSync/0.1")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Platform", "open_platform")
	req.Header.Set("Authorization", session.Authorization)
	if withJSONContentType {
		req.Header.Set("Content-Type", "application/json;charset=UTF-8")
	}
}

func pollPan123OpenUploadAsyncResult(session pan123OpenSession, preuploadID string, attempts int) (int, map[string]interface{}, error) {
	if attempts <= 0 {
		attempts = 1
	}
	latestStatus := 0
	latestPayload := map[string]interface{}{}
	for index := 0; index < attempts; index++ {
		statusCode, payload, err := postPan123OpenJSON(context.Background(), session, "/upload/v1/oss/file/upload_async_result", map[string]interface{}{
			"preuploadID": preuploadID,
		})
		if err != nil {
			return latestStatus, latestPayload, err
		}
		latestStatus = statusCode
		latestPayload = payload
		data, _ := payload["data"].(map[string]interface{})
		if boolMapValue(data, "completed") {
			return latestStatus, latestPayload, nil
		}
		if index+1 < attempts {
			time.Sleep(200 * time.Millisecond)
		}
	}
	return latestStatus, latestPayload, nil
}

func joinPan123OpenNotes(primary string, suffix string) string {
	primary = strings.TrimSpace(primary)
	suffix = strings.TrimSpace(suffix)
	if suffix == "" {
		return primary
	}
	if primary == "" {
		return suffix
	}
	return primary + " " + suffix
}

func normalizeAliyunOpenRequestErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "auth_invalid") {
		return "auth_invalid"
	}
	return "provider_request_failed"
}

func normalizePan123OpenRequestErrorStatus(err error) string {
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
