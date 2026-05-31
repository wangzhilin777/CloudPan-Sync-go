package provider

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
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
	pan115ListURL           = "https://webapi.115.com/files"
	pan115InfoURL           = "https://webapi.115.com/files/get_info"
	pan115MkdirURL          = "https://webapi.115.com/files/add"
	pan115UploadInitURL     = "https://proapi.115.com/open/upload/init"
	pan115UploadGetTokenURL = "https://proapi.115.com/open/upload/get_token"
)

type Pan115FamilyAdapter struct {
	StaticAdapter
}

type pan115Session struct {
	ListEndpoint           string
	InfoEndpoint           string
	MkdirEndpoint          string
	UploadInitEndpoint     string
	UploadGetTokenEndpoint string
	AccessToken            string
	Cookie                 string
	ProviderKey            string
}

var pan115OSSUploader = uploadPan115OSSBinary

func NewPan115FamilyAdapter(meta Provider, capability CapabilitySet) Adapter {
	return Pan115FamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
	}
}

func (a Pan115FamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	session, err := a.newPan115Session(profile)
	if err != nil {
		return OperationResult{
			Status:  normalizePan115SessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "pan115_family_real_auth",
		}
	}
	statusCode, payload, requestErr := getPan115JSON(context.Background(), session.ListEndpoint, map[string]string{
		"aid":      "1",
		"cid":      "0",
		"offset":   "0",
		"limit":    "1",
		"show_dir": "1",
		"format":   "json",
		"natsort":  "1",
	}, session)
	if requestErr != nil {
		return OperationResult{
			Status:  normalizePan115RequestErrorStatus(requestErr),
			Message: fmt.Sprintf("115 Open auth validation request failed: %v", requestErr),
			Mode:    "pan115_family_real_auth",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "115 Open rejected the supplied access credential.",
			Mode:    "pan115_family_real_auth",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("115 Open auth validation returned HTTP %d.", statusCode),
			Mode:    "pan115_family_real_auth",
			Payload: payload,
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "115 Open validated the supplied access credential against the live list endpoint.",
		Mode:    "pan115_family_real_auth",
		Payload: payload,
	}
}

func (a Pan115FamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	session, err := a.newPan115Session(req.Profile)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizePan115SessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "pan115_family_real_directory",
			},
		}
	}
	cid := strings.TrimSpace(req.ParentID)
	if cid == "" {
		cid = "0"
	}
	statusCode, payload, requestErr := getPan115JSON(context.Background(), session.ListEndpoint, map[string]string{
		"aid":      "1",
		"cid":      cid,
		"offset":   "0",
		"limit":    strconv.Itoa(clampPan115Limit(req.PageSize)),
		"show_dir": "1",
		"format":   "json",
		"natsort":  "1",
	}, session)
	if requestErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizePan115RequestErrorStatus(requestErr),
				Message: fmt.Sprintf("115 Open list request failed: %v", requestErr),
				Mode:    "pan115_family_real_directory",
			},
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "115 Open rejected the supplied access credential.",
				Mode:    "pan115_family_real_directory",
				Payload: payload,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("115 Open list returned HTTP %d.", statusCode),
				Mode:    "pan115_family_real_directory",
				Payload: payload,
			},
		}
	}
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "115 Open listed live directory entries.",
			Mode:    "pan115_family_real_directory",
		},
		Items: normalizePan115ListItems(payload),
	}
}

func (a Pan115FamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return MetadataResult{OperationResult: validation}
	}
	session, err := a.newPan115Session(req.Profile)
	if err != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizePan115SessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "pan115_family_real_directory",
			},
		}
	}
	fileID := strings.TrimSpace(req.FileID)
	if fileID == "" {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "missing_file_id",
				Message: "115 Open live metadata currently requires fileId.",
				Mode:    "pan115_family_real_directory",
			},
		}
	}
	statusCode, payload, requestErr := getPan115JSON(context.Background(), session.InfoEndpoint, map[string]string{
		"file_id": fileID,
	}, session)
	if requestErr != nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  normalizePan115RequestErrorStatus(requestErr),
				Message: fmt.Sprintf("115 Open metadata request failed: %v", requestErr),
				Mode:    "pan115_family_real_directory",
			},
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "auth_invalid",
				Message: "115 Open rejected the supplied access credential.",
				Mode:    "pan115_family_real_directory",
				Payload: payload,
			},
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("115 Open metadata returned HTTP %d.", statusCode),
				Mode:    "pan115_family_real_directory",
				Payload: payload,
			},
		}
	}
	data, _ := payload["data"].(map[string]interface{})
	if data == nil {
		data = payload
	}
	entry := normalizePan115MetadataEntry(data)
	if stringMapValue(entry, "path") == "" {
		entry["path"] = defaultPath(req.Path, stringMapValue(entry, "name"))
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "exists",
			Message: "115 Open returned live metadata.",
			Mode:    "pan115_family_real_directory",
		},
		Entry: entry,
	}
}

