package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"

	"cloudpan-sync-go/internal/auth"
	"cloudpan-sync-go/internal/planner"
	"cloudpan-sync-go/internal/provider"
	"cloudpan-sync-go/internal/task"
)

type loginRequest struct {
	Password string `json:"password"`
}

type providerListRequest struct {
	ProfileID string `json:"profileId"`
	Path      string `json:"path"`
	ParentID  string `json:"parentId"`
	PageSize  int    `json:"pageSize"`
}

type providerMetadataRequest struct {
	ProfileID string `json:"profileId"`
	Path      string `json:"path"`
	FileID    string `json:"fileId"`
	ParentID  string `json:"parentId"`
}

type providerCreateDirRequest struct {
	ProfileID string `json:"profileId"`
	ParentID  string `json:"parentId"`
	DirName   string `json:"dirName"`
}

type providerFastCheckRequest struct {
	ProfileID string `json:"profileId"`
	Path      string `json:"path"`
	ParentID  string `json:"parentId"`
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	MD5       string `json:"md5"`
	SHA1      string `json:"sha1"`
	GCID      string `json:"gcid"`
}

type providerUploadRequest struct {
	ProfileID      string `json:"profileId"`
	Path           string `json:"path"`
	ParentID       string `json:"parentId"`
	Name           string `json:"name"`
	Size           int64  `json:"size"`
	LocalPath      string `json:"localPath"`
	ConflictPolicy string `json:"conflictPolicy"`
	Strategy       string `json:"strategy"`
	MD5            string `json:"md5"`
	SHA1           string `json:"sha1"`
	GCID           string `json:"gcid"`
}

type evidenceReportRequest struct {
	Title string `json:"title"`
	Note  string `json:"note"`
}

type providerSmokeRecordRequest struct {
	ProviderKey   string            `json:"providerKey"`
	ProtocolGroup string            `json:"protocolGroup"`
	AuthMode      string            `json:"authMode"`
	Category      string            `json:"category"`
	Result        string            `json:"result"`
	Title         string            `json:"title"`
	Note          string            `json:"note"`
	Operations    []string          `json:"operations"`
	Environment   map[string]string `json:"environment"`
}

type retryTaskRequest struct {
	Paths []string `json:"paths"`
	Scope string   `json:"scope"`
}

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleIndex)
	mux.Handle("/assets/", a.webStatic)
	mux.HandleFunc("/api/health", a.handleHealth)
	mux.HandleFunc("/api/session/login", a.handleLogin)
	mux.HandleFunc("/api/providers", a.handleProviders)
	mux.HandleFunc("/api/providers/", a.handleProviderByKey)
	mux.HandleFunc("/api/auth/profiles", a.handleAuthProfiles)
	mux.HandleFunc("/api/auth/profiles/", a.handleAuthProfileByID)
	mux.HandleFunc("/api/plans/preview", a.handlePlanPreview)
	mux.HandleFunc("/api/tasks", a.handleTasks)
	mux.HandleFunc("/api/tasks/", a.handleTaskByID)
	mux.HandleFunc("/api/evidence/runtime", a.handleRuntimeEvidence)
	mux.HandleFunc("/api/evidence/report", a.handleEvidenceReport)
	mux.HandleFunc("/api/evidence/reports", a.handleEvidenceReports)
	mux.HandleFunc("/api/evidence/reports/", a.handleEvidenceReportByID)
	mux.HandleFunc("/api/provider-smokes", a.handleProviderSmokeRecords)
	mux.HandleFunc("/api/provider-smokes/matrix", a.handleProviderSmokeMatrix)
	mux.HandleFunc("/api/provider-smokes/summary", a.handleProviderSmokeSummary)
	mux.HandleFunc("/api/provider-smokes/", a.handleProviderSmokeRecordByID)
	mux.HandleFunc("/api/status/providers", a.handleProviderStatuses)
	return a.loggingMiddleware(mux)
}

func (a *App) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(a.webIndex)
}

func (a *App) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	writeOK(w, http.StatusOK, map[string]string{
		"status": "ok",
		"env":    a.cfg.Env,
	})
}

