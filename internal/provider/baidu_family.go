package provider

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	baiduXPanHost      = "https://pan.baidu.com"
	baiduXPanFilePath  = "/rest/2.0/xpan/file"
	baiduPCSUploadHost = "https://d.pcs.baidu.com"
	baiduPCSUploadPath = "/rest/2.0/pcs/superfile2"
	baiduUploadPartSeq = "0"
)

type BaiduFamilyAdapter struct {
	StaticAdapter
}

type baiduSession struct {
	XPanEndpoint string
	PCSEndpoint  string
	AccessToken  string
	Cookie       string
	ProviderKey  string
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
	session, err := a.newBaiduSession(profile)
	if err != nil {
		return OperationResult{
			Status:  normalizeBaiduSessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "baidu_family_real_auth",
		}
	}
	statusCode, payload, requestErr := getBaiduJSON(context.Background(), session, "list", map[string]string{
		"dir":        "/",
		"folder":     "0",
		"order":      "name",
		"desc":       "0",
		"limit":      "0,1",
		"web":        "1",
		"clienttype": "0",
	})
	if requestErr != nil {
		return OperationResult{
			Status:  normalizeBaiduRequestErrorStatus(requestErr),
			Message: fmt.Sprintf("Baidu Netdisk auth validation request failed: %v", requestErr),
			Mode:    "baidu_family_real_auth",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "Baidu Netdisk rejected the supplied access credential.",
			Mode:    "baidu_family_real_auth",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Baidu Netdisk auth validation returned HTTP %d.", statusCode),
			Mode:    "baidu_family_real_auth",
			Payload: payload,
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Baidu Netdisk validated the supplied access credential against the live list endpoint.",
		Mode:    "baidu_family_real_auth",
		Payload: payload,
	}
}

func (a BaiduFamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	session, err := a.newBaiduSession(req.Profile)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduSessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "baidu_family_real_directory",
			},
		}
	}
	dirPath := a.resolveBaiduDirectoryPath(req.Path, req.ParentID)
	statusCode, payload, requestErr := getBaiduJSON(context.Background(), session, "list", map[string]string{
		"dir":        dirPath,
		"folder":     "0",
		"order":      "name",
		"desc":       "0",
		"limit":      fmt.Sprintf("0,%d", clampBaiduListLimit(req.PageSize)),
		"web":        "1",
		"clienttype": "0",
	})
	if requestErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduRequestErrorStatus(requestErr),
				Message: fmt.Sprintf("Baidu Netdisk list request failed: %v", requestErr),
				Mode:    "baidu_family_real_directory",
			},
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "Baidu Netdisk rejected the supplied access credential.",
				Mode:    "baidu_family_real_directory",
				Payload: payload,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Baidu Netdisk list returned HTTP %d.", statusCode),
				Mode:    "baidu_family_real_directory",
				Payload: payload,
			},
		}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Baidu Netdisk listed live directory entries.",
			Mode:    "baidu_family_real_directory",
		},
		Items: normalizeBaiduListItems(payload, dirPath),
	}
}