func (a Pan115FamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	session, err := a.newPan115Session(req.Profile)
	if err != nil {
		return OperationResult{
			Status:  normalizePan115SessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "pan115_family_real_directory",
		}
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = "0"
	}
	statusCode, payload, requestErr := postPan115Form(context.Background(), session.MkdirEndpoint, map[string]string{
		"cname": strings.TrimSpace(req.DirName),
		"pid":   parentID,
	}, session)
	if requestErr != nil {
		return OperationResult{
			Status:  normalizePan115RequestErrorStatus(requestErr),
			Message: fmt.Sprintf("115 Open create-dir request failed: %v", requestErr),
			Mode:    "pan115_family_real_directory",
		}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return OperationResult{
			Status:  "auth_invalid",
			Message: "115 Open rejected the supplied access credential while creating a directory.",
			Mode:    "pan115_family_real_directory",
			Payload: payload,
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("115 Open create-dir returned HTTP %d.", statusCode),
			Mode:    "pan115_family_real_directory",
			Payload: payload,
		}
	}
	data, _ := payload["data"].(map[string]interface{})
	if data == nil {
		data = payload
	}
	entry := map[string]interface{}{
		"exists":   true,
		"fileId":   firstNonEmptyString(data, "cid", "file_id", "id"),
		"parentId": parentID,
		"name":     strings.TrimSpace(req.DirName),
		"path":     strings.TrimSpace(req.DirName),
		"type":     "dir",
		"isDir":    true,
		"size":     int64(0),
		"provider": a.MetaInfo.Key,
		"raw":      payload,
	}
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "115 Open created the requested directory.",
		Mode:    "pan115_family_real_directory",
		Payload: entry,
	}
}

func (a Pan115FamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.SHA1) != "" && req.Size > 0
	message := "115 Open fast-upload requires sha1 and size."
	if candidate {
		message = "115 Open fast-upload candidate is available."
	}
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "pan115_family_real_upload",
			Payload: map[string]interface{}{
				"requires": []string{"sha1", "size"},
			},
		},
		Candidate: candidate,
	}
}

