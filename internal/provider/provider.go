package provider

import "strings"

type RiskTemplateSummary struct {
	RecommendedMode       string   `json:"recommendedMode,omitempty"`
	Base                  any      `json:"base,omitempty"`
	Calibrated            any      `json:"calibrated,omitempty"`
	RecoverBudget         any      `json:"recoverBudget,omitempty"`
	CalibrationReasons    []string `json:"calibrationReasons,omitempty"`
	ProviderRiskHints     []string `json:"providerRiskHints,omitempty"`
	ProviderRiskTraits    []string `json:"providerRiskTraits,omitempty"`
	RecommendedReason     string   `json:"recommendedReason,omitempty"`
	AggressiveRiskWarning string   `json:"aggressiveRiskWarning,omitempty"`
}

type ConflictPolicy string

const (
	ConflictPolicyOverwriteExisting ConflictPolicy = "overwrite_existing"
	ConflictPolicyAutoRenameNew     ConflictPolicy = "auto_rename_new"
)

type Provider struct {
	Key                 string              `json:"key"`
	DisplayName         string              `json:"displayName"`
	ProtocolGroup       string              `json:"protocolGroup"`
	RiskHints           []string            `json:"riskHints,omitempty"`
	RiskTraits          []string            `json:"riskTraits,omitempty"`
	DefaultRiskTemplate RiskTemplateSummary `json:"defaultRiskTemplate,omitempty"`
	AuthModes           []string            `json:"authModes"`
	FastUploadInputs    []string            `json:"fastUploadInputs"`
	FallbackModes       []string            `json:"fallbackModes"`
	ConflictPolicies    []ConflictPolicy    `json:"conflictPolicies"`
	SupportsOverwrite   bool                `json:"supportsOverwrite"`
	SupportsAutoRename  bool                `json:"supportsAutoRename"`
	OverwriteBehavior   string              `json:"overwriteBehavior"`
	Status              string              `json:"status"`
}

type CapabilitySet struct {
	SupportsAuthValidation bool `json:"supportsAuthValidation"`
	SupportsList           bool `json:"supportsList"`
	SupportsMetadata       bool `json:"supportsMetadata"`
	SupportsCreateDir      bool `json:"supportsCreateDir"`
	SupportsFastUpload     bool `json:"supportsFastUpload"`
	SupportsUpload         bool `json:"supportsUpload"`
}

type AuthProfile struct {
	ID          string            `json:"id"`
	ProviderKey string            `json:"providerKey"`
	AuthMode    string            `json:"authMode"`
	DisplayName string            `json:"displayName"`
	Token       string            `json:"token,omitempty"`
	Cookie      string            `json:"cookie,omitempty"`
	Extra       map[string]string `json:"extra,omitempty"`
}

type OperationResult struct {
	OK      bool                   `json:"ok"`
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Mode    string                 `json:"mode,omitempty"`
	Payload map[string]interface{} `json:"payload,omitempty"`
}

type ListRequest struct {
	Profile  AuthProfile `json:"profile"`
	Path     string      `json:"path"`
	ParentID string      `json:"parentId"`
	PageSize int         `json:"pageSize"`
}

type ListResult struct {
	OperationResult
	Items []map[string]interface{} `json:"items,omitempty"`
}

type MetadataRequest struct {
	Profile  AuthProfile `json:"profile"`
	Path     string      `json:"path"`
	FileID   string      `json:"fileId"`
	ParentID string      `json:"parentId"`
}

type MetadataResult struct {
	OperationResult
	Entry map[string]interface{} `json:"entry,omitempty"`
}

type CreateDirRequest struct {
	Profile  AuthProfile `json:"profile"`
	ParentID string      `json:"parentId"`
	DirName  string      `json:"dirName"`
}

type FastUploadCheckRequest struct {
	Profile  AuthProfile `json:"profile"`
	Path     string      `json:"path"`
	ParentID string      `json:"parentId"`
	Name     string      `json:"name"`
	Size     int64       `json:"size"`
	MD5      string      `json:"md5,omitempty"`
	SHA1     string      `json:"sha1,omitempty"`
	GCID     string      `json:"gcid,omitempty"`
}

type FastUploadCheckResult struct {
	OperationResult
	Candidate bool `json:"candidate"`
}

type UploadRequest struct {
	Profile        AuthProfile    `json:"profile"`
	Path           string         `json:"path"`
	ParentID       string         `json:"parentId"`
	Name           string         `json:"name"`
	Size           int64          `json:"size"`
	LocalPath      string         `json:"localPath,omitempty"`
	ConflictPolicy ConflictPolicy `json:"conflictPolicy"`
	Strategy       string         `json:"strategy"`
	MD5            string         `json:"md5,omitempty"`
	SHA1           string         `json:"sha1,omitempty"`
	GCID           string         `json:"gcid,omitempty"`
	ResumeUpload   *ResumeUpload  `json:"resumeUpload,omitempty"`
}

