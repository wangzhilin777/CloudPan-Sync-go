package provider

import "sort"

type Entry struct {
	Meta       Provider      `json:"meta"`
	Capability CapabilitySet `json:"capability"`
	Adapter    Adapter       `json:"-"`
}

type Registry struct {
	items map[string]Entry
}

func NewRegistry(adapters ...Adapter) *Registry {
	items := make(map[string]Entry, len(adapters))
	for _, adapter := range adapters {
		meta := adapter.Meta()
		items[meta.Key] = Entry{
			Meta:       meta,
			Capability: adapter.Capabilities(),
			Adapter:    adapter,
		}
	}
	return &Registry{items: items}
}

func (r *Registry) List() []Entry {
	rows := make([]Entry, 0, len(r.items))
	for _, item := range r.items {
		rows = append(rows, item)
	}
	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Meta.Key < rows[j].Meta.Key
	})
	return rows
}

func (r *Registry) Get(key string) (Entry, bool) {
	item, ok := r.items[key]
	return item, ok
}

func DefaultCatalog() []Adapter {
	fullCapability := CapabilitySet{
		SupportsAuthValidation: true,
		SupportsList:           true,
		SupportsMetadata:       true,
		SupportsCreateDir:      true,
		SupportsFastUpload:     true,
		SupportsUpload:         true,
	}

	return []Adapter{
		NewGuangyaFamilyAdapter(Provider{Key: "guangya", DisplayName: "Guangya", ProtocolGroup: "guangya", RiskHints: []string{"Prefer low concurrency and monitor anti-abuse responses.", "Large-file fallback may require manual confirmation."}, RiskTraits: []string{"risk_sensitive", "manual_confirmation_possible", "multipart_resume"}, AuthModes: []string{"web_login_capture", "manual_token"}, FastUploadInputs: []string{"md5", "size", "name"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability),
		NewOpenFamilyAdapter(Provider{Key: "aliyundrive_open", DisplayName: "Aliyun Drive Open", ProtocolGroup: "aliyun_123_open", RiskHints: []string{"Open API flow is smoother for small batches but still needs watch on large fast runs."}, RiskTraits: []string{"open_api", "balanced_default", "domain_drive_required"}, AuthModes: []string{"official_oauth"}, FastUploadInputs: []string{"sha1", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: true, SupportsAutoRename: true, OverwriteBehavior: "provider_managed", Status: "planned"}, fullCapability, true),
		NewPan115FamilyAdapter(Provider{Key: "115_open", DisplayName: "115 Open", ProtocolGroup: "115_open", RiskHints: []string{"Keep list pacing conservative and expect manual confirmation on some pending-manual branches."}, RiskTraits: []string{"risk_sensitive", "manual_confirmation_possible", "oss_resume"}, AuthModes: []string{"official_oauth", "manual_cookie"}, FastUploadInputs: []string{"sha1", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: false, OverwriteBehavior: "not_implemented", Status: "planned"}, fullCapability),
		NewShareFamilyAdapter(Provider{Key: "quark", DisplayName: "Quark", ProtocolGroup: "quark_uc", RiskHints: []string{"Risk control is sensitive; smaller page size and slower subtree progression are safer."}, RiskTraits: []string{"risk_sensitive", "manual_confirmation_possible", "share_token_flow"}, AuthModes: []string{"web_login_capture", "manual_cookie"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability, true),
		NewCloud189FamilyAdapter(Provider{Key: "189cloud", DisplayName: "Tianyi 189Cloud", ProtocolGroup: "189cloud", RiskHints: []string{"Prefer conservative paging and directory pacing on Tianyi 189Cloud."}, RiskTraits: []string{"risk_sensitive", "manual_confirmation_possible", "upload_session_resume"}, AuthModes: []string{"web_login_capture", "manual_cookie"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: false, OverwriteBehavior: "readonly_auth_blocked", Status: "planned"}, fullCapability),
		NewBaiduFamilyAdapter(Provider{Key: "baidu_netdisk", DisplayName: "Baidu Netdisk", ProtocolGroup: "baidu_netdisk", RiskHints: []string{"Baidu Netdisk is more likely to hit captcha or risk-control when pacing is too aggressive."}, RiskTraits: []string{"risk_sensitive", "captcha_prone", "tmpfile_checkpoint"}, AuthModes: []string{"official_oauth", "manual_cookie"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability),
		NewShareFamilyAdapter(Provider{Key: "uc", DisplayName: "UC Drive", ProtocolGroup: "quark_uc", RiskHints: []string{"UC shares Quark-like risk sensitivity; keep list and retry pacing conservative."}, RiskTraits: []string{"risk_sensitive", "manual_confirmation_possible", "share_token_flow"}, AuthModes: []string{"web_login_capture", "manual_cookie"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability, true),
		NewHashFamilyAdapter(Provider{Key: "xunlei", DisplayName: "Xunlei Drive", ProtocolGroup: "xunlei_pikpak", RiskHints: []string{"Xunlei can keep moderate pace, but pending-manual fallback still needs caution."}, RiskTraits: []string{"gcid_fast_path", "manual_confirmation_possible", "moderate_risk"}, AuthModes: []string{"web_login_capture", "manual_token"}, FastUploadInputs: []string{"gcid", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability),
		NewHashFamilyAdapter(Provider{Key: "pikpak", DisplayName: "PikPak", ProtocolGroup: "xunlei_pikpak", RiskHints: []string{"PikPak allows moderate pacing, but large fallback runs should still avoid burst concurrency."}, RiskTraits: []string{"gcid_fast_path", "manual_confirmation_possible", "moderate_risk"}, AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"gcid", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability),
		NewOpenFamilyAdapter(Provider{Key: "123_open", DisplayName: "123Pan Open", ProtocolGroup: "aliyun_123_open", RiskHints: []string{"123Pan Open can use larger page budgets, but aggressive mode still needs retry-limit discipline."}, RiskTraits: []string{"open_api", "balanced_default", "manual_confirmation_possible"}, AuthModes: []string{"official_oauth", "manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability, false),
	}
}