func (a Pan115FamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "115 Open pending_manual items still require manual confirmation.",
				Mode:    "pan115_family_real_upload",
			},
		}
	}
	session, err := a.newPan115Session(req.Profile)
	if err != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizePan115SessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "pan115_family_real_upload",
			},
		}
	}
	localPath := strings.TrimSpace(req.LocalPath)
	if localPath == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: "115 Open upload requires a readable local file.",
				Mode:    "pan115_family_real_upload",
			},
		}
	}
	info, statErr := os.Stat(localPath)
	if statErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("115 Open could not stat local file: %v", statErr),
				Mode:    "pan115_family_real_upload",
			},
		}
	}
	sha1Value := strings.ToUpper(strings.TrimSpace(req.SHA1))
	preid := ""
	if sha1Value == "" {
		computedSHA1, computedPreID, computeErr := computePan115LocalSHA1s(localPath)
		if computeErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "local_file_missing",
					Message: fmt.Sprintf("115 Open could not compute local sha1: %v", computeErr),
					Mode:    "pan115_family_real_upload",
				},
			}
		}
		sha1Value = computedSHA1
		preid = computedPreID
	} else {
		_, computedPreID, computeErr := computePan115LocalSHA1s(localPath)
		if computeErr == nil {
			preid = computedPreID
		}
	}
	if sha1Value == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_sha1",
				Message: "115 Open upload requires sha1 or a readable local file to compute it.",
				Mode:    "pan115_family_real_upload",
			},
		}
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = "0"
	}
	resolvedTargetName := strings.TrimSpace(req.Name)
	if resolvedTargetName == "" {
		resolvedTargetName = inferName(req.Path, "upload.bin")
	}
	if resumed := a.resumePan115BinaryUpload(req, session, parentID, resolvedTargetName, localPath, sha1Value); resumed != nil {
		return *resumed
	}
	baseForm := map[string]string{
		"file_name": resolvedTargetName,
		"file_size": strconv.FormatInt(info.Size(), 10),
		"target":    pan115TargetValue(parentID),
		"fileid":    sha1Value,
		"preid":     preid,
		"topupload": "1",
		"pick_code": "",
		"sign_key":  "",
		"sign_val":  "",
	}
	_, initPayload, initErr := postPan115Form(context.Background(), session.UploadInitEndpoint, baseForm, session)
	if initErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizePan115RequestErrorStatus(initErr),
				Message: fmt.Sprintf("115 Open upload/init request failed: %v", initErr),
				Mode:    "pan115_family_real_upload",
			},
		}
	}
	initState, initData := unwrapPan115Response(initPayload)
	if !initState {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: "115 Open upload/init reached the API but was rejected.",
				Mode:    "pan115_family_real_upload",
				Payload: initPayload,
			},
		}
	}

	finalPayload := initPayload
	finalData := initData
	secondAttempt := map[string]interface{}{}
	responseStatus := int(int64MapValue(finalData, "status"))
	signCheck := ""
	if responseStatus == 6 || responseStatus == 7 || responseStatus == 8 {
		signKey := firstNonEmptyString(finalData, "sign_key")
		signCheck = firstNonEmptyString(finalData, "sign_check")
		if signKey == "" || signCheck == "" {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "missing_sign_check",
					Message: "115 Open returned a sign-check status without enough follow-up data.",
					Mode:    "pan115_family_real_upload",
					Payload: initPayload,
				},
			}
		}
		signVal, signErr := computePan115SignCheckSHA1(localPath, signCheck)
		if signErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "provider_request_failed",
					Message: fmt.Sprintf("115 Open sign-check follow-up failed: %v", signErr),
					Mode:    "pan115_family_real_upload",
					Payload: initPayload,
				},
			}
		}
		followupForm := cloneStringMap(baseForm)
		followupForm["sign_key"] = signKey
		followupForm["sign_val"] = signVal
		secondAttempt["signKey"] = signKey
		secondAttempt["signCheck"] = signCheck
		secondAttempt["signVal"] = signVal
		_, finalPayload, initErr = postPan115Form(context.Background(), session.UploadInitEndpoint, followupForm, session)
		if initErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  normalizePan115RequestErrorStatus(initErr),
					Message: fmt.Sprintf("115 Open sign-check follow-up request failed: %v", initErr),
					Mode:    "pan115_family_real_upload",
				},
			}
		}
		var ok bool
		ok, finalData = unwrapPan115Response(finalPayload)
		if !ok {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "provider_request_failed",
					Message: "115 Open sign-check follow-up reached the API but was rejected.",
					Mode:    "pan115_family_real_upload",
					Payload: finalPayload,
				},
			}
		}
		responseStatus = int(int64MapValue(finalData, "status"))
	}

	fileID := firstNonEmptyString(finalData, "file_id", "fileId")
	pickCode := firstNonEmptyString(finalData, "pick_code", "pickCode")
	commonPayload := map[string]interface{}{
		"initResponse":       initPayload,
		"followupResponse":   finalPayload,
		"secondAttempt":      secondAttempt,
		"fileId":             fileID,
		"pickCode":           pickCode,
		"resolvedTargetName": resolvedTargetName,
		"target":             baseForm["target"],
		"sha1":               sha1Value,
		"preid":              preid,
		"responseStatus":     responseStatus,
	}
	if responseStatus == 2 {
		verifyEntry, verifyMode, verifyOK := a.verifyPan115UploadedFile(req.Profile, parentID, resolvedTargetName, fileID, sha1Value)
		if verifyEntry != nil {
			commonPayload["verifyEntry"] = verifyEntry
		}
		commonPayload["verifyMode"] = verifyMode
		commonPayload["verifyOk"] = verifyOK
		return UploadResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: "115 Open rapid-upload init request succeeded and confirmed a hash-based reuse hit.",
				Mode:    "pan115_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: "none",
		}
	}

	tokenStatus, tokenPayload, tokenErr := getPan115UploadToken(context.Background(), session)
	if tokenErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizePan115RequestErrorStatus(tokenErr),
				Message: fmt.Sprintf("115 Open upload token request failed: %v", tokenErr),
				Mode:    "pan115_family_real_upload",
				Payload: commonPayload,
			},
		}
	}
	tokenState, tokenData := unwrapPan115Response(tokenPayload)
	if !tokenState {
		commonPayload["uploadTokenResponse"] = tokenPayload
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_upload_token",
				Message: "115 Open upload/init reached the live API, but OSS upload token retrieval did not succeed.",
				Mode:    "pan115_family_real_upload",
				Payload: commonPayload,
			},
		}
	}
	uploadSession := extractPan115UploadSession(finalData, tokenData)
	commonPayload["uploadTokenResponse"] = tokenPayload
	commonPayload["uploadSession"] = uploadSession
	commonPayload["providerData"] = map[string]interface{}{
		"uploadSession":      cloneMap(uploadSession),
		"resolvedTargetName": resolvedTargetName,
		"sha1":               sha1Value,
		"parentId":           parentID,
		"fileId":             fileID,
		"uploadId":           pan115OSSUploadID(uploadSession),
	}
	uploadPayload, uploadErr := pan115OSSUploader(localPath, uploadSession)
	commonPayload["binaryUpload"] = uploadPayload
	if uploadErr != nil {
		commonPayload = mergeMaps(commonPayload, pan115WholeObjectCheckpoint(uploadSession, fileID, uploadPayload, false))
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "binary_upload_failed",
				Message: fmt.Sprintf("115 Open OSS binary upload failed: %v", uploadErr),
				Mode:    "pan115_family_real_upload",
				Payload: commonPayload,
			},
		}
	}
	commonPayload = mergeMaps(commonPayload, pan115WholeObjectCheckpoint(uploadSession, fileID, uploadPayload, true))
	verifyEntry, verifyMode, verifyOK := a.verifyPan115UploadedFile(req.Profile, parentID, resolvedTargetName, fileID, sha1Value)
	if verifyEntry != nil {
		commonPayload["verifyEntry"] = verifyEntry
	}
	commonPayload["verifyMode"] = verifyMode
	commonPayload["verifyOk"] = verifyOK
	commonPayload["usedBinaryFallback"] = true
	commonPayload["tokenStatus"] = tokenStatus
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "115 Open upload/init hash miss fell back to OSS binary upload and the uploaded file was verified afterwards.",
			Mode:    "pan115_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: "none",
	}
}