func (a BaiduFamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	session, err := a.newBaiduSession(req.Profile)
	if err != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduSessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "baidu_family_real_directory",
			},
		}
	}

	targetPath := normalizeBaiduDirPath(req.Path)
	fileID := strings.TrimSpace(req.FileID)
	if targetPath == "/" && fileID == "" {
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: "Baidu Netdisk root directory metadata is available.",
				Mode:    "baidu_family_real_directory",
			},
			Entry: map[string]interface{}{
				"exists":   true,
				"isDir":    true,
				"name":     "/",
				"path":     "/",
				"fileId":   "",
				"parentId": "",
				"provider": a.MetaInfo.Key,
			},
		}
	}

	params := map[string]string{
		"dlink":      "0",
		"thumb":      "1",
		"web":        "1",
		"clienttype": "0",
	}
	if fileID != "" {
		params["fsids"] = fmt.Sprintf("[%s]", strings.TrimSpace(fileID))
	} else if targetPath != "" {
		params["path"] = targetPath
	}
	statusCode, payload, requestErr := getBaiduJSON(context.Background(), session, "filemetas", params)
	if requestErr != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduRequestErrorStatus(requestErr),
				Message: fmt.Sprintf("Baidu Netdisk metadata request failed: %v", requestErr),
				Mode:    "baidu_family_real_directory",
			},
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "Baidu Netdisk rejected the supplied access credential.",
				Mode:    "baidu_family_real_directory",
				Payload: payload,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Baidu Netdisk metadata returned HTTP %d.", statusCode),
				Mode:    "baidu_family_real_directory",
				Payload: payload,
			},
		}
	}
	if info := mapSliceValue(payload, "info"); len(info) > 0 {
		entry := a.normalizeBaiduMetadataEntry(info[0], targetPath)
		return MetadataResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "exists",
				Message: "Baidu Netdisk returned live metadata.",
				Mode:    "baidu_family_real_directory",
			},
			Entry: entry,
		}
	}
	if targetPath != "" {
		parentDir := normalizeBaiduDirPath(parentDirectory(targetPath))
		listResult := a.List(ListRequest{Profile: req.Profile, Path: parentDir, ParentID: parentDir, PageSize: 200})
		if listResult.OK {
			for _, item := range listResult.Items {
				if strings.TrimSpace(stringMapValue(item, "path")) == targetPath {
					return MetadataResult{
						OperationResult: OperationResult{
							OK:      true,
							Status:  "exists",
							Message: "Baidu Netdisk returned live metadata through parent listing fallback.",
							Mode:    "baidu_family_real_directory",
						},
						Entry: item,
					}
				}
			}
		}
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "missing",
			Message: "Baidu Netdisk did not find the requested path.",
			Mode:    "baidu_family_real_directory",
		},
		Entry: map[string]interface{}{
			"exists":   false,
			"path":     targetPath,
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a BaiduFamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	session, err := a.newBaiduSession(req.Profile)
	if err != nil {
		return OperationResult{
			Status:  normalizeBaiduSessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "baidu_family_real_directory",
		}
	}
	parentDir := a.resolveBaiduDirectoryPath("", req.ParentID)
	fullPath := baiduJoinPath(parentDir, strings.TrimSpace(req.DirName))
	statusCode, payload, requestErr := postBaiduForm(context.Background(), session, "create", map[string]string{
		"path":       fullPath,
		"isdir":      "1",
		"size":       "0",
		"block_list": "[]",
		"rtype":      "1",
		"web":        "1",
		"clienttype": "0",
	})
	if requestErr != nil {
		return OperationResult{
			Status:  normalizeBaiduRequestErrorStatus(requestErr),
			Message: fmt.Sprintf("Baidu Netdisk create-dir request failed: %v", requestErr),
			Mode:    "baidu_family_real_directory",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "Baidu Netdisk rejected the supplied access credential while creating a directory.",
			Mode:    "baidu_family_real_directory",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("Baidu Netdisk create-dir returned HTTP %d.", statusCode),
			Mode:    "baidu_family_real_directory",
			Payload: payload,
		}
	}
	entry := map[string]interface{}{
		"exists":   true,
		"name":     strings.TrimSpace(req.DirName),
		"path":     fullPath,
		"fileId":   firstNonEmptyString(payload, "fs_id", "fsid"),
		"parentId": parentDir,
		"type":     "dir",
		"isDir":    true,
		"size":     int64(0),
		"provider": a.MetaInfo.Key,
		"raw":      payload,
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Baidu Netdisk created the requested directory.",
		Mode:    "baidu_family_real_directory",
		Payload: entry,
	}
}