func (a *App) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
		return
	}
	if req.Password != a.cfg.AdminPassword {
		writeError(w, http.StatusUnauthorized, "invalid_password", "Invalid password.")
		return
	}
	writeOK(w, http.StatusOK, map[string]string{
		"message": "Login validated for scaffold mode.",
	})
}

func (a *App) handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	writeOK(w, http.StatusOK, map[string]interface{}{
		"items": a.providers.List(),
	})
}

func (a *App) handleProviderByKey(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/providers/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}

	item, ok := a.providers.Get(parts[0])
	if !ok {
		writeError(w, http.StatusNotFound, "provider_not_found", "Provider was not found.")
		return
	}

	switch {
	case len(parts) == 2 && parts[1] == "capabilities" && r.Method == http.MethodGet:
		writeOK(w, http.StatusOK, map[string]interface{}{
			"provider":     item.Meta,
			"capabilities": item.Capability,
		})
	case len(parts) == 2 && parts[1] == "list" && r.Method == http.MethodPost:
		var req providerListRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
			return
		}
		profile, err := a.resolveProviderProfile(r.Context(), req.ProfileID)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		result := item.Adapter.List(provider.ListRequest{
			Profile:  profile,
			Path:     req.Path,
			ParentID: req.ParentID,
			PageSize: req.PageSize,
		})
		if !result.OK {
			handleServiceError(w, errors.New(result.Status))
			return
		}
		writeOK(w, http.StatusOK, result)
	case len(parts) == 2 && parts[1] == "metadata" && r.Method == http.MethodPost:
		var req providerMetadataRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
			return
		}
		profile, err := a.resolveProviderProfile(r.Context(), req.ProfileID)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		result := item.Adapter.Metadata(provider.MetadataRequest{
			Profile:  profile,
			Path:     req.Path,
			FileID:   req.FileID,
			ParentID: req.ParentID,
		})
		if !result.OK {
			handleServiceError(w, errors.New(result.Status))
			return
		}
		writeOK(w, http.StatusOK, result)
	case len(parts) == 2 && parts[1] == "create_dir" && r.Method == http.MethodPost:
		var req providerCreateDirRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
			return
		}
		profile, err := a.resolveProviderProfile(r.Context(), req.ProfileID)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		result := item.Adapter.CreateDir(provider.CreateDirRequest{
			Profile:  profile,
			ParentID: req.ParentID,
			DirName:  req.DirName,
		})
		if !result.OK {
			handleServiceError(w, errors.New(result.Status))
			return
		}
		writeOK(w, http.StatusOK, result)
	case len(parts) == 2 && parts[1] == "fast_check" && r.Method == http.MethodPost:
		var req providerFastCheckRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
			return
		}
		profile, err := a.resolveProviderProfile(r.Context(), req.ProfileID)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		result := item.Adapter.FastUploadCheck(provider.FastUploadCheckRequest{
			Profile:  profile,
			Path:     req.Path,
			ParentID: req.ParentID,
			Name:     req.Name,
			Size:     req.Size,
			MD5:      req.MD5,
			SHA1:     req.SHA1,
			GCID:     req.GCID,
		})
		if !result.OK {
			handleServiceError(w, errors.New(result.Status))
			return
		}
		writeOK(w, http.StatusOK, result)
	case len(parts) == 2 && parts[1] == "upload" && r.Method == http.MethodPost:
		var req providerUploadRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
			return
		}
		profile, err := a.resolveProviderProfile(r.Context(), req.ProfileID)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		result := item.Adapter.Upload(provider.UploadRequest{
			Profile:        profile,
			Path:           req.Path,
			ParentID:       req.ParentID,
			Name:           req.Name,
			Size:           req.Size,
			LocalPath:      req.LocalPath,
			ConflictPolicy: provider.ConflictPolicy(req.ConflictPolicy),
			Strategy:       req.Strategy,
			MD5:            req.MD5,
			SHA1:           req.SHA1,
			GCID:           req.GCID,
		})
		if !result.OK {
			handleServiceError(w, errors.New(result.Status))
			return
		}
		writeOK(w, http.StatusOK, result)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	}
}