func (a Pan115FamilyAdapter) resumePan115BinaryUpload(req UploadRequest, session pan115Session, parentID string, fallbackTargetName string, localPath string, sha1Value string) *UploadResult {
	resume := req.ResumeUpload
	if resume == nil || strings.TrimSpace(localPath) == "" || len(resume.ProviderData) == 0 {
		return nil
	}
	rawUploadSession, _ := resume.ProviderData["uploadSession"].(map[string]interface{})
	if len(rawUploadSession) == 0 {
		return nil
	}
	uploadSession := cloneMap(rawUploadSession)
	fileID := strings.TrimSpace(resume.FileID)
	if fileID == "" {
		fileID = firstNonEmptyString(uploadSession, "fileId")
	}
	resolvedTargetName := stringMapValue(resume.ProviderData, "resolvedTargetName")
	if resolvedTargetName == "" {
		resolvedTargetName = fallbackTargetName
	}
	if resumedParentID := stringMapValue(resume.ProviderData, "parentId"); resumedParentID != "" {
		parentID = resumedParentID
	}
	if resumedSHA1 := stringMapValue(resume.ProviderData, "sha1"); resumedSHA1 != "" {
		sha1Value = resumedSHA1
	}
	uploadPayload, uploadErr := pan115OSSUploader(localPath, uploadSession)
	commonPayload := map[string]interface{}{
		"fileId":             fileID,
		"uploadId":           firstNonEmptyValue(resume.UploadID, pan115OSSUploadID(uploadSession)),
		"resolvedTargetName": resolvedTargetName,
		"usedBinaryFallback": true,
		"resumedUpload":      true,
		"providerData":       cloneMap(resume.ProviderData),
		"uploadSession":      uploadSession,
		"binaryUpload":       uploadPayload,
	}
	if uploadErr != nil {
		commonPayload = mergeMaps(commonPayload, pan115WholeObjectCheckpoint(uploadSession, fileID, uploadPayload, false))
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  "binary_upload_failed",
				Message: fmt.Sprintf("115 Open resumed OSS binary upload failed: %v", uploadErr),
				Mode:    "pan115_family_real_upload",
				Payload: commonPayload,
			},
		}
	}
	commonPayload = mergeMaps(commonPayload, pan115WholeObjectCheckpoint(uploadSession, fileID, uploadPayload, true))
	verifyEntry, verifyMode, verifyOK := a.verifyPan115UploadedFile(req.Profile, parentID, resolvedTargetName, fileID, sha1Value)
	if verifyEntry != nil {
		commonPayload["verifyEntry"] = verifyEntry
	}
	commonPayload["verifyMode"] = verifyMode
	commonPayload["verifyOk"] = verifyOK
	return &UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "115 Open resumed a cached OSS upload session and verified the uploaded file afterwards.",
			Mode:    "pan115_family_real_upload",
			Payload: commonPayload,
		},
	}
}

func (a Pan115FamilyAdapter) newPan115Session(profile AuthProfile) (pan115Session, error) {
	accessToken := normalizeBaiduAccessToken(profile.Token)
	if accessToken == "" {
		accessToken = normalizeBaiduAccessToken(firstNonEmptyExtra(profile.Extra, "authorization", "Authorization", "accessToken", "access_token"))
	}
	cookie := strings.TrimSpace(profile.Cookie)
	if cookie == "" {
		cookie = firstNonEmptyExtra(profile.Extra, "cookie", "cookie_header")
	}
	if accessToken == "" && cookie == "" {
		return pan115Session{}, fmt.Errorf("115 Open adapter requires a token or cookie")
	}
	listEndpoint, err := resolvePan115Endpoint(profile, "listEndpoint", pan115ListURL)
	if err != nil {
		return pan115Session{}, err
	}
	infoEndpoint, err := resolvePan115Endpoint(profile, "infoEndpoint", pan115InfoURL)
	if err != nil {
		return pan115Session{}, err
	}
	mkdirEndpoint, err := resolvePan115Endpoint(profile, "mkdirEndpoint", pan115MkdirURL)
	if err != nil {
		return pan115Session{}, err
	}
	uploadInitEndpoint, err := resolvePan115Endpoint(profile, "uploadInitEndpoint", pan115UploadInitURL)
	if err != nil {
		return pan115Session{}, err
	}
	uploadGetTokenEndpoint, err := resolvePan115Endpoint(profile, "uploadGetTokenEndpoint", pan115UploadGetTokenURL)
	if err != nil {
		return pan115Session{}, err
	}
	return pan115Session{
		ListEndpoint:           listEndpoint,
		InfoEndpoint:           infoEndpoint,
		MkdirEndpoint:          mkdirEndpoint,
		UploadInitEndpoint:     uploadInitEndpoint,
		UploadGetTokenEndpoint: uploadGetTokenEndpoint,
		AccessToken:            accessToken,
		Cookie:                 cookie,
		ProviderKey:            a.MetaInfo.Key,
	}, nil
}