func (a BaiduFamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.MD5) != "" && req.Size > 0
	message := "Baidu Netdisk fast-upload requires md5 and size."
	if candidate {
		message = "Baidu Netdisk fast-upload candidate is available."
	}
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "baidu_family_real_upload",
			Payload: map[string]interface{}{
				"requires": []string{"md5", "size"},
			},
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
				Message: "Baidu Netdisk pending_manual items still require manual confirmation.",
				Mode:    "baidu_family_real_upload",
			},
		}
	}
	session, err := a.newBaiduSession(req.Profile)
	if err != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduSessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "baidu_family_real_upload",
			},
		}
	}

	localPath := strings.TrimSpace(req.LocalPath)
	if localPath == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: "Baidu Netdisk upload requires a readable local file.",
				Mode:    "baidu_family_real_upload",
			},
		}
	}
	info, statErr := os.Stat(localPath)
	if statErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("Baidu Netdisk could not stat local file: %v", statErr),
				Mode:    "baidu_family_real_upload",
			},
		}
	}
	localMD5 := strings.ToLower(strings.TrimSpace(req.MD5))
	if localMD5 == "" {
		computed, computeErr := computeBaiduLocalMD5(localPath)
		if computeErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "local_file_missing",
					Message: fmt.Sprintf("Baidu Netdisk could not compute local md5: %v", computeErr),
					Mode:    "baidu_family_real_upload",
				},
			}
		}
		localMD5 = computed
	}
	if localMD5 == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_md5",
				Message: "Baidu Netdisk upload requires md5 or a readable local file to compute it.",
				Mode:    "baidu_family_real_upload",
			},
		}
	}

	parentDir := a.resolveBaiduDirectoryPath(req.Path, req.ParentID)
	targetName, conflictAction, conflictNote, conflictErr := a.resolveBaiduUploadName(req.Profile, parentDir, inferName(req.Path, req.Name), req.ConflictPolicy)
	if conflictErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduRequestErrorStatus(conflictErr),
				Message: fmt.Sprintf("Baidu Netdisk upload conflict preflight failed: %v", conflictErr),
				Mode:    "baidu_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	remotePath := baiduJoinPath(parentDir, targetName)

	_, precreatePayload, precreateErr := postBaiduForm(context.Background(), session, "precreate", map[string]string{
		"path":        remotePath,
		"size":        strconv.FormatInt(info.Size(), 10),
		"isdir":       "0",
		"autoinit":    "1",
		"rtype":       "1",
		"block_list":  fmt.Sprintf("[\"%s\"]", localMD5),
		"content-md5": localMD5,
		"slice-md5":   localMD5,
		"web":         "1",
		"clienttype":  "0",
	})
	if precreateErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduRequestErrorStatus(precreateErr),
				Message: fmt.Sprintf("Baidu Netdisk precreate request failed: %v", precreateErr),
				Mode:    "baidu_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	uploadID := firstNonEmptyString(precreatePayload, "uploadid")
	commonPayload := map[string]interface{}{
		"precreateResponse":  precreatePayload,
		"uploadId":           uploadID,
		"resolvedTargetName": targetName,
		"remotePath":         remotePath,
		"conflictAction":     conflictAction,
		"md5":                localMD5,
	}
	if uploadID == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_uploadid",
				Message: "Baidu Netdisk precreate succeeded but did not return uploadid.",
				Mode:    "baidu_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	baiduApplyWholeObjectProgress(commonPayload, "", localMD5, info.Size(), false)

	tmpfileStatus, tmpfilePayload, tmpfileErr := postBaiduTmpfile(context.Background(), session, remotePath, uploadID, localPath)
	commonPayload["tmpfileResponse"] = tmpfilePayload
	if tmpfileErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduRequestErrorStatus(tmpfileErr),
				Message: fmt.Sprintf("Baidu Netdisk tmpfile upload failed: %v", tmpfileErr),
				Mode:    "baidu_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	if tmpfileStatus < 200 || tmpfileStatus >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Baidu Netdisk tmpfile upload returned HTTP %d.", tmpfileStatus),
				Mode:    "baidu_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	baiduApplyWholeObjectProgress(commonPayload, "", localMD5, info.Size(), true)

	createStatus, createPayload, createErr := postBaiduForm(context.Background(), session, "create", map[string]string{
		"path":       remotePath,
		"size":       strconv.FormatInt(info.Size(), 10),
		"isdir":      "0",
		"rtype":      "1",
		"uploadid":   uploadID,
		"block_list": fmt.Sprintf("[\"%s\"]", localMD5),
	})
	commonPayload["createResponse"] = createPayload
	fileID := firstNonEmptyString(createPayload, "fs_id", "fsid")
	commonPayload["fileId"] = fileID
	baiduApplyWholeObjectProgress(commonPayload, fileID, localMD5, info.Size(), true)
	if createErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeBaiduRequestErrorStatus(createErr),
				Message: fmt.Sprintf("Baidu Netdisk create request failed: %v", createErr),
				Mode:    "baidu_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	if createStatus < 200 || createStatus >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("Baidu Netdisk create returned HTTP %d.", createStatus),
				Mode:    "baidu_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}

	verifyEntry, verifyMode, verifyOK := a.verifyBaiduUploadedFile(req.Profile, parentDir, remotePath, fileID)
	if verifyEntry != nil {
		commonPayload["verifyEntry"] = verifyEntry
	}
	commonPayload["verifyMode"] = verifyMode
	commonPayload["verifyOk"] = verifyOK
	commonPayload["usedBinaryFallback"] = false
	message := "Baidu Netdisk binary upload completed through precreate + superfile2 tmpfile + create."
	if conflictNote != "" {
		message += " " + conflictNote
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "baidu_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: conflictAction,
	}
}