func (a *App) handleAuthProfiles(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.auth.ListProfiles(r.Context())
		if err != nil {
			handleError(w, err)
			return
		}
		writeOK(w, http.StatusOK, map[string]interface{}{"items": items})
	case http.MethodPost:
		var req auth.CreateProfileInput
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
			return
		}
		item, err := a.auth.CreateProfile(r.Context(), req)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		writeOK(w, http.StatusCreated, item)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	}
}

func (a *App) handleAuthProfileByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/auth/profiles/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}
	id := parts[0]

	if len(parts) == 2 && parts[1] == "validate" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
			return
		}
		item, ok, err := a.auth.ValidateProfile(r.Context(), id)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "profile_not_found", "Auth profile was not found.")
			return
		}
		writeOK(w, http.StatusOK, item)
		return
	}

	switch r.Method {
	case http.MethodGet:
		item, ok, err := a.auth.GetProfile(r.Context(), id)
		if err != nil {
			handleError(w, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "profile_not_found", "Auth profile was not found.")
			return
		}
		writeOK(w, http.StatusOK, item)
	case http.MethodPatch:
		var req auth.UpdateProfileInput
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
			return
		}
		item, ok, err := a.auth.UpdateProfile(r.Context(), id, req)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "profile_not_found", "Auth profile was not found.")
			return
		}
		writeOK(w, http.StatusOK, item)
	case http.MethodDelete:
		ok, err := a.auth.DeleteProfile(r.Context(), id)
		if err != nil {
			handleError(w, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "profile_not_found", "Auth profile was not found.")
			return
		}
		writeOK(w, http.StatusOK, map[string]string{"deleted": id})
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	}
}

func (a *App) handlePlanPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	var req planner.PreviewRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
		return
	}
	item, err := planner.BuildPreview(a.providers, req)
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeOK(w, http.StatusOK, item)
}

func (a *App) handleTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := a.tasks.List(r.Context())
		if err != nil {
			handleError(w, err)
			return
		}
		writeOK(w, http.StatusOK, map[string]interface{}{"items": items})
	case http.MethodPost:
		var req task.CreateRequest
		if err := decodeJSON(r, &req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
			return
		}
		item, err := a.tasks.Create(r.Context(), req)
		if err != nil {
			handleServiceError(w, err)
			return
		}
		writeOK(w, http.StatusCreated, item)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
	}
}

func (a *App) handleTaskByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/tasks/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}
	id := parts[0]
	if len(parts) == 1 && r.Method == http.MethodGet {
		item, ok, err := a.tasks.Get(r.Context(), id)
		if err != nil {
			handleError(w, err)
			return
		}
		if !ok {
			writeError(w, http.StatusNotFound, "task_not_found", "Task was not found.")
			return
		}
		writeOK(w, http.StatusOK, item)
		return
	}
	if len(parts) != 2 || r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}

	var (
		item task.Detail
		ok   bool
		err  error
	)
	switch parts[1] {
	case "run":
		item, ok, err = a.tasks.Run(r.Context(), id)
	case "pause":
		item, ok, err = a.tasks.Pause(r.Context(), id)
	case "resume":
		item, ok, err = a.tasks.Resume(r.Context(), id)
	case "retry":
		req := retryTaskRequest{}
		if body, readErr := io.ReadAll(r.Body); readErr != nil {
			handleError(w, readErr)
			return
		} else if len(strings.TrimSpace(string(body))) > 0 {
			if err := decodeJSONFromBytes(body, &req); err != nil {
				writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
				return
			}
		}
		item, ok, err = a.tasks.RetryWithOptions(r.Context(), id, task.RetryOptions{
			Paths: req.Paths,
			Scope: req.Scope,
		})
	default:
		writeError(w, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}
	if err != nil {
		handleServiceError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "task_not_found", "Task was not found.")
		return
	}
	writeOK(w, http.StatusOK, item)
}