func resolvePan115Endpoint(profile AuthProfile, key string, fallback string) (string, error) {
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

func clampPan115Limit(pageSize int) int {
	if pageSize <= 0 {
		return 100
	}
	if pageSize > 1150 {
		return 1150
	}
	return pageSize
}

func getPan115JSON(ctx context.Context, endpoint string, params map[string]string, session pan115Session) (int, map[string]interface{}, error) {
	query := url.Values{}
	for key, value := range params {
		if strings.TrimSpace(value) == "" {
			continue
		}
		query.Set(key, value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return 0, nil, err
	}
	applyPan115Headers(req, session)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodePan115JSON(resp)
}

func postPan115Form(ctx context.Context, endpoint string, form map[string]string, session pan115Session) (int, map[string]interface{}, error) {
	values := url.Values{}
	for key, value := range form {
		if value == "" {
			continue
		}
		values.Set(key, value)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return 0, nil, err
	}
	applyPan115Headers(req, session)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodePan115JSON(resp)
}

func decodePan115JSON(resp *http.Response) (int, map[string]interface{}, error) {
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return resp.StatusCode, nil, err
	}
	payload, err := decodeProviderJSONResponse(resp.StatusCode, bodyBytes)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, payload, nil
}

func applyPan115Headers(req *http.Request, session pan115Session) {
	req.Header.Set("User-Agent", "Mozilla/5.0 115Browser/27.0.5.7")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	req.Header.Set("Referer", "https://115.com/")
	if strings.TrimSpace(session.AccessToken) != "" {
		req.Header.Set("Authorization", "Bearer "+session.AccessToken)
	}
	if strings.TrimSpace(session.Cookie) != "" {
		req.Header.Set("Cookie", session.Cookie)
	}
}

func unwrapPan115Response(payload map[string]interface{}) (bool, map[string]interface{}) {
	if _, ok := payload["state"]; ok {
		data, _ := payload["data"].(map[string]interface{})
		if data == nil {
			data = map[string]interface{}{}
		}
		return boolMapValue(payload, "state"), data
	}
	return true, payload
}

func normalizePan115ListItems(payload map[string]interface{}) []map[string]interface{} {
	items := make([]map[string]interface{}, 0)
	for _, raw := range interfaceSliceValue(payload, "data") {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		items = append(items, normalizePan115FileItem(item))
	}
	if len(items) > 0 {
		return items
	}
	for _, raw := range interfaceSliceValue(payload, "files") {
		item, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		items = append(items, normalizePan115FileItem(item))
	}
	return items
}

func normalizePan115FileItem(item map[string]interface{}) map[string]interface{} {
	fileID := firstNonEmptyString(item, "fid", "file_id", "fileId", "cid", "id")
	parentID := firstNonEmptyString(item, "cid", "pid", "parent_id", "parentId")
	name := firstNonEmptyString(item, "n", "file_name", "name")
	isDir := boolMapValue(item, "is_dir") || boolMapValue(item, "isfolder") || boolMapValue(item, "isFolder") || (firstNonEmptyString(item, "cid") != "" && firstNonEmptyString(item, "fid") == "")
	sha1Value := strings.ToUpper(firstNonEmptyString(item, "sha", "sha1"))
	entry := map[string]interface{}{
		"exists":   true,
		"fileId":   fileID,
		"parentId": defaultPath(parentID, "0"),
		"name":     defaultPath(name, fileID),
		"path":     defaultPath(name, fileID),
		"type":     "file",
		"isDir":    isDir,
		"size":     int64MapValue(item, "s"),
		"sha1":     sha1Value,
		"md5":      "",
		"etag":     firstNonEmptyString(item, "pc", "pick_code"),
		"pickcode": firstNonEmptyString(item, "pc", "pick_code"),
		"provider": "115_open",
		"raw":      item,
	}
	if entry["size"].(int64) == 0 {
		entry["size"] = int64MapValue(item, "size")
	}
	if isDir {
		entry["type"] = "dir"
	}
	return entry
}

func normalizePan115MetadataEntry(data map[string]interface{}) map[string]interface{} {
	entry := normalizePan115FileItem(data)
	entry["path"] = defaultPath(firstNonEmptyString(data, "n", "file_name", "name"), stringMapValue(entry, "fileId"))
	return entry
}

func computePan115LocalSHA1s(localPath string) (string, string, error) {
	file, err := os.Open(localPath)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = file.Close() }()
	outer := sha1.New()
	preHasher := sha1.New()
	remainingPreBytes := 128 * 1024
	buffer := make([]byte, 1024*1024)
	for {
		n, readErr := file.Read(buffer)
		if n > 0 {
			chunk := buffer[:n]
			_, _ = outer.Write(chunk)
			if remainingPreBytes > 0 {
				preChunk := chunk
				if len(preChunk) > remainingPreBytes {
					preChunk = preChunk[:remainingPreBytes]
				}
				_, _ = preHasher.Write(preChunk)
				remainingPreBytes -= len(preChunk)
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", "", readErr
		}
	}
	return strings.ToUpper(fmt.Sprintf("%x", outer.Sum(nil))), strings.ToUpper(fmt.Sprintf("%x", preHasher.Sum(nil))), nil
}

func computePan115SignCheckSHA1(localPath string, signCheck string) (string, error) {
	parts := strings.Split(signCheck, "-")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid_sign_check:%s", signCheck)
	}
	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", err
	}
	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}
	if start < 0 || end < start {
		return "", fmt.Errorf("invalid_sign_check:%s", signCheck)
	}
	file, err := os.Open(localPath)
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Seek(int64(start), io.SeekStart); err != nil {
		return "", err
	}
	remaining := end - start + 1
	hasher := sha1.New()
	buffer := make([]byte, 1024*1024)
	for remaining > 0 {
		chunkSize := len(buffer)
		if remaining < chunkSize {
			chunkSize = remaining
		}
		n, readErr := file.Read(buffer[:chunkSize])
		if n > 0 {
			_, _ = hasher.Write(buffer[:n])
			remaining -= n
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}
	if remaining > 0 {
		return "", fmt.Errorf("incomplete_sign_check_range:%s", signCheck)
	}
	return strings.ToUpper(fmt.Sprintf("%x", hasher.Sum(nil))), nil
}