func baiduApplyWholeObjectProgress(payload map[string]interface{}, fileID string, md5 string, size int64, completed bool) {
	payload["partCount"] = 1
	if strings.TrimSpace(fileID) != "" {
		payload["fileId"] = fileID
	}
	if completed {
		payload["uploadedPartCount"] = 1
		payload["uploadedParts"] = []map[string]interface{}{
			{
				"partNumber": 1,
				"md5":        strings.ToLower(strings.TrimSpace(md5)),
				"size":       size,
			},
		}
		payload["failedPartNumber"] = 0
		payload["nextPartNumber"] = 2
		return
	}
	payload["uploadedPartCount"] = 0
	payload["failedPartNumber"] = 1
	payload["nextPartNumber"] = 1
	delete(payload, "uploadedParts")
}

func (a BaiduFamilyAdapter) newBaiduSession(profile AuthProfile) (baiduSession, error) {
	accessToken := normalizeBaiduAccessToken(profile.Token)
	if accessToken == "" {
		accessToken = normalizeBaiduAccessToken(firstNonEmptyExtra(profile.Extra, "authorization", "access_token", "accessToken"))
	}
	cookie := strings.TrimSpace(profile.Cookie)
	if cookie == "" {
		cookie = firstNonEmptyExtra(profile.Extra, "cookie", "cookie_header")
	}
	if accessToken == "" && cookie == "" {
		return baiduSession{}, fmt.Errorf("Baidu adapter requires a token or cookie")
	}
	xpanEndpoint, err := resolveBaiduEndpoint(profile, "apiEndpoint", baiduXPanHost)
	if err != nil {
		return baiduSession{}, err
	}
	pcsEndpoint, err := resolveBaiduEndpoint(profile, "pcsEndpoint", baiduPCSUploadHost)
	if err != nil {
		return baiduSession{}, err
	}
	return baiduSession{
		XPanEndpoint: xpanEndpoint,
		PCSEndpoint:  pcsEndpoint,
		AccessToken:  accessToken,
		Cookie:       cookie,
		ProviderKey:  a.MetaInfo.Key,
	}, nil
}

func normalizeBaiduAccessToken(value string) string {
	token := strings.TrimSpace(value)
	if token == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(token), "bearer ") {
		return strings.TrimSpace(token[7:])
	}
	return token
}