func (a *App) handleRuntimeEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	item, err := a.tasks.RuntimeEvidence(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	writeOK(w, http.StatusOK, item)
}

func (a *App) handleEvidenceReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	report, err := a.tasks.EvidenceReport(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	if r.URL.Query().Get("format") == "markdown" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(report.Markdown))
		return
	}
	if r.Method == http.MethodPost {
		var req evidenceReportRequest
		if r.Body != nil {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				handleError(w, err)
				return
			}
			if len(strings.TrimSpace(string(body))) > 0 {
				if err := decodeJSONFromBytes(body, &req); err != nil {
					writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
					return
				}
			}
		}
		record, err := a.tasks.SaveEvidenceReport(r.Context(), req.Title, req.Note)
		if err != nil {
			handleError(w, err)
			return
		}
		writeOK(w, http.StatusOK, record)
		return
	}
	writeOK(w, http.StatusOK, report)
}

func (a *App) handleEvidenceReports(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	items, err := a.tasks.ListEvidenceReports(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (a *App) handleEvidenceReportByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/evidence/reports/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}
	record, ok, err := a.tasks.GetEvidenceReport(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "report_not_found", "Report was not found.")
		return
	}
	if r.URL.Query().Get("format") == "markdown" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(record.Markdown))
		return
	}
	writeOK(w, http.StatusOK, record)
}

func (a *App) handleProviderSmokeRecords(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		items, err := a.tasks.ListProviderSmokeRecords(r.Context())
		if err != nil {
			handleError(w, err)
			return
		}
		writeOK(w, http.StatusOK, map[string]interface{}{"items": items})
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	var req providerSmokeRecordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Invalid JSON payload.")
		return
	}
	record, err := a.tasks.SaveProviderSmokeRecord(r.Context(), task.ProviderSmokeRecord{
		ProviderKey:   req.ProviderKey,
		ProtocolGroup: req.ProtocolGroup,
		AuthMode:      req.AuthMode,
		Category:      req.Category,
		Result:        req.Result,
		Title:         req.Title,
		Note:          req.Note,
		Operations:    req.Operations,
		Environment:   req.Environment,
	})
	if err != nil {
		handleServiceError(w, err)
		return
	}
	writeOK(w, http.StatusOK, record)
}

func (a *App) handleProviderSmokeSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	items, err := a.tasks.ProviderSmokeSummary(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (a *App) handleProviderSmokeMatrix(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	items, err := a.tasks.ProviderSmokeMatrix(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]interface{}{"items": items})
}

func (a *App) handleProviderSmokeRecordByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/provider-smokes/")
	id = strings.Trim(id, "/")
	if id == "" {
		writeError(w, http.StatusNotFound, "not_found", "Resource not found.")
		return
	}
	record, ok, err := a.tasks.GetProviderSmokeRecord(r.Context(), id)
	if err != nil {
		handleError(w, err)
		return
	}
	if !ok {
		writeError(w, http.StatusNotFound, "provider_smoke_not_found", "Provider smoke record was not found.")
		return
	}
	if r.URL.Query().Get("format") == "markdown" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(record.Markdown))
		return
	}
	writeOK(w, http.StatusOK, record)
}

func (a *App) handleProviderStatuses(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed.")
		return
	}
	items, err := a.tasks.ProviderStatuses(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}
	writeOK(w, http.StatusOK, map[string]interface{}{"items": items})
}

func handleServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, planner.ErrTargetProviderNotFound):
		writeError(w, http.StatusBadRequest, "target_provider_not_found", "Target provider was not found.")
	case errors.Is(err, planner.ErrInvalidExecutionMode):
		writeError(w, http.StatusBadRequest, "invalid_execution_mode", "Execution mode is not supported.")
	case err != nil && err.Error() == "provider_not_found":
		writeError(w, http.StatusBadRequest, "provider_not_found", "Provider was not found.")
	case err != nil && err.Error() == "auth_mode_not_supported":
		writeError(w, http.StatusBadRequest, "auth_mode_not_supported", "Auth mode is not supported by the provider.")
	case err != nil && err.Error() == "display_name_required":
		writeError(w, http.StatusBadRequest, "display_name_required", "Display name is required.")
	case err != nil && err.Error() == "provider_key_required":
		writeError(w, http.StatusBadRequest, "provider_key_required", "Provider key is required.")
	case err != nil && err.Error() == "task_not_runnable":
		writeError(w, http.StatusBadRequest, "task_not_runnable", "Task cannot be run from the current state.")
	case err != nil && err.Error() == "task_state_transition_not_allowed":
		writeError(w, http.StatusBadRequest, "task_state_transition_not_allowed", "Task state transition is not allowed.")
	case err != nil && err.Error() == "target_profile_not_found":
		writeError(w, http.StatusBadRequest, "target_profile_not_found", "Target profile was not found.")
	case err != nil && err.Error() == "source_profile_not_found":
		writeError(w, http.StatusBadRequest, "source_profile_not_found", "Source profile was not found.")
	case err != nil && err.Error() == "source_profile_required_for_lazy_scan":
		writeError(w, http.StatusBadRequest, "source_profile_required_for_lazy_scan", "Lazy scan mode requires a source profile.")
	case err != nil && err.Error() == "missing_access_token":
		writeError(w, http.StatusBadRequest, "missing_access_token", "Provider token is required.")
	case err != nil && err.Error() == "missing_domain_or_drive_id":
		writeError(w, http.StatusBadRequest, "missing_domain_or_drive_id", "Provider requires domainId and driveId.")
	case err != nil && err.Error() == "missing_cookie":
		writeError(w, http.StatusBadRequest, "missing_cookie", "Provider cookie is required.")
	case err != nil && err.Error() == "missing_pwd_id":
		writeError(w, http.StatusBadRequest, "missing_pwd_id", "Provider requires pwdId.")
	case err != nil && err.Error() == "missing_access_token_or_cookie":
		writeError(w, http.StatusBadRequest, "missing_access_token_or_cookie", "Provider requires token or cookie.")
	case err != nil && err.Error() == "missing_md5":
		writeError(w, http.StatusBadRequest, "missing_md5", "Fast upload requires md5.")
	case err != nil && err.Error() == "missing_sha1":
		writeError(w, http.StatusBadRequest, "missing_sha1", "Fast upload requires sha1.")
	case err != nil && err.Error() == "missing_gcid":
		writeError(w, http.StatusBadRequest, "missing_gcid", "Fast upload requires gcid.")
	case err != nil && err.Error() == "pending_manual_requires_confirmation":
		writeError(w, http.StatusBadRequest, "pending_manual_requires_confirmation", "Pending-manual items cannot run until fallback runtime is implemented.")
	case err != nil && strings.HasPrefix(err.Error(), "retry_cooldown_active:"):
		writeError(w, http.StatusBadRequest, "retry_cooldown_active", "Retry queue is still cooling down for a rate-limited item.")
	case err != nil && strings.HasPrefix(err.Error(), "retry_blocked:"):
		writeError(w, http.StatusBadRequest, "retry_blocked", "Retry queue is blocked and still requires manual intervention.")
	case err != nil && err.Error() == "retry_selection_empty":
		writeError(w, http.StatusBadRequest, "retry_selection_empty", "The selected retry paths do not contain runnable pending or retryable items.")
	default:
		handleError(w, err)
	}
}

func (a *App) resolveProviderProfile(ctx context.Context, profileID string) (provider.AuthProfile, error) {
	if strings.TrimSpace(profileID) == "" {
		return provider.AuthProfile{}, nil
	}
	profile, ok, err := a.auth.GetProfile(ctx, profileID)
	if err != nil {
		return provider.AuthProfile{}, err
	}
	if !ok {
		return provider.AuthProfile{}, errors.New("target_profile_not_found")
	}
	return provider.AuthProfile{
		ID:          profile.ID,
		ProviderKey: profile.ProviderKey,
		AuthMode:    profile.AuthMode,
		DisplayName: profile.DisplayName,
		Token:       profile.Token,
		Cookie:      profile.Cookie,
		Extra:       profile.Extra,
	}, nil
}
