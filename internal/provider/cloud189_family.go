package provider

import (
	"context"
	"crypto/hmac"
	"crypto/md5"
	"crypto/sha1"
	"encoding/xml"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	cloud189ShareInfoURL    = "https://cloud.189.cn/api/open/share/getShareInfoByCodeV2.action"
	cloud189CheckAccessURL  = "https://cloud.189.cn/api/open/share/checkAccessCode.action"
	cloud189ListShareDirURL = "https://cloud.189.cn/api/open/share/listShareDir.action"
	cloud189CreateFolderURL = "https://cloud.189.cn/api/open/file/createFolder.action"

	cloud189SessionAuthURL  = "https://api.cloud.189.cn/getSessionForPC.action"
	cloud189CreateUploadURL = "https://api.cloud.189.cn/createUploadFile.action"
	cloud189UploadStatusURL = "https://api.cloud.189.cn/getUploadFileStatus.action"

	cloud189AppID        = "8025431004"
	cloud189ClientType   = "TELEPC"
	cloud189ClientVer    = "6.2"
	cloud189ClientChanID = "web_cloud.189.cn"
)

type Cloud189FamilyAdapter struct {
	StaticAdapter
}

type cloud189Session struct {
	ShareInfoEndpoint    string
	CheckAccessEndpoint  string
	ListEndpoint         string
	CreateDirEndpoint    string
	AuthSessionEndpoint  string
	CreateUploadEndpoint string
	UploadStatusEndpoint string
	Cookie               string
	ShareCode            string
	AccessCode           string
	AccessToken          string
	Signature            string
	Date                 string
	PathPrefix           string
	ProviderKey          string
}

type cloud189UploadAuthSession struct {
	SessionKey    string
	SessionSecret string
	StatusCode    int
	RawPayload    map[string]interface{}
}

type cloud189CommitResult struct {
	FileID     string `xml:"id"`
	Name       string `xml:"name"`
	Size       int64  `xml:"size"`
	MD5        string `xml:"md5"`
	CreateDate string `xml:"createDate"`
	RawXML     string `xml:"-"`
}

func NewCloud189FamilyAdapter(meta Provider, capability CapabilitySet) Adapter {
	return Cloud189FamilyAdapter{
		StaticAdapter: StaticAdapter{
			MetaInfo:       meta,
			CapabilityInfo: capability,
		},
	}
}

func (a Cloud189FamilyAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	session, err := a.newCloud189Session(profile)
	if err != nil {
		return OperationResult{
			Status:  normalizeCloud189SessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "cloud189_family_real_auth",
		}
	}
	if strings.TrimSpace(session.ShareCode) != "" {
		statusCode, payload, requestErr := cloud189FetchShareInfo(context.Background(), session)
		if requestErr != nil {
			return OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(requestErr),
				Message: fmt.Sprintf("189Cloud share info request failed: %v", requestErr),
				Mode:    "cloud189_family_real_auth",
			}
		}
		if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
			return OperationResult{
				Status:  "auth_invalid",
				Message: "189Cloud rejected the supplied cookie or share credentials.",
				Mode:    "cloud189_family_real_auth",
				Payload: payload,
			}
		}
		if statusCode < 200 || statusCode >= 300 {
			return OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud share info returned HTTP %d.", statusCode),
				Mode:    "cloud189_family_real_auth",
				Payload: payload,
			}
		}
		ok, code := cloud189ResponseOK(payload)
		if !ok {
			return OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud share info was rejected with code %s.", code),
				Mode:    "cloud189_family_real_auth",
				Payload: payload,
			}
		}
		return OperationResult{
			OK:      true,
			Status:  "verified",
			Message: "189Cloud validated the supplied share credentials against the live share-info endpoint.",
			Mode:    "cloud189_family_real_auth",
			Payload: payload,
		}
	}
	if session.AccessToken != "" {
		return OperationResult{
			OK:      true,
			Status:  "verified",
			Message: "189Cloud account-level upload access token is present and can be used for live createUploadFile requests.",
			Mode:    "cloud189_family_real_auth",
			Payload: map[string]interface{}{
				"authKind":             "account_upload_access_token",
				"providedWriteHeaders": session.Signature != "" && session.Date != "",
			},
		}
	}
	return OperationResult{
		Status:  "missing_share_code",
		Message: "189Cloud adapter currently requires extra.shareCode for live read validation or accessToken for account upload auth.",
		Mode:    "cloud189_family_real_auth",
	}
}

func (a Cloud189FamilyAdapter) List(req ListRequest) ListResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return ListResult{OperationResult: validation}
	}
	session, err := a.newCloud189Session(req.Profile)
	if err != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189SessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "cloud189_family_real_directory",
			},
		}
	}
	if strings.TrimSpace(session.ShareCode) == "" {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "missing_share_code",
				Message: "189Cloud live list currently requires extra.shareCode.",
				Mode:    "cloud189_family_real_directory",
			},
		}
	}
	shareInfoStatus, shareInfoPayload, shareInfoErr := cloud189FetchShareInfo(context.Background(), session)
	if shareInfoErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(shareInfoErr),
				Message: fmt.Sprintf("189Cloud share info request failed: %v", shareInfoErr),
				Mode:    "cloud189_family_real_directory",
			},
		}
	}
	if shareInfoStatus < 200 || shareInfoStatus >= 300 {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud share info returned HTTP %d.", shareInfoStatus),
				Mode:    "cloud189_family_real_directory",
				Payload: shareInfoPayload,
			},
		}
	}
	ok, code := cloud189ResponseOK(shareInfoPayload)
	if !ok {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud share info was rejected with code %s.", code),
				Mode:    "cloud189_family_real_directory",
				Payload: shareInfoPayload,
			},
		}
	}
	shareInfo := cloud189ResponseData(shareInfoPayload)
	shareMode := cloud189ShareMode(shareInfo)
	targetFileID := strings.TrimSpace(req.ParentID)
	if targetFileID == "" {
		targetFileID = firstNonEmptyExtra(req.Profile.Extra, "fileId", "file_id", "rootFileId")
	}
	if targetFileID == "" {
		targetFileID = firstNonEmptyString(shareInfo, "fileId", "id")
	}
	if targetFileID == "" {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "missing_root_file_id",
				Message: "189Cloud live list requires parentId, extra.fileId, or a share root fileId from the provider response.",
				Mode:    "cloud189_family_real_directory",
				Payload: shareInfoPayload,
			},
		}
	}
	_, shareID, shareIDPayload, shareIDErr := cloud189FetchShareID(context.Background(), session, shareInfo)
	if shareIDErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(shareIDErr),
				Message: fmt.Sprintf("189Cloud share-id request failed: %v", shareIDErr),
				Mode:    "cloud189_family_real_directory",
			},
		}
	}
	if strings.TrimSpace(shareID) == "" {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: "189Cloud access-code validation did not return shareId.",
				Mode:    "cloud189_family_real_directory",
				Payload: shareIDPayload,
			},
		}
	}
	listStatus, listPayload, listErr := cloud189FetchDirPage(context.Background(), session, shareID, shareMode, targetFileID, clampCloud189Limit(req.PageSize))
	if listErr != nil {
		return ListResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(listErr),
				Message: fmt.Sprintf("189Cloud live list request failed: %v", listErr),
				Mode:    "cloud189_family_real_directory",
			},
		}
	}
	if listStatus < 200 || listStatus >= 300 {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud live list returned HTTP %d.", listStatus),
				Mode:    "cloud189_family_real_directory",
				Payload: listPayload,
			},
		}
	}
	ok, code = cloud189ResponseOK(listPayload)
	if !ok {
		return ListResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud live list was rejected with code %s.", code),
				Mode:    "cloud189_family_real_directory",
				Payload: listPayload,
			},
		}
	}
	rows := normalizeCloud189ListItems(listPayload, targetFileID, defaultPath(session.PathPrefix, "/189cloud-share"))
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "189Cloud listed live share directory entries.",
			Mode:    "cloud189_family_real_directory",
			Payload: map[string]interface{}{
				"shareCode": session.ShareCode,
				"shareId":   shareID,
				"parentId":  targetFileID,
			},
		},
		Items: rows,
	}
}