func pan115TargetValue(parentID string) string {
	resolved := strings.TrimSpace(parentID)
	if resolved == "" {
		resolved = "0"
	}
	if strings.HasPrefix(resolved, "U_") {
		return resolved
	}
	return "U_1_" + resolved
}

func getPan115UploadToken(ctx context.Context, session pan115Session) (int, map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, session.UploadGetTokenEndpoint, nil)
	if err != nil {
		return 0, nil, err
	}
	applyPan115Headers(req, session)
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	return decodePan115JSON(resp)
}

func extractPan115UploadSession(data map[string]interface{}, tokenPayload map[string]interface{}) map[string]interface{} {
	tokenData, _ := tokenPayload["data"].(map[string]interface{})
	if tokenData == nil {
		tokenData = tokenPayload
	}
	callbackText, callbackVarText := extractPan115CallbackPayload(data)
	return map[string]interface{}{
		"bucket":          firstNonEmptyString(data, "bucket", "Bucket"),
		"object":          firstNonEmptyString(data, "object", "Object"),
		"callback":        callbackText,
		"callbackVar":     callbackVarText,
		"endpoint":        normalizePan115Endpoint(firstNonEmptyString(tokenData, "endpoint", "Endpoint")),
		"accessKeyId":     firstNonEmptyString(tokenData, "access_key_id", "accessKeyId", "AccessKeyId"),
		"accessKeySecret": firstNonEmptyString(tokenData, "access_key_secret", "accessKeySecret", "AccessKeySecret"),
		"securityToken":   firstNonEmptyString(tokenData, "security_token", "securityToken", "SecurityToken"),
	}
}

func normalizePan115Endpoint(endpoint string) string {
	resolved := strings.TrimSpace(endpoint)
	if resolved == "" {
		return ""
	}
	if strings.HasPrefix(resolved, "http://") || strings.HasPrefix(resolved, "https://") {
		return resolved
	}
	return "https://" + resolved
}

func extractPan115CallbackPayload(data map[string]interface{}) (string, string) {
	callback, _ := data["callback"].(map[string]interface{})
	if callback == nil {
		return "", ""
	}
	value, _ := callback["value"].(map[string]interface{})
	if value == nil {
		value = callback
	}
	return firstNonEmptyString(value, "callback"), firstNonEmptyString(value, "callback_var", "callbackVar")
}

func pan115OSSUploadID(session map[string]interface{}) string {
	if len(session) == 0 {
		return ""
	}
	if value := firstNonEmptyString(session, "uploadId", "upload_id", "id"); value != "" {
		return value
	}
	return firstNonEmptyString(session, "object", "objectKey", "key")
}

