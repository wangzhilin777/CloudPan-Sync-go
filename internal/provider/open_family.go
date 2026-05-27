package provider

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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

var aliyunOpenDefaultPartSize int64 = 16 * 1024 * 1024

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
		return a.validateAliyunOpenAuth(profile, domainID, driveID)
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

	statusCode, payload, createErr := postProviderJSON(context.Background(), session.BaseEndpoint+"/adrive/v1.0/openFile/create", session.Token, createBody)
	if createErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Aliyun Open upload create request failed: %v", createErr),
				Mode:    "open_family_real_upload",
			},
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "Aliyun Open rejected the supplied access token while creating an upload.",
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Aliyun Open upload create returned HTTP %d.", statusCode),
				Mode:    "open_family_real_upload",
				Payload: payload,
			},
		}
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
	partPayloads, putHeaders, uploadPartsErr := a.uploadAliyunOpenParts(partItems, content)
	if uploadPartsErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Aliyun Open upload part request failed: %v", uploadPartsErr),
				Mode:    "open_family_real_upload",
				Payload: mergePayloads(payload, partPayloads),
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
	resultPayload["rapidUpload"] = false
	resultPayload["partCount"] = len(partItems)
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

func (a OpenFamilyAdapter) uploadAliyunOpenParts(partItems []map[string]interface{}, content []byte) (map[string]interface{}, http.Header, error) {
	headers := http.Header{}
	partPayloads := map[string]interface{}{
		"upload.partCount": len(partItems),
	}
	if len(partItems) == 0 {
		return partPayloads, headers, nil
	}
	ranges := splitAliyunOpenContentRanges(len(content), len(partItems))
	for idx, item := range partItems {
		uploadURL := firstNonEmptyString(item, "upload_url", "internal_upload_url")
		if uploadURL == "" {
			return partPayloads, headers, fmt.Errorf("part %d missing upload_url", idx+1)
		}
		start := ranges[idx][0]
		end := ranges[idx][1]
		chunk := content[start:end]
		putStatus, putHeaders, putErr := putProviderBytes(context.Background(), uploadURL, chunk, map[string]string{
			"Content-Length": strconv.FormatInt(int64(len(chunk)), 10),
		})
		if putErr != nil {
			return partPayloads, headers, putErr
		}
		if putStatus < 200 || putStatus >= 300 {
			partPayloads[fmt.Sprintf("upload.part.%d.http_status", idx+1)] = putStatus
			return partPayloads, putHeaders, fmt.Errorf("part %d returned HTTP %d", idx+1, putStatus)
		}
		if etag := strings.TrimSpace(putHeaders.Get("ETag")); etag != "" {
			partPayloads[fmt.Sprintf("upload.part.%d.etag", idx+1)] = strings.Trim(etag, "\"")
		}
		if idx == len(partItems)-1 {
			headers = putHeaders
		}
	}
	return partPayloads, headers, nil
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

func splitAliyunOpenContentRanges(totalSize int, partCount int) [][2]int {
	if partCount <= 1 || totalSize <= 0 {
		return [][2]int{{0, totalSize}}
	}
	ranges := make([][2]int, 0, partCount)
	base := totalSize / partCount
	remainder := totalSize % partCount
	offset := 0
	for idx := 0; idx < partCount; idx++ {
		chunkSize := base
		if idx < remainder {
			chunkSize++
		}
		next := offset + chunkSize
		ranges = append(ranges, [2]int{offset, next})
		offset = next
	}
	if len(ranges) == 0 {
		return [][2]int{{0, totalSize}}
	}
	return ranges
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

func normalizeAliyunOpenRequestErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "auth_invalid") {
		return "auth_invalid"
	}
	return "provider_request_failed"
}