func (a Cloud189FamilyAdapter) Metadata(req MetadataRequest) MetadataResult {
	listResult := a.List(ListRequest{
		Profile:  req.Profile,
		ParentID: req.ParentID,
		Path:     req.Path,
		PageSize: 100,
	})
	if !listResult.OK {
		return MetadataResult{OperationResult: listResult.OperationResult}
	}
	targetFileID := strings.TrimSpace(req.FileID)
	var chosen map[string]interface{}
	if targetFileID != "" {
		for _, item := range listResult.Items {
			if strings.TrimSpace(stringMapValue(item, "fileId")) == targetFileID {
				chosen = item
				break
			}
		}
	}
	if chosen == nil {
		for _, item := range listResult.Items {
			if !boolMapValue(item, "isDir") {
				chosen = item
				break
			}
		}
	}
	if chosen == nil {
		return MetadataResult{
			OperationResult: OperationResult{
				Status:  "metadata_not_found",
				Message: "189Cloud metadata probe did not find a file entry in the current directory page.",
				Mode:    "cloud189_family_real_directory",
			},
		}
	}
	entry := map[string]interface{}{
		"exists":   true,
		"fileId":   stringMapValue(chosen, "fileId"),
		"parentId": stringMapValue(chosen, "parentId"),
		"name":     stringMapValue(chosen, "name"),
		"path":     stringMapValue(chosen, "path"),
		"type":     stringMapValue(chosen, "type"),
		"isDir":    boolMapValue(chosen, "isDir"),
		"size":     int64MapValue(chosen, "size"),
		"md5":      stringMapValue(chosen, "md5"),
		"sha1":     "",
		"sha256":   "",
		"gcid":     "",
		"etag":     stringMapValue(chosen, "etag"),
		"provider": a.MetaInfo.Key,
		"raw":      chosen["raw"],
	}
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "exists",
			Message: "189Cloud live metadata succeeded using the current directory page payload.",
			Mode:    "cloud189_family_real_directory",
		},
		Entry: entry,
	}
}

func (a Cloud189FamilyAdapter) CreateDir(req CreateDirRequest) OperationResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return validation
	}
	session, err := a.newCloud189Session(req.Profile)
	if err != nil {
		return OperationResult{
			Status:  normalizeCloud189SessionErrorStatus(err),
			Message: err.Error(),
			Mode:    "cloud189_family_real_directory",
		}
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = firstNonEmptyExtra(req.Profile.Extra, "parentId", "fileId")
	}
	if parentID == "" {
		return OperationResult{
			Status:  "missing_parent_id",
			Message: "189Cloud create folder requires parentId or auth profile extra.parentId/fileId.",
			Mode:    "cloud189_family_real_directory",
			Payload: map[string]interface{}{
				"requiredAuth": []string{"AccessToken", "Signature", "Date"},
			},
		}
	}
	dirName := strings.TrimSpace(req.DirName)
	if dirName == "" {
		return OperationResult{
			Status:  "missing_dir_name",
			Message: "189Cloud create folder requires a non-empty dirName.",
			Mode:    "cloud189_family_real_directory",
			Payload: map[string]interface{}{
				"parentId":     parentID,
				"requiredAuth": []string{"AccessToken", "Signature", "Date"},
			},
		}
	}
	if session.AccessToken == "" || session.Signature == "" || session.Date == "" {
		if session.ShareCode != "" {
			return OperationResult{
				Status:  "share_auth_readonly",
				Message: "189Cloud create folder is not available on the current shareCode/accessCode read-only path; createFolder.action requires account-level OAuth headers.",
				Mode:    "cloud189_family_real_directory",
				Payload: map[string]interface{}{
					"parentId":     parentID,
					"dirName":      dirName,
					"requiredAuth": []string{"AccessToken", "Signature", "Date"},
				},
			}
		}
		return OperationResult{
			Status:  "missing_account_level_auth",
			Message: "189Cloud create folder still needs account-level OAuth auth fields before createFolder.action can be called.",
			Mode:    "cloud189_family_real_directory",
			Payload: map[string]interface{}{
				"parentId":     parentID,
				"dirName":      dirName,
				"requiredAuth": []string{"AccessToken", "Signature", "Date"},
			},
		}
	}
	statusCode, payload, requestErr := cloud189RequestCreateFolder(context.Background(), session, parentID, dirName)
	if requestErr != nil {
		return OperationResult{
			Status:  normalizeCloud189RequestErrorStatus(requestErr),
			Message: fmt.Sprintf("189Cloud create-dir request failed: %v", requestErr),
			Mode:    "cloud189_family_real_directory",
		}
	}
	if statusCode < 200 || statusCode >= 300 {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("189Cloud create-dir returned HTTP %d.", statusCode),
			Mode:    "cloud189_family_real_directory",
			Payload: payload,
		}
	}
	ok, code := cloud189ResponseOK(payload)
	if !ok {
		return OperationResult{
			Status:  "provider_request_failed",
			Message: fmt.Sprintf("189Cloud create-dir was rejected with code %s.", code),
			Mode:    "cloud189_family_real_directory",
			Payload: payload,
		}
	}
	createdID := extractCloud189CreatedFolderID(payload)
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "189Cloud live create folder succeeded with account-level OAuth headers.",
		Mode:    "cloud189_family_real_directory",
		Payload: map[string]interface{}{
			"parentId": parentID,
			"item": map[string]interface{}{
				"fileId":   createdID,
				"parentId": parentID,
				"name":     dirName,
				"path":     dirName,
				"type":     "dir",
				"isDir":    true,
			},
			"raw": payload,
		},
	}
}

func (a Cloud189FamilyAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return FastUploadCheckResult{OperationResult: validation}
	}
	candidate := strings.TrimSpace(req.MD5) != "" && req.Size > 0
	message := "189Cloud fast-upload requires md5 and size."
	if candidate {
		message = "189Cloud fast-upload candidate is available."
	}
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "cloud189_family_real_upload",
			Payload: map[string]interface{}{
				"requires": []string{"md5", "size"},
			},
		},
		Candidate: candidate,
	}
}