func pan115WholeObjectCheckpoint(session map[string]interface{}, fileID string, uploadPayload map[string]interface{}, completed bool) map[string]interface{} {
	out := map[string]interface{}{
		"fileId":            fileID,
		"uploadId":          pan115OSSUploadID(session),
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
				"etag":       strings.Trim(stringMapValue(uploadPayload, "etag"), "\""),
				"size":       int64MapValue(uploadPayload, "objectSize"),
			},
		}
		out["failedPartNumber"] = 0
		out["nextPartNumber"] = 2
	}
	return out
}

func pan115ApplyWholeObjectProgress(payload map[string]interface{}, completed bool) {
	payload["partCount"] = 1
	if completed {
		payload["uploadedPartCount"] = 1
		payload["uploadedParts"] = []map[string]interface{}{
			{
				"partNumber": 1,
				"etag":       strings.Trim(stringMapValue(payload, "etag"), "\""),
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

func uploadPan115OSSBinary(localPath string, session map[string]interface{}) (map[string]interface{}, error) {
	bucketName := firstNonEmptyString(session, "bucket")
	objectKey := firstNonEmptyString(session, "object")
	endpoint := firstNonEmptyString(session, "endpoint")
	accessKeyID := firstNonEmptyString(session, "accessKeyId")
	accessKeySecret := firstNonEmptyString(session, "accessKeySecret")
	securityToken := firstNonEmptyString(session, "securityToken")
	payload := map[string]interface{}{
		"session":   session,
		"localPath": localPath,
		"bucket":    bucketName,
		"object":    objectKey,
		"endpoint":  endpoint,
	}
	if bucketName == "" || objectKey == "" || endpoint == "" || accessKeyID == "" || accessKeySecret == "" || securityToken == "" {
		pan115ApplyWholeObjectProgress(payload, false)
		return payload, fmt.Errorf("115 Open hash-miss response did not include a complete OSS upload session")
	}
	if callbackText := firstNonEmptyString(session, "callback"); callbackText != "" {
		payload["x-oss-callback"] = base64.StdEncoding.EncodeToString([]byte(callbackText))
	}
	if callbackVarText := firstNonEmptyString(session, "callbackVar"); callbackVarText != "" {
		payload["x-oss-callback-var"] = base64.StdEncoding.EncodeToString([]byte(callbackVarText))
	}
	file, err := os.Open(localPath)
	if err != nil {
		return payload, err
	}
	defer func() { _ = file.Close() }()

	info, err := file.Stat()
	if err != nil {
		pan115ApplyWholeObjectProgress(payload, false)
		return payload, err
	}
	payload["objectSize"] = info.Size()
	requestURL, canonicalResource, host, canonicalURI, usePathStyle, err := buildPan115OSSObjectURL(endpoint, bucketName, objectKey)
	if err != nil {
		pan115ApplyWholeObjectProgress(payload, false)
		return payload, err
	}
	dateValue := time.Now().UTC().Format(http.TimeFormat)
	contentType := "application/octet-stream"
	canonicalHeaders := pan115CanonicalOSSHeaders(map[string]string{
		"x-oss-callback":       firstNonEmptyString(payload, "x-oss-callback"),
		"x-oss-callback-var":   firstNonEmptyString(payload, "x-oss-callback-var"),
		"x-oss-security-token": securityToken,
	})
	stringToSign := strings.Join([]string{
		http.MethodPut,
		"",
		contentType,
		dateValue,
		canonicalHeaders + canonicalResource,
	}, "\n")
	signature := base64.StdEncoding.EncodeToString(pan115HMACSHA1([]byte(accessKeySecret), stringToSign))
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, requestURL, file)
	if err != nil {
		return payload, fmt.Errorf("build 115 Open OSS upload request: %w", err)
	}
	req.Header.Set("Authorization", "OSS "+accessKeyID+":"+signature)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Date", dateValue)
	req.Header.Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	req.Header.Set("Host", host)
	req.Header.Set("x-oss-security-token", securityToken)
	if callbackHeader := firstNonEmptyString(payload, "x-oss-callback"); callbackHeader != "" {
		req.Header.Set("x-oss-callback", callbackHeader)
	}
	if callbackVarHeader := firstNonEmptyString(payload, "x-oss-callback-var"); callbackVarHeader != "" {
		req.Header.Set("x-oss-callback-var", callbackVarHeader)
	}
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		pan115ApplyWholeObjectProgress(payload, false)
		return payload, fmt.Errorf("put 115 Open oss object: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	payload["requestURL"] = requestURL
	payload["canonicalResource"] = canonicalResource
	payload["canonicalURI"] = canonicalURI
	payload["usePathStyle"] = usePathStyle
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		payload["responseBody"] = strings.TrimSpace(string(bodyBytes))
		payload["responseStatus"] = resp.StatusCode
		pan115ApplyWholeObjectProgress(payload, false)
		return payload, fmt.Errorf("put 115 Open oss object returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	payload["responseStatus"] = resp.StatusCode
	if etag := strings.TrimSpace(resp.Header.Get("ETag")); etag != "" {
		payload["etag"] = etag
	}
	pan115ApplyWholeObjectProgress(payload, true)
	return payload, nil
}

func buildPan115OSSObjectURL(endpoint string, bucket string, objectKey string) (string, string, string, string, bool, error) {
	resolvedEndpoint := strings.TrimSpace(endpoint)
	if resolvedEndpoint == "" {
		return "", "", "", "", false, fmt.Errorf("115 Open hash-miss response did not include an OSS endpoint")
	}
	if !strings.Contains(resolvedEndpoint, "://") {
		resolvedEndpoint = "https://" + resolvedEndpoint
	}
	parsed, err := url.Parse(resolvedEndpoint)
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("invalid 115 Open OSS endpoint: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", "", "", "", false, fmt.Errorf("invalid 115 Open OSS endpoint: scheme and host are required")
	}
	trimmedObject := strings.TrimLeft(strings.TrimSpace(objectKey), "/")
	if trimmedObject == "" {
		return "", "", "", "", false, fmt.Errorf("115 Open hash-miss response did not include an OSS object key")
	}
	escapedObject := encodePan115OSSObjectKey(trimmedObject)
	endpointHost := trimBucketFromEndpoint(parsed.Host, bucket)
	baseURL := parsed.Scheme + "://" + endpointHost
	usePathStyle := shouldUsePan115OSSPathStyle(parsed.Hostname(), endpointHost)
	baseParsed, err := url.Parse(baseURL)
	if err != nil {
		return "", "", "", "", false, fmt.Errorf("invalid 115 Open OSS base endpoint: %w", err)
	}
	canonicalResource := "/" + bucket + "/" + trimmedObject
	if usePathStyle {
		baseParsed.Path = "/" + url.PathEscape(bucket) + "/" + escapedObject
		return baseParsed.String(), canonicalResource, baseParsed.Host, baseParsed.EscapedPath(), true, nil
	}
	baseParsed.Host = bucket + "." + baseParsed.Host
	baseParsed.Path = "/" + escapedObject
	return baseParsed.String(), canonicalResource, baseParsed.Host, baseParsed.EscapedPath(), false, nil
}

func encodePan115OSSObjectKey(key string) string {
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

func shouldUsePan115OSSPathStyle(hostname string, endpointHost string) bool {
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

func pan115CanonicalOSSHeaders(headers map[string]string) string {
	if len(headers) == 0 {
		return ""
	}
	keys := make([]string, 0, len(headers))
	for key, value := range headers {
		if strings.TrimSpace(value) == "" {
			continue
		}
		keys = append(keys, strings.ToLower(strings.TrimSpace(key)))
	}
	if len(keys) == 0 {
		return ""
	}
	sortStrings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, key+":"+strings.TrimSpace(headers[key]))
	}
	return strings.Join(lines, "\n") + "\n"
}

func sortStrings(values []string) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func pan115HMACSHA1(key []byte, value string) []byte {
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write([]byte(value))
	return mac.Sum(nil)
}

func (a Pan115FamilyAdapter) verifyPan115UploadedFile(profile AuthProfile, parentID string, targetName string, fileID string, expectedSHA1 string) (map[string]interface{}, string, bool) {
	if strings.TrimSpace(fileID) != "" {
		metadata := a.Metadata(MetadataRequest{
			Profile: profile,
			FileID:  fileID,
			Path:    targetName,
		})
		if metadata.OK && metadata.Status == "exists" {
			itemSHA1 := strings.ToUpper(strings.TrimSpace(stringMapValue(metadata.Entry, "sha1")))
			normalized := strings.ToUpper(strings.TrimSpace(expectedSHA1))
			return metadata.Entry, "metadata_by_file_id", normalized == "" || itemSHA1 == "" || itemSHA1 == normalized
		}
	}
	listResult := a.List(ListRequest{
		Profile:  profile,
		ParentID: parentID,
		PageSize: 200,
	})
	if listResult.OK {
		for _, item := range listResult.Items {
			if strings.TrimSpace(stringMapValue(item, "name")) != strings.TrimSpace(targetName) {
				continue
			}
			itemSHA1 := strings.ToUpper(strings.TrimSpace(stringMapValue(item, "sha1")))
			normalized := strings.ToUpper(strings.TrimSpace(expectedSHA1))
			return item, "list_by_parent_name", normalized == "" || itemSHA1 == "" || itemSHA1 == normalized
		}
		return nil, "list_by_parent_name", false
	}
	return nil, "verify_unavailable", false
}

func cloneStringMap(values map[string]string) map[string]string {
	cloned := make(map[string]string, len(values))
	for key, value := range values {
		cloned[key] = value
	}
	return cloned
}

func normalizePan115SessionErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "token or cookie") {
		return "missing_access_token_or_cookie"
	}
	if strings.Contains(err.Error(), "invalid ") {
		return "invalid_provider_endpoint"
	}
	return "provider_request_failed"
}

func normalizePan115RequestErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "auth_invalid") {
		return "auth_invalid"
	}
	return "provider_request_failed"
}