func resolveBaiduEndpoint(profile AuthProfile, key string, fallback string) (string, error) {
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

func clampBaiduListLimit(pageSize int) int {
	if pageSize <= 0 {
		return 100
	}
	if pageSize > 200 {
		return 200
	}
	return pageSize
}

func (a BaiduFamilyAdapter) resolveBaiduDirectoryPath(path string, parentID string) string {
	if strings.TrimSpace(parentID) != "" {
		return normalizeBaiduDirPath(parentID)
	}
	if strings.TrimSpace(path) == "" || normalizeBaiduDirPath(path) == "/" {
		return "/"
	}
	return normalizeBaiduDirPath(path)
}

func normalizeBaiduDirPath(path string) string {
	text := strings.TrimSpace(path)
	if text == "" || text == "/" {
		return "/"
	}
	if !strings.HasPrefix(text, "/") {
		text = "/" + text
	}
	return strings.TrimRight(text, "/")
}

func baiduJoinPath(parent string, name string) string {
	parentDir := normalizeBaiduDirPath(parent)
	child := strings.Trim(strings.TrimSpace(name), "/")
	if parentDir == "/" {
		if child == "" {
			return "/"
		}
		return "/" + child
	}
	if child == "" {
		return parentDir
	}
	return parentDir + "/" + child
}

func getBaiduJSON(ctx context.Context, session baiduSession, method string, params map[string]string) (int, map[string]interface{}, error) {
	query := url.Values{}
	query.Set("method", method)
	if strings.TrimSpace(session.AccessToken) != "" {
		query.Set("access_token", session.AccessToken)
	}
	for key, value := range params {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(session.XPanEndpoint, "/")+baiduXPanFilePath+"?"+query.Encode(), nil)
	if err != nil {
		return 0, nil, err
	}
	applyBaiduHeaders(req, session.Cookie, "")
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeBaiduJSON(resp)
}

func postBaiduForm(ctx context.Context, session baiduSession, method string, body map[string]string) (int, map[string]interface{}, error) {
	query := url.Values{}
	query.Set("method", method)
	if strings.TrimSpace(session.AccessToken) != "" {
		query.Set("access_token", session.AccessToken)
	}
	form := url.Values{}
	for key, value := range body {
		if value == "" {
			continue
		}
		form.Set(key, value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(session.XPanEndpoint, "/")+baiduXPanFilePath+"?"+query.Encode(), strings.NewReader(form.Encode()))
	if err != nil {
		return 0, nil, err
	}
	applyBaiduHeaders(req, session.Cookie, "application/x-www-form-urlencoded;charset=UTF-8")
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeBaiduJSON(resp)
}

func postBaiduTmpfile(ctx context.Context, session baiduSession, remotePath string, uploadID string, localPath string) (int, map[string]interface{}, error) {
	fileBytes, err := os.ReadFile(localPath)
	if err != nil {
		return 0, nil, err
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "blob")
	if err != nil {
		return 0, nil, err
	}
	if _, err := part.Write(fileBytes); err != nil {
		return 0, nil, err
	}
	if err := writer.Close(); err != nil {
		return 0, nil, err
	}
	query := url.Values{}
	query.Set("method", "upload")
	query.Set("type", "tmpfile")
	query.Set("path", remotePath)
	query.Set("partseq", baiduUploadPartSeq)
	query.Set("uploadid", uploadID)
	if strings.TrimSpace(session.AccessToken) != "" {
		query.Set("access_token", session.AccessToken)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(session.PCSEndpoint, "/")+baiduPCSUploadPath+"?"+query.Encode(), &body)
	if err != nil {
		return 0, nil, err
	}
	applyBaiduHeaders(req, session.Cookie, writer.FormDataContentType())
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodeBaiduJSON(resp)
}

func decodeBaiduJSON(resp *http.Response) (int, map[string]interface{}, error) {
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

func applyBaiduHeaders(req *http.Request, cookie string, contentType string) {
	req.Header.Set("User-Agent", "CloudPanSync/0.1")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Referer", "https://pan.baidu.com/")
	req.Header.Set("Origin", "https://pan.baidu.com")
	if strings.TrimSpace(cookie) != "" {
		req.Header.Set("Cookie", cookie)
	}
	if strings.TrimSpace(contentType) != "" {
		req.Header.Set("Content-Type", contentType)
	}
}

func normalizeBaiduListItems(payload map[string]interface{}, parentPath string) []map[string]interface{} {
	rawItems := interfaceSliceValue(payload, "list")
	items := make([]map[string]interface{}, 0, len(rawItems))
	for _, raw := range rawItems {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		path := firstNonEmptyString(item, "path")
		name := firstNonEmptyString(item, "server_filename", "filename", "name")
		if path == "" {
			path = baiduJoinPath(parentPath, name)
		}
		md5Value := strings.ToLower(firstNonEmptyString(item, "md5"))
		isDir := int64MapValue(item, "isdir") == 1 || boolMapValue(item, "isDir")
		entry := map[string]interface{}{
			"exists":   true,
			"fileId":   firstNonEmptyString(item, "fs_id", "fsid"),
			"parentId": normalizeBaiduDirPath(parentPath),
			"name":     name,
			"path":     path,
			"type":     "file",
			"isDir":    isDir,
			"size":     int64MapValue(item, "size"),
			"md5":      "",
			"etag":     "",
			"gcid":     "",
			"provider": "baidu_netdisk",
			"raw":      item,
		}
		if isDir {
			entry["type"] = "dir"
		}
		if len(md5Value) == 32 {
			entry["md5"] = md5Value
			entry["etag"] = md5Value
		}
		items = append(items, entry)
	}
	return items
}

func (a BaiduFamilyAdapter) normalizeBaiduMetadataEntry(item map[string]interface{}, fallbackPath string) map[string]interface{} {
	path := firstNonEmptyString(item, "path")
	if path == "" {
		path = fallbackPath
	}
	md5Value := strings.ToLower(firstNonEmptyString(item, "md5"))
	isDir := int64MapValue(item, "isdir") == 1 || boolMapValue(item, "isDir")
	entry := map[string]interface{}{
		"exists":   true,
		"fileId":   firstNonEmptyString(item, "fs_id", "fsid"),
		"parentId": normalizeBaiduDirPath(parentDirectory(path)),
		"name":     inferName(path, firstNonEmptyString(item, "server_filename", "filename")),
		"path":     path,
		"type":     "file",
		"isDir":    isDir,
		"size":     int64MapValue(item, "size"),
		"md5":      "",
		"etag":     "",
		"gcid":     "",
		"provider": a.MetaInfo.Key,
		"raw":      item,
	}
	if isDir {
		entry["type"] = "dir"
	}
	if len(md5Value) == 32 {
		entry["md5"] = md5Value
		entry["etag"] = md5Value
	}
	return entry
}

func computeBaiduLocalMD5(localPath string) (string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	hasher := md5.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

func (a BaiduFamilyAdapter) resolveBaiduUploadName(profile AuthProfile, parentDir string, targetName string, policy ConflictPolicy) (string, string, string, error) {
	listResult := a.List(ListRequest{
		Profile:  profile,
		Path:     parentDir,
		ParentID: parentDir,
		PageSize: 200,
	})
	if !listResult.OK {
		return targetName, "conflict_check_unavailable", "Could not verify same-name conflicts before Baidu Netdisk upload, so the original file name was kept.", fmt.Errorf("%s", listResult.Status)
	}
	existing := make(map[string]bool, len(listResult.Items))
	for _, item := range listResult.Items {
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
		return candidate, "auto_rename_new", "A same-name file already exists under the target path, so Baidu Netdisk upload auto-renamed the new file.", nil
	}
	return candidate, "overwrite_downgraded_to_auto_rename", "The requested overwrite policy was downgraded because the current Baidu Netdisk upload path does not support verified in-place overwrite.", nil
}

func (a BaiduFamilyAdapter) verifyBaiduUploadedFile(profile AuthProfile, parentDir string, remotePath string, fileID string) (map[string]interface{}, string, bool) {
	metadata := a.Metadata(MetadataRequest{
		Profile: profile,
		Path:    remotePath,
		FileID:  fileID,
	})
	if metadata.OK && metadata.Status == "exists" {
		return metadata.Entry, "metadata_by_file_id", true
	}
	listResult := a.List(ListRequest{
		Profile:  profile,
		Path:     parentDir,
		ParentID: parentDir,
		PageSize: 200,
	})
	if listResult.OK {
		for _, item := range listResult.Items {
			if strings.TrimSpace(stringMapValue(item, "path")) == remotePath {
				return item, "list_by_parent_name", true
			}
		}
		return nil, "list_by_parent_name", false
	}
	return nil, "verify_unavailable", false
}

func normalizeBaiduSessionErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "token or cookie") {
		return "missing_access_token_or_cookie"
	}
	if strings.Contains(err.Error(), "invalid apiEndpoint") || strings.Contains(err.Error(), "invalid pcsEndpoint") {
		return "invalid_provider_endpoint"
	}
	return "provider_request_failed"
}

func normalizeBaiduRequestErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "auth_invalid") {
		return "auth_invalid"
	}
	return "provider_request_failed"
}