func (a Cloud189FamilyAdapter) Upload(req UploadRequest) UploadResult {
	validation := a.ValidateAuth(req.Profile)
	if !validation.OK {
		return UploadResult{OperationResult: validation}
	}
	if req.Strategy == "pending_manual" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "pending_manual_requires_confirmation",
				Message: "189Cloud pending_manual items still require manual confirmation.",
				Mode:    "cloud189_family_real_upload",
			},
		}
	}
	session, err := a.newCloud189Session(req.Profile)
	if err != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189SessionErrorStatus(err),
				Message: err.Error(),
				Mode:    "cloud189_family_real_upload",
			},
		}
	}
	localPath := strings.TrimSpace(req.LocalPath)
	if localPath == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: "189Cloud upload requires a readable local file.",
				Mode:    "cloud189_family_real_upload",
			},
		}
	}
	info, statErr := os.Stat(localPath)
	if statErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("189Cloud could not stat local file: %v", statErr),
				Mode:    "cloud189_family_real_upload",
			},
		}
	}
	if session.AccessToken == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_access_token",
				Message: "189Cloud upload requires token or extra.accessToken.",
				Mode:    "cloud189_family_real_upload",
				Payload: map[string]interface{}{
					"requiredAuth":         []string{"AccessToken"},
					"providedWriteHeaders": session.Signature != "" && session.Date != "",
				},
			},
		}
	}
	actualMD5, hashErr := computeCloud189LocalMD5(localPath)
	if hashErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_file_missing",
				Message: fmt.Sprintf("189Cloud could not compute local md5: %v", hashErr),
				Mode:    "cloud189_family_real_upload",
			},
		}
	}
	if normalizedMD5 := strings.ToLower(strings.TrimSpace(req.MD5)); normalizedMD5 != "" && normalizedMD5 != actualMD5 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "local_hash_mismatch",
				Message: "189Cloud fast upload aborted because local md5 does not match the task entry.",
				Mode:    "cloud189_family_real_upload",
				Payload: map[string]interface{}{
					"actualMd5": actualMD5,
				},
			},
		}
	}
	parentID := strings.TrimSpace(req.ParentID)
	if parentID == "" {
		parentID = firstNonEmptyExtra(req.Profile.Extra, "parentId", "fileId")
	}
	if parentID == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "missing_parent_id",
				Message: "189Cloud upload requires parentId or auth profile extra.parentId/fileId.",
				Mode:    "cloud189_family_real_upload",
				Payload: map[string]interface{}{
					"requiredAuth": []string{"AccessToken"},
				},
			},
		}
	}
	targetName, conflictAction, conflictNote := a.resolveCloud189UploadName(req.Profile, parentID, inferName(req.Path, req.Name), req.ConflictPolicy)
	if resumed := a.resumeCloud189BinaryUpload(req, session, parentID, targetName, localPath, actualMD5, conflictAction); resumed != nil {
		if conflictNote != "" && resumed.OK {
			resumed.Message += " " + conflictNote
		}
		return *resumed
	}

	authSession, authErr := cloud189GetUploadSession(context.Background(), session)
	if authErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(authErr),
				Message: fmt.Sprintf("189Cloud getSessionForPC request failed: %v", authErr),
				Mode:    "cloud189_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	if authSession.SessionKey == "" || authSession.SessionSecret == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: "189Cloud access token refresh did not return sessionKey/sessionSecret for the upload request.",
				Mode:    "cloud189_family_real_upload",
				Payload: map[string]interface{}{
					"sessionResponse": authSession.RawPayload,
				},
			},
			ConflictAction: conflictAction,
		}
	}

	createStatus, createPayload, createErr := cloud189SignedJSON(context.Background(), session, authSession, http.MethodPost, session.CreateUploadEndpoint, nil, map[string]string{
		"parentFolderId": parentID,
		"fileName":       targetName,
		"size":           strconv.FormatInt(info.Size(), 10),
		"md5":            actualMD5,
		"opertype":       "3",
		"flag":           "1",
		"resumePolicy":   "1",
		"isLog":          "0",
	})
	if createErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(createErr),
				Message: fmt.Sprintf("189Cloud createUploadFile request failed: %v", createErr),
				Mode:    "cloud189_family_real_upload",
			},
			ConflictAction: conflictAction,
		}
	}
	uploadFileID := cloud189UploadFileID(createPayload)
	fileUploadURL := firstNonEmptyString(createPayload, "fileUploadUrl")
	commitURL := firstNonEmptyString(createPayload, "fileCommitUrl")
	fileDataExists := int64MapValue(createPayload, "fileDataExists")
	commonPayload := map[string]interface{}{
		"sessionResponse":      authSession.RawPayload,
		"createResponse":       createPayload,
		"uploadId":             strconv.FormatInt(uploadFileID, 10),
		"uploadFileId":         uploadFileID,
		"fileUploadUrl":        fileUploadURL,
		"fileCommitUrl":        commitURL,
		"fileDataExists":       fileDataExists,
		"resolvedTargetName":   targetName,
		"conflictAction":       conflictAction,
		"md5":                  actualMD5,
		"parentId":             parentID,
		"providedWriteHeaders": session.Signature != "" && session.Date != "",
		"usedBinaryFallback":   false,
	}
	if createStatus < 200 || createStatus >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud createUploadFile returned HTTP %d.", createStatus),
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}

	if fileDataExists != 1 {
		if uploadFileID <= 0 || strings.TrimSpace(fileUploadURL) == "" || strings.TrimSpace(commitURL) == "" {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "provider_request_failed",
					Message: "189Cloud createUploadFile reached the live API, but the provider did not return enough binary fallback information.",
					Mode:    "cloud189_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		commonPayload["providerData"] = map[string]interface{}{
			"uploadFileId":       uploadFileID,
			"fileUploadUrl":      fileUploadURL,
			"fileCommitUrl":      commitURL,
			"resolvedTargetName": targetName,
			"parentId":           parentID,
			"md5":                actualMD5,
			"conflictAction":     conflictAction,
		}
		cloud189ApplyWholeObjectProgress(commonPayload, "", nil, info.Size(), false)
		putStatus, putPayload, putErr := cloud189UploadBinaryToURL(context.Background(), localPath, fileUploadURL, uploadFileID)
		commonPayload["binaryUploadResponse"] = putPayload
		commonPayload["uploadPutStatus"] = putStatus
		commonPayload["usedBinaryFallback"] = true
		if putErr != nil {
			cloud189ApplyWholeObjectProgress(commonPayload, "", putPayload, info.Size(), false)
			return UploadResult{
				OperationResult: OperationResult{
					Status:  normalizeCloud189RequestErrorStatus(putErr),
					Message: fmt.Sprintf("189Cloud binary PUT upload failed: %v", putErr),
					Mode:    "cloud189_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		cloud189ApplyWholeObjectProgress(commonPayload, "", putPayload, info.Size(), true)
		_, statusPayload, statusErr := cloud189SignedJSON(context.Background(), session, authSession, http.MethodGet, session.UploadStatusEndpoint, map[string]string{
			"uploadFileId": strconv.FormatInt(uploadFileID, 10),
			"resumePolicy": "1",
		}, nil)
		commonPayload["statusResponse"] = statusPayload
		if statusErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  normalizeCloud189RequestErrorStatus(statusErr),
					Message: fmt.Sprintf("189Cloud getUploadFileStatus request failed: %v", statusErr),
					Mode:    "cloud189_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		statusView := cloud189ExtractStatusView(statusPayload, uploadFileID, fileUploadURL, commitURL)
		commonPayload["statusView"] = statusView
		_, commitText, commitErr := cloud189SignedXML(context.Background(), session, authSession, http.MethodPost, stringMapValue(statusView, "fileCommitUrl"), map[string]string{
			"opertype":     "3",
			"resumePolicy": "1",
			"uploadFileId": strconv.FormatInt(int64MapValue(statusView, "uploadFileId"), 10),
			"isLog":        "0",
		})
		if commitErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  normalizeCloud189RequestErrorStatus(commitErr),
					Message: fmt.Sprintf("189Cloud binary upload commit request failed: %v", commitErr),
					Mode:    "cloud189_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		commitPayload, parseErr := cloud189ParseCommitXML(commitText)
		if parseErr != nil {
			return UploadResult{
				OperationResult: OperationResult{
					Status:  "provider_request_failed",
					Message: fmt.Sprintf("189Cloud binary upload commit XML decode failed: %v", parseErr),
					Mode:    "cloud189_family_real_upload",
					Payload: commonPayload,
				},
				ConflictAction: conflictAction,
			}
		}
		commonPayload["commitResponse"] = map[string]interface{}{
			"fileId":     commitPayload.FileID,
			"name":       commitPayload.Name,
			"size":       commitPayload.Size,
			"md5":        commitPayload.MD5,
			"createDate": commitPayload.CreateDate,
			"rawXml":     commitPayload.RawXML,
		}
		commonPayload["fileId"] = firstNonEmptyString(map[string]interface{}{
			"fileId": commitPayload.FileID,
		}, "fileId")
		cloud189ApplyWholeObjectProgress(commonPayload, commitPayload.FileID, putPayload, info.Size(), true)
		verifyOK := strings.ToLower(strings.TrimSpace(commitPayload.MD5)) == "" || strings.ToLower(strings.TrimSpace(commitPayload.MD5)) == actualMD5
		commonPayload["verifyMode"] = "commit_response_xml_after_binary_put"
		commonPayload["verifyOk"] = verifyOK
		commonPayload["verifyPayload"] = map[string]interface{}{
			"fileId":     commitPayload.FileID,
			"name":       commitPayload.Name,
			"size":       commitPayload.Size,
			"md5":        commitPayload.MD5,
			"createDate": commitPayload.CreateDate,
			"statusView": statusView,
		}
		message := "189Cloud createUploadFile hash miss fell back to binary upload, and the provider commit response confirmed the file."
		if conflictNote != "" {
			message += " " + conflictNote
		}
		return UploadResult{
			OperationResult: OperationResult{
				OK:      true,
				Status:  "ok",
				Message: message,
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}

	if uploadFileID <= 0 || strings.TrimSpace(commitURL) == "" {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: "189Cloud createUploadFile reported a reuse hit, but did not provide enough commit information.",
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	commitStatus, commitText, commitErr := cloud189SignedXML(context.Background(), session, authSession, http.MethodPost, commitURL, map[string]string{
		"opertype":     "3",
		"resumePolicy": "1",
		"uploadFileId": strconv.FormatInt(uploadFileID, 10),
		"isLog":        "0",
	})
	if commitErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(commitErr),
				Message: fmt.Sprintf("189Cloud rapid-upload commit request failed: %v", commitErr),
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	if commitStatus < 200 || commitStatus >= 300 {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud rapid-upload commit returned HTTP %d.", commitStatus),
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	commitPayload, parseErr := cloud189ParseCommitXML(commitText)
	if parseErr != nil {
		return UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud rapid-upload commit XML decode failed: %v", parseErr),
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	commonPayload["commitResponse"] = map[string]interface{}{
		"fileId":     commitPayload.FileID,
		"name":       commitPayload.Name,
		"size":       commitPayload.Size,
		"md5":        commitPayload.MD5,
		"createDate": commitPayload.CreateDate,
		"rawXml":     commitPayload.RawXML,
	}
	commonPayload["fileId"] = commitPayload.FileID
	commonPayload["verifyMode"] = "commit_response_xml"
	commonPayload["verifyOk"] = true
	commonPayload["verifyPayload"] = map[string]interface{}{
		"fileId":     commitPayload.FileID,
		"name":       commitPayload.Name,
		"size":       commitPayload.Size,
		"md5":        commitPayload.MD5,
		"createDate": commitPayload.CreateDate,
	}
	message := "189Cloud rapid-upload request succeeded and was confirmed by the provider commit response."
	if conflictNote != "" {
		message += " " + conflictNote
	}
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: message,
			Mode:    "cloud189_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: conflictAction,
	}
}

func (a Cloud189FamilyAdapter) resumeCloud189BinaryUpload(req UploadRequest, session cloud189Session, parentID string, fallbackTargetName string, localPath string, actualMD5 string, conflictAction string) *UploadResult {
	resume := req.ResumeUpload
	if resume == nil || len(resume.ProviderData) == 0 || strings.TrimSpace(localPath) == "" {
		return nil
	}
	uploadFileID := int64MapValue(resume.ProviderData, "uploadFileId")
	fileUploadURL := stringMapValue(resume.ProviderData, "fileUploadUrl")
	fileCommitURL := stringMapValue(resume.ProviderData, "fileCommitUrl")
	if uploadFileID <= 0 || fileUploadURL == "" || fileCommitURL == "" {
		return nil
	}
	targetName := stringMapValue(resume.ProviderData, "resolvedTargetName")
	if targetName == "" {
		targetName = fallbackTargetName
	}
	if resumedParentID := stringMapValue(resume.ProviderData, "parentId"); resumedParentID != "" {
		parentID = resumedParentID
	}
	if resumedMD5 := stringMapValue(resume.ProviderData, "md5"); resumedMD5 != "" {
		actualMD5 = strings.ToLower(strings.TrimSpace(resumedMD5))
	}
	if resumedConflictAction := stringMapValue(resume.ProviderData, "conflictAction"); resumedConflictAction != "" {
		conflictAction = resumedConflictAction
	}
	authSession, authErr := cloud189GetUploadSession(context.Background(), session)
	if authErr != nil {
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(authErr),
				Message: fmt.Sprintf("189Cloud getSessionForPC request failed during resumed upload: %v", authErr),
				Mode:    "cloud189_family_real_upload",
				Payload: map[string]interface{}{
					"providerData":  cloneMap(resume.ProviderData),
					"resumedUpload": true,
				},
			},
			ConflictAction: conflictAction,
		}
	}
	commonPayload := map[string]interface{}{
		"sessionResponse":      authSession.RawPayload,
		"uploadId":             strconv.FormatInt(uploadFileID, 10),
		"uploadFileId":         uploadFileID,
		"fileUploadUrl":        fileUploadURL,
		"fileCommitUrl":        fileCommitURL,
		"resolvedTargetName":   targetName,
		"conflictAction":       conflictAction,
		"md5":                  actualMD5,
		"parentId":             parentID,
		"providedWriteHeaders": session.Signature != "" && session.Date != "",
		"usedBinaryFallback":   true,
		"resumedUpload":        true,
		"providerData":         cloneMap(resume.ProviderData),
	}
	cloud189ApplyWholeObjectProgress(commonPayload, "", nil, req.Size, false)
	putStatus, putPayload, putErr := cloud189UploadBinaryToURL(context.Background(), localPath, fileUploadURL, uploadFileID)
	commonPayload["binaryUploadResponse"] = putPayload
	commonPayload["uploadPutStatus"] = putStatus
	if putErr != nil {
		cloud189ApplyWholeObjectProgress(commonPayload, "", putPayload, req.Size, false)
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(putErr),
				Message: fmt.Sprintf("189Cloud resumed binary PUT upload failed: %v", putErr),
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	cloud189ApplyWholeObjectProgress(commonPayload, "", putPayload, req.Size, true)
	_, statusPayload, statusErr := cloud189SignedJSON(context.Background(), session, authSession, http.MethodGet, session.UploadStatusEndpoint, map[string]string{
		"uploadFileId": strconv.FormatInt(uploadFileID, 10),
		"resumePolicy": "1",
	}, nil)
	commonPayload["statusResponse"] = statusPayload
	if statusErr != nil {
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(statusErr),
				Message: fmt.Sprintf("189Cloud getUploadFileStatus request failed during resumed upload: %v", statusErr),
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	statusView := cloud189ExtractStatusView(statusPayload, uploadFileID, fileUploadURL, fileCommitURL)
	commonPayload["statusView"] = statusView
	_, commitText, commitErr := cloud189SignedXML(context.Background(), session, authSession, http.MethodPost, stringMapValue(statusView, "fileCommitUrl"), map[string]string{
		"opertype":     "3",
		"resumePolicy": "1",
		"uploadFileId": strconv.FormatInt(int64MapValue(statusView, "uploadFileId"), 10),
		"isLog":        "0",
	})
	if commitErr != nil {
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  normalizeCloud189RequestErrorStatus(commitErr),
				Message: fmt.Sprintf("189Cloud resumed binary upload commit request failed: %v", commitErr),
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	commitPayload, parseErr := cloud189ParseCommitXML(commitText)
	if parseErr != nil {
		return &UploadResult{
			OperationResult: OperationResult{
				Status:  "provider_request_failed",
				Message: fmt.Sprintf("189Cloud resumed binary upload commit XML decode failed: %v", parseErr),
				Mode:    "cloud189_family_real_upload",
				Payload: commonPayload,
			},
			ConflictAction: conflictAction,
		}
	}
	commonPayload["commitResponse"] = map[string]interface{}{
		"fileId":     commitPayload.FileID,
		"name":       commitPayload.Name,
		"size":       commitPayload.Size,
		"md5":        commitPayload.MD5,
		"createDate": commitPayload.CreateDate,
		"rawXml":     commitPayload.RawXML,
	}
	commonPayload["fileId"] = commitPayload.FileID
	cloud189ApplyWholeObjectProgress(commonPayload, commitPayload.FileID, putPayload, req.Size, true)
	verifyOK := strings.ToLower(strings.TrimSpace(commitPayload.MD5)) == "" || strings.ToLower(strings.TrimSpace(commitPayload.MD5)) == actualMD5
	commonPayload["verifyMode"] = "commit_response_xml_after_binary_put"
	commonPayload["verifyOk"] = verifyOK
	commonPayload["verifyPayload"] = map[string]interface{}{
		"fileId":     commitPayload.FileID,
		"name":       commitPayload.Name,
		"size":       commitPayload.Size,
		"md5":        commitPayload.MD5,
		"createDate": commitPayload.CreateDate,
		"statusView": statusView,
	}
	return &UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "189Cloud resumed a cached binary upload session and the provider commit response confirmed the file.",
			Mode:    "cloud189_family_real_upload",
			Payload: commonPayload,
		},
		ConflictAction: conflictAction,
	}
}

func cloud189ApplyWholeObjectProgress(payload map[string]interface{}, fileID string, uploadPayload map[string]interface{}, size int64, completed bool) {
	payload["partCount"] = 1
	if strings.TrimSpace(fileID) != "" {
		payload["fileId"] = fileID
	}
	if completed {
		objectSize := int64MapValue(uploadPayload, "objectSize")
		if objectSize <= 0 {
			objectSize = size
		}
		payload["uploadedPartCount"] = 1
		payload["uploadedParts"] = []map[string]interface{}{
			{
				"partNumber": 1,
				"status":     int64MapValue(uploadPayload, "status"),
				"size":       objectSize,
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

func (a Cloud189FamilyAdapter) newCloud189Session(profile AuthProfile) (cloud189Session, error) {
	if strings.TrimSpace(profile.Cookie) == "" {
		return cloud189Session{}, fmt.Errorf("189Cloud adapter requires a cookie")
	}
	accessToken := normalizeBaiduAccessToken(profile.Token)
	if accessToken == "" {
		accessToken = normalizeBaiduAccessToken(firstNonEmptyExtra(profile.Extra, "authorization", "Authorization", "accessToken", "access_token"))
	}
	shareInfoEndpoint, err := resolveCloud189Endpoint(profile, "shareInfoEndpoint", cloud189ShareInfoURL)
	if err != nil {
		return cloud189Session{}, err
	}
	checkAccessEndpoint, err := resolveCloud189Endpoint(profile, "checkAccessEndpoint", cloud189CheckAccessURL)
	if err != nil {
		return cloud189Session{}, err
	}
	listEndpoint, err := resolveCloud189Endpoint(profile, "listEndpoint", cloud189ListShareDirURL)
	if err != nil {
		return cloud189Session{}, err
	}
	createDirEndpoint, err := resolveCloud189Endpoint(profile, "createDirEndpoint", cloud189CreateFolderURL)
	if err != nil {
		return cloud189Session{}, err
	}
	authSessionEndpoint, err := resolveCloud189Endpoint(profile, "authSessionEndpoint", cloud189SessionAuthURL)
	if err != nil {
		return cloud189Session{}, err
	}
	createUploadEndpoint, err := resolveCloud189Endpoint(profile, "createUploadEndpoint", cloud189CreateUploadURL)
	if err != nil {
		return cloud189Session{}, err
	}
	uploadStatusEndpoint, err := resolveCloud189Endpoint(profile, "uploadStatusEndpoint", cloud189UploadStatusURL)
	if err != nil {
		return cloud189Session{}, err
	}
	return cloud189Session{
		ShareInfoEndpoint:    shareInfoEndpoint,
		CheckAccessEndpoint:  checkAccessEndpoint,
		ListEndpoint:         listEndpoint,
		CreateDirEndpoint:    createDirEndpoint,
		AuthSessionEndpoint:  authSessionEndpoint,
		CreateUploadEndpoint: createUploadEndpoint,
		UploadStatusEndpoint: uploadStatusEndpoint,
		Cookie:               strings.TrimSpace(profile.Cookie),
		ShareCode:            firstNonEmptyExtra(profile.Extra, "shareCode", "sharecode"),
		AccessCode:           firstNonEmptyExtra(profile.Extra, "accessCode", "accesscode", "passcode"),
		AccessToken:          accessToken,
		Signature:            firstNonEmptyExtra(profile.Extra, "signature", "Signature"),
		Date:                 firstNonEmptyExtra(profile.Extra, "date", "Date"),
		PathPrefix:           firstNonEmptyExtra(profile.Extra, "pathPrefix"),
		ProviderKey:          a.MetaInfo.Key,
	}, nil
}

func resolveCloud189Endpoint(profile AuthProfile, key string, fallback string) (string, error) {
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

func cloud189Headers(session cloud189Session, form bool, shareCode string) map[string]string {
	headers := map[string]string{
		"Accept":     "application/json;charset=UTF-8",
		"Origin":     "https://cloud.189.cn",
		"Referer":    "https://cloud.189.cn/web/",
		"Sign-Type":  "1",
		"User-Agent": "CloudPanSync/0.1",
		"Cookie":     session.Cookie,
	}
	if strings.TrimSpace(shareCode) != "" {
		headers["Referer"] = "https://cloud.189.cn/web/share?code=" + strings.TrimSpace(shareCode)
	}
	if form {
		headers["Content-Type"] = "application/x-www-form-urlencoded"
	}
	return headers
}

func cloud189AccountWriteHeaders(session cloud189Session, form bool) map[string]string {
	headers := cloud189Headers(session, form, "")
	headers["AccessToken"] = session.AccessToken
	headers["Accesstoken"] = session.AccessToken
	headers["Signature"] = session.Signature
	headers["Date"] = session.Date
	headers["Timestamp"] = session.Date
	return headers
}

func cloud189RequestJSON(ctx context.Context, method string, endpoint string, headers map[string]string, body string) (int, map[string]interface{}, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	if method == http.MethodGet {
		req.Body = nil
		req.GetBody = nil
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
	payload, err := decodeProviderJSONResponse(resp.StatusCode, bodyBytes)
	if err != nil {
		return resp.StatusCode, nil, err
	}
	return resp.StatusCode, payload, nil
}

func cloud189FetchShareInfo(ctx context.Context, session cloud189Session) (int, map[string]interface{}, error) {
	return cloud189RequestJSON(ctx, http.MethodPost, session.ShareInfoEndpoint, cloud189Headers(session, true, session.ShareCode), url.Values{
		"shareCode": []string{session.ShareCode},
	}.Encode())
}

func cloud189FetchShareID(ctx context.Context, session cloud189Session, shareInfo map[string]interface{}) (int, string, map[string]interface{}, error) {
	direct := firstNonEmptyString(shareInfo, "shareId", "shareID")
	if direct != "" {
		return http.StatusOK, direct, shareInfo, nil
	}
	values := url.Values{}
	values.Set("shareCode", session.ShareCode)
	if strings.TrimSpace(session.AccessCode) != "" {
		values.Set("accessCode", session.AccessCode)
	}
	statusCode, payload, err := cloud189RequestJSON(ctx, http.MethodGet, session.CheckAccessEndpoint+"?"+values.Encode(), cloud189Headers(session, false, session.ShareCode), "")
	if err != nil {
		return statusCode, "", nil, err
	}
	ok, _ := cloud189ResponseOK(payload)
	if !ok {
		return statusCode, "", payload, nil
	}
	return statusCode, firstNonEmptyString(cloud189ResponseData(payload), "shareId", "shareID"), payload, nil
}

func cloud189FetchDirPage(ctx context.Context, session cloud189Session, shareID string, shareMode int, fileID string, pageSize int) (int, map[string]interface{}, error) {
	values := url.Values{}
	values.Set("pageNum", "1")
	values.Set("pageSize", strconv.Itoa(pageSize))
	values.Set("fileId", fileID)
	values.Set("shareDirFileId", fileID)
	values.Set("isFolder", "true")
	values.Set("shareId", shareID)
	values.Set("shareMode", strconv.Itoa(shareMode))
	values.Set("iconOption", "5")
	values.Set("orderBy", "lastOpTime")
	values.Set("descending", "true")
	if strings.TrimSpace(session.AccessCode) != "" {
		values.Set("accessCode", session.AccessCode)
	}
	return cloud189RequestJSON(ctx, http.MethodGet, session.ListEndpoint+"?"+values.Encode(), cloud189Headers(session, false, session.ShareCode), "")
}

func cloud189RequestCreateFolder(ctx context.Context, session cloud189Session, parentID string, dirName string) (int, map[string]interface{}, error) {
	values := url.Values{}
	values.Set("parentFolderId", parentID)
	values.Set("folderName", dirName)
	return cloud189RequestJSON(ctx, http.MethodPost, session.CreateDirEndpoint, cloud189AccountWriteHeaders(session, true), values.Encode())
}

func cloud189ResponseOK(payload map[string]interface{}) (bool, string) {
	code := firstNonEmptyString(payload, "res_code", "resCode", "code")
	if code == "" {
		code = "0"
	}
	return code == "0", code
}

func cloud189ResponseData(payload map[string]interface{}) map[string]interface{} {
	data, _ := payload["data"].(map[string]interface{})
	if data != nil {
		return data
	}
	return payload
}

func cloud189ShareMode(shareInfo map[string]interface{}) int {
	value := firstNonEmptyString(shareInfo, "shareMode", "share_mode")
	if value == "" {
		return 1
	}
	number, err := strconv.Atoi(value)
	if err != nil || number == 0 {
		return 1
	}
	return number
}

func clampCloud189Limit(pageSize int) int {
	if pageSize <= 0 {
		return 100
	}
	if pageSize > 200 {
		return 200
	}
	return pageSize
}

func normalizeCloud189ListItems(payload map[string]interface{}, parentID string, pathPrefix string) []map[string]interface{} {
	root := cloud189ExtractFileListAO(payload)
	rows := make([]map[string]interface{}, 0)
	for _, raw := range interfaceSliceValue(root, "folderList") {
		folder, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name := cloud189ItemName(folder)
		fileID := cloud189ItemID(folder)
		if name == "" || fileID == "" {
			continue
		}
		rows = append(rows, map[string]interface{}{
			"exists":   true,
			"fileId":   fileID,
			"parentId": parentID,
			"name":     name,
			"path":     strings.ReplaceAll(pathPrefix+"/"+name, "//", "/"),
			"type":     "dir",
			"isDir":    true,
			"size":     int64(0),
			"md5":      "",
			"etag":     "",
			"provider": "189cloud",
			"raw":      folder,
		})
	}
	for _, raw := range interfaceSliceValue(root, "fileList") {
		file, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name := cloud189ItemName(file)
		fileID := cloud189ItemID(file)
		if name == "" || fileID == "" {
			continue
		}
		md5Value := cloud189ItemMD5(file)
		rows = append(rows, map[string]interface{}{
			"exists":   true,
			"fileId":   fileID,
			"parentId": parentID,
			"name":     name,
			"path":     strings.ReplaceAll(pathPrefix+"/"+name, "//", "/"),
			"type":     "file",
			"isDir":    false,
			"size":     cloud189ItemSize(file),
			"md5":      md5Value,
			"etag":     md5Value,
			"provider": "189cloud",
			"raw":      file,
		})
	}
	return rows
}

func cloud189ExtractFileListAO(payload map[string]interface{}) map[string]interface{} {
	if value, ok := payload["fileListAO"].(map[string]interface{}); ok {
		return value
	}
	if data, ok := payload["data"].(map[string]interface{}); ok {
		if value, ok := data["fileListAO"].(map[string]interface{}); ok {
			return value
		}
		return data
	}
	return payload
}

func cloud189ItemName(item map[string]interface{}) string {
	return firstNonEmptyString(item, "name", "fileName", "filename", "file_name")
}

func cloud189ItemID(item map[string]interface{}) string {
	return firstNonEmptyString(item, "id", "fileId", "fileID", "file_id")
}

func cloud189ItemSize(item map[string]interface{}) int64 {
	for _, key := range []string{"size", "fileSize", "file_size", "bytes"} {
		if value := int64MapValue(item, key); value != 0 {
			return value
		}
	}
	return 0
}

func cloud189ItemMD5(item map[string]interface{}) string {
	value := strings.ToLower(firstNonEmptyString(item, "md5", "fileMd5", "file_md5", "etag"))
	if len(value) != 32 {
		return ""
	}
	for _, ch := range value {
		if !strings.ContainsRune("0123456789abcdef", ch) {
			return ""
		}
	}
	return value
}

func extractCloud189CreatedFolderID(payload map[string]interface{}) string {
	candidates := []map[string]interface{}{
		payload,
		cloud189ResponseData(payload),
	}
	if data := cloud189ResponseData(payload); data != nil {
		if createVO, ok := data["createFolderVO"].(map[string]interface{}); ok {
			candidates = append(candidates, createVO)
		}
	}
	if createVO, ok := payload["createFolderVO"].(map[string]interface{}); ok {
		candidates = append(candidates, createVO)
	}
	for _, candidate := range candidates {
		if candidate == nil {
			continue
		}
		if value := firstNonEmptyString(candidate, "id", "fileId", "folderId", "folderID", "fileID"); value != "" {
			return value
		}
	}
	return ""
}

func cloud189ClientSuffix() map[string]string {
	return map[string]string{
		"clientType": cloud189ClientType,
		"version":    cloud189ClientVer,
		"channelId":  cloud189ClientChanID,
		"rand":       fmt.Sprintf("%d_%d", rand.Intn(100000), time.Now().UnixNano()%10000000000),
	}
}

func cloud189RequestID() string {
	return fmt.Sprintf("req-%d-%d", time.Now().UnixNano(), rand.Intn(100000))
}

func cloud189HTTPSignedDate() string {
	return time.Now().UTC().Format(http.TimeFormat)
}

func cloud189UploadSignature(sessionSecret string, sessionKey string, method string, fullURL string, dateOfGMT string) string {
	parsed, err := url.Parse(fullURL)
	requestPath := "/"
	if err == nil && parsed.Path != "" {
		requestPath = parsed.Path
	}
	data := fmt.Sprintf("SessionKey=%s&Operate=%s&RequestURI=%s&Date=%s", sessionKey, method, requestPath, dateOfGMT)
	mac := hmac.New(sha1.New, []byte(sessionSecret))
	_, _ = mac.Write([]byte(data))
	return strings.ToUpper(fmt.Sprintf("%x", mac.Sum(nil)))
}

func cloud189SignedHeaders(authSession cloud189UploadAuthSession, method string, fullURL string) map[string]string {
	dateOfGMT := cloud189HTTPSignedDate()
	return map[string]string{
		"Date":         dateOfGMT,
		"SessionKey":   authSession.SessionKey,
		"X-Request-ID": cloud189RequestID(),
		"Signature":    cloud189UploadSignature(authSession.SessionSecret, authSession.SessionKey, method, fullURL, dateOfGMT),
		"Accept":       "application/json;charset=UTF-8",
		"Referer":      "https://cloud.189.cn/",
		"User-Agent":   "CloudPanSync/0.1",
	}
}

func cloud189RawRequest(ctx context.Context, method string, endpoint string, query map[string]string, headers map[string]string, form map[string]string, body io.Reader) (int, []byte, http.Header, error) {
	requestURL := endpoint
	if len(query) > 0 {
		values := url.Values{}
		for key, value := range query {
			if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
				continue
			}
			values.Set(key, value)
		}
		if encoded := values.Encode(); encoded != "" {
			if strings.Contains(requestURL, "?") {
				requestURL += "&" + encoded
			} else {
				requestURL += "?" + encoded
			}
		}
	}
	var requestBody io.Reader
	if body != nil {
		requestBody = body
	} else if len(form) > 0 {
		values := url.Values{}
		for key, value := range form {
			if value == "" {
				continue
			}
			values.Set(key, value)
		}
		requestBody = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL, requestBody)
	if err != nil {
		return 0, nil, nil, err
	}
	for key, value := range headers {
		if strings.TrimSpace(value) == "" {
			continue
		}
		req.Header.Set(key, value)
	}
	if len(form) > 0 && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	resp, err := providerHTTPClient.Do(req)
	if err != nil {
		return 0, nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return resp.StatusCode, nil, resp.Header.Clone(), err
	}
	return resp.StatusCode, bodyBytes, resp.Header.Clone(), nil
}

func cloud189GetUploadSession(ctx context.Context, session cloud189Session) (cloud189UploadAuthSession, error) {
	query := cloud189ClientSuffix()
	query["appId"] = cloud189AppID
	query["accessToken"] = session.AccessToken
	statusCode, bodyBytes, _, err := cloud189RawRequest(ctx, http.MethodGet, session.AuthSessionEndpoint, query, map[string]string{
		"X-Request-ID": cloud189RequestID(),
		"Cookie":       session.Cookie,
		"User-Agent":   "CloudPanSync/0.1",
	}, nil, nil)
	if err != nil {
		return cloud189UploadAuthSession{}, err
	}
	var payload map[string]interface{}
	if len(bodyBytes) > 0 {
		payload, err = decodeProviderJSONResponse(statusCode, bodyBytes)
		if err != nil {
			return cloud189UploadAuthSession{}, err
		}
	}
	if payload == nil {
		payload = map[string]interface{}{}
	}
	if statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden {
		return cloud189UploadAuthSession{}, fmt.Errorf("auth_invalid")
	}
	return cloud189UploadAuthSession{
		SessionKey:    firstNonEmptyString(payload, "sessionKey"),
		SessionSecret: firstNonEmptyString(payload, "sessionSecret"),
		StatusCode:    statusCode,
		RawPayload:    payload,
	}, nil
}

func cloud189SignedJSON(ctx context.Context, session cloud189Session, authSession cloud189UploadAuthSession, method string, endpoint string, query map[string]string, form map[string]string) (int, map[string]interface{}, error) {
	mergedQuery := cloud189ClientSuffix()
	for key, value := range query {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		mergedQuery[key] = value
	}
	headers := cloud189SignedHeaders(authSession, method, endpoint)
	headers["Cookie"] = session.Cookie
	statusCode, bodyBytes, _, err := cloud189RawRequest(ctx, method, endpoint, mergedQuery, headers, form, nil)
	if err != nil {
		return 0, nil, err
	}
	payload, err := decodeProviderJSONResponse(statusCode, bodyBytes)
	if err != nil {
		return statusCode, nil, err
	}
	return statusCode, payload, nil
}

func cloud189SignedXML(ctx context.Context, session cloud189Session, authSession cloud189UploadAuthSession, method string, endpoint string, form map[string]string) (int, string, error) {
	headers := cloud189SignedHeaders(authSession, method, endpoint)
	headers["Cookie"] = session.Cookie
	statusCode, bodyBytes, _, err := cloud189RawRequest(ctx, method, endpoint, cloud189ClientSuffix(), headers, form, nil)
	if err != nil {
		return 0, "", err
	}
	return statusCode, string(bodyBytes), nil
}

func computeCloud189LocalMD5(localPath string) (string, error) {
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

func (a Cloud189FamilyAdapter) resolveCloud189UploadName(profile AuthProfile, parentID string, targetName string, policy ConflictPolicy) (string, string, string) {
	if strings.TrimSpace(profile.Extra["shareCode"]) == "" && strings.TrimSpace(profile.Extra["sharecode"]) == "" {
		return targetName, "conflict_check_unavailable", "Could not verify same-name conflicts before 189Cloud upload, so the original file name was kept."
	}
	listResult := a.List(ListRequest{
		Profile:  profile,
		ParentID: parentID,
		PageSize: 200,
	})
	if !listResult.OK {
		return targetName, "conflict_check_unavailable", "Could not verify same-name conflicts before 189Cloud upload, so the original file name was kept."
	}
	existing := make(map[string]bool, len(listResult.Items))
	for _, item := range listResult.Items {
		name := strings.TrimSpace(stringMapValue(item, "name"))
		if name != "" {
			existing[name] = true
		}
	}
	if !existing[targetName] {
		return targetName, "none", ""
	}
	index := 1
	stem, suffix := splitHashFamilyName(targetName)
	candidate := targetName
	for existing[candidate] {
		candidate = fmt.Sprintf("%s (%d)%s", stem, index, suffix)
		index++
	}
	if policy == ConflictPolicyAutoRenameNew {
		return candidate, "auto_rename_new", "A same-name file already exists under the target path, so 189Cloud upload auto-renamed the new file."
	}
	return candidate, "overwrite_downgraded_to_auto_rename", "The requested overwrite policy was downgraded because the current 189Cloud upload path does not support verified in-place overwrite."
}

func cloud189UploadBinaryToURL(ctx context.Context, localPath string, uploadURL string, uploadFileID int64) (int, map[string]interface{}, error) {
	fileBytes, err := os.ReadFile(localPath)
	if err != nil {
		return 0, nil, err
	}
	statusCode, _, headers, err := cloud189RawRequest(ctx, http.MethodPut, uploadURL, nil, map[string]string{
		"ResumePolicy":        "1",
		"Expect":              "100-continue",
		"Edrive-UploadFileId": strconv.FormatInt(uploadFileID, 10),
		"User-Agent":          "CloudPanSync/0.1",
	}, nil, strings.NewReader(string(fileBytes)))
	if err != nil {
		return 0, nil, err
	}
	headerMap := map[string]interface{}{}
	for key, values := range headers {
		if len(values) == 1 {
			headerMap[key] = values[0]
		} else {
			headerMap[key] = values
		}
	}
	payload := map[string]interface{}{
		"status":     statusCode,
		"headers":    headerMap,
		"objectSize": int64(len(fileBytes)),
	}
	if statusCode < 200 || statusCode >= 300 {
		return statusCode, payload, fmt.Errorf("cloud189 binary put returned HTTP %d", statusCode)
	}
	return statusCode, payload, nil
}

func cloud189ExtractStatusView(payload map[string]interface{}, uploadFileID int64, fallbackUploadURL string, fallbackCommitURL string) map[string]interface{} {
	data := payload
	if typed, ok := payload["data"].(map[string]interface{}); ok {
		data = typed
	}
	dataSize := int64MapValue(data, "dataSize")
	sizeValue := int64MapValue(data, "size")
	fileDataExists := int64MapValue(data, "fileDataExists")
	resolvedUploadFileID := cloud189UploadFileID(data)
	if resolvedUploadFileID == 0 {
		resolvedUploadFileID = uploadFileID
	}
	fileUploadURL := firstNonEmptyString(data, "fileUploadUrl")
	if fileUploadURL == "" {
		fileUploadURL = fallbackUploadURL
	}
	fileCommitURL := firstNonEmptyString(data, "fileCommitUrl")
	if fileCommitURL == "" {
		fileCommitURL = fallbackCommitURL
	}
	return map[string]interface{}{
		"uploadFileId":   resolvedUploadFileID,
		"fileUploadUrl":  fileUploadURL,
		"fileCommitUrl":  fileCommitURL,
		"fileDataExists": fileDataExists,
		"dataSize":       dataSize,
		"size":           sizeValue,
		"uploadedBytes":  dataSize + sizeValue,
		"raw":            payload,
	}
}

func cloud189ParseCommitXML(text string) (cloud189CommitResult, error) {
	var result cloud189CommitResult
	if err := xml.Unmarshal([]byte(text), &result); err != nil {
		return cloud189CommitResult{}, err
	}
	result.MD5 = strings.ToLower(strings.TrimSpace(result.MD5))
	result.RawXML = text
	return result, nil
}

func cloud189UploadFileID(payload map[string]interface{}) int64 {
	if payload == nil {
		return 0
	}
	if value := int64MapValue(payload, "uploadFileId"); value != 0 {
		return value
	}
	text := firstNonEmptyString(payload, "uploadFileId")
	if text == "" {
		return 0
	}
	number, err := strconv.ParseInt(strings.TrimSpace(text), 10, 64)
	if err != nil {
		return 0
	}
	return number
}

func normalizeCloud189SessionErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "requires a cookie") {
		return "missing_cookie"
	}
	if strings.Contains(err.Error(), "invalid ") {
		return "invalid_provider_endpoint"
	}
	return "provider_request_failed"
}

func normalizeCloud189RequestErrorStatus(err error) string {
	if err == nil {
		return "provider_request_failed"
	}
	if strings.Contains(err.Error(), "auth_invalid") {
		return "auth_invalid"
	}
	return "provider_request_failed"
}