type UploadResult struct {
	OperationResult
	ConflictAction string `json:"conflictAction,omitempty"`
}

type ResumeUpload struct {
	ItemPath          string                   `json:"itemPath,omitempty"`
	ProviderStatus    string                   `json:"providerStatus,omitempty"`
	UpdatedAt         string                   `json:"updatedAt,omitempty"`
	FileID            string                   `json:"fileId,omitempty"`
	UploadID          string                   `json:"uploadId,omitempty"`
	PartCount         int                      `json:"partCount,omitempty"`
	UploadedPartCount int                      `json:"uploadedPartCount,omitempty"`
	FailedPartNumber  int                      `json:"failedPartNumber,omitempty"`
	NextPartNumber    int                      `json:"nextPartNumber,omitempty"`
	UploadedParts     []map[string]interface{} `json:"uploadedParts,omitempty"`
	ProviderData      map[string]interface{}   `json:"providerData,omitempty"`
}

type Adapter interface {
	Meta() Provider
	Capabilities() CapabilitySet
	ValidateAuth(profile AuthProfile) OperationResult
	List(req ListRequest) ListResult
	Metadata(req MetadataRequest) MetadataResult
	CreateDir(req CreateDirRequest) OperationResult
	FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult
	Upload(req UploadRequest) UploadResult
}

type StaticAdapter struct {
	MetaInfo       Provider
	CapabilityInfo CapabilitySet
}

func (a StaticAdapter) Meta() Provider {
	return a.MetaInfo
}

func (a StaticAdapter) Capabilities() CapabilitySet {
	return a.CapabilityInfo
}

func (a StaticAdapter) ValidateAuth(profile AuthProfile) OperationResult {
	if strings.TrimSpace(profile.Token) == "" && strings.TrimSpace(profile.Cookie) == "" && len(profile.Extra) == 0 {
		return OperationResult{
			Status:  "missing_credentials",
			Message: "At least one credential source is required.",
			Mode:    "static_placeholder",
		}
	}
	return OperationResult{
		OK:      true,
		Status:  "verified",
		Message: "Static adapter validation accepted scaffold credentials.",
		Mode:    "static_placeholder",
	}
}

func (a StaticAdapter) List(req ListRequest) ListResult {
	return ListResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Static adapter returned placeholder list data.",
			Mode:    "static_placeholder",
		},
		Items: []map[string]interface{}{
			{
				"name":     inferName(req.Path, "placeholder.txt"),
				"path":     defaultPath(req.Path, "/"),
				"provider": a.MetaInfo.Key,
			},
		},
	}
}

func (a StaticAdapter) Metadata(req MetadataRequest) MetadataResult {
	return MetadataResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Static adapter returned placeholder metadata.",
			Mode:    "static_placeholder",
		},
		Entry: map[string]interface{}{
			"name":     inferName(req.Path, "placeholder.txt"),
			"path":     defaultPath(req.Path, "/"),
			"provider": a.MetaInfo.Key,
		},
	}
}

func (a StaticAdapter) CreateDir(req CreateDirRequest) OperationResult {
	return OperationResult{
		OK:      true,
		Status:  "ok",
		Message: "Static adapter accepted create-dir request in scaffold mode.",
		Mode:    "static_placeholder",
		Payload: map[string]interface{}{
			"parentId": req.ParentID,
			"dirName":  req.DirName,
		},
	}
}

func (a StaticAdapter) FastUploadCheck(req FastUploadCheckRequest) FastUploadCheckResult {
	return FastUploadCheckResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Static adapter evaluated fast-upload candidate in scaffold mode.",
			Mode:    "static_placeholder",
		},
		Candidate: strings.TrimSpace(req.MD5) != "" || strings.TrimSpace(req.SHA1) != "" || strings.TrimSpace(req.GCID) != "",
	}
}

func (a StaticAdapter) Upload(req UploadRequest) UploadResult {
	return UploadResult{
		OperationResult: OperationResult{
			OK:      true,
			Status:  "ok",
			Message: "Static adapter accepted upload request in scaffold mode.",
			Mode:    "static_placeholder",
		},
		ConflictAction: "none",
	}
}

func defaultPath(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func inferName(path, fallback string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" || trimmed == "/" {
		return fallback
	}
	parts := strings.Split(trimmed, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.TrimSpace(parts[i]) != "" {
			return parts[i]
		}
	}
	return fallback
}
