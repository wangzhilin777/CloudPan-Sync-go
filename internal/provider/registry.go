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
		StaticAdapter{MetaInfo: Provider{Key: "guangya", DisplayName: "Guangya", ProtocolGroup: "guangya", AuthModes: []string{"web_login_capture", "manual_token"}, FastUploadInputs: []string{"md5", "size", "name"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, CapabilityInfo: fullCapability},
		NewOpenFamilyAdapter(Provider{Key: "aliyundrive_open", DisplayName: "Aliyun Drive Open", ProtocolGroup: "aliyun_123_open", AuthModes: []string{"official_oauth"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: true, SupportsAutoRename: true, OverwriteBehavior: "provider_managed", Status: "planned"}, fullCapability, true),
		NewPan115FamilyAdapter(Provider{Key: "115_open", DisplayName: "115 Open", ProtocolGroup: "115_open", AuthModes: []string{"official_oauth", "manual_cookie"}, FastUploadInputs: []string{"sha1", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: false, OverwriteBehavior: "not_implemented", Status: "planned"}, fullCapability),
		NewShareFamilyAdapter(Provider{Key: "quark", DisplayName: "Quark", ProtocolGroup: "quark_uc", AuthModes: []string{"web_login_capture", "manual_cookie"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability, true),
		StaticAdapter{MetaInfo: Provider{Key: "189cloud", DisplayName: "Tianyi 189Cloud", ProtocolGroup: "189cloud", AuthModes: []string{"web_login_capture", "manual_cookie"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: false, OverwriteBehavior: "readonly_auth_blocked", Status: "planned"}, CapabilityInfo: fullCapability},
		NewBaiduFamilyAdapter(Provider{Key: "baidu_netdisk", DisplayName: "Baidu Netdisk", ProtocolGroup: "baidu_netdisk", AuthModes: []string{"official_oauth", "manual_cookie"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability),
		NewShareFamilyAdapter(Provider{Key: "uc", DisplayName: "UC Drive", ProtocolGroup: "quark_uc", AuthModes: []string{"web_login_capture", "manual_cookie"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability, true),
		NewHashFamilyAdapter(Provider{Key: "xunlei", DisplayName: "Xunlei Drive", ProtocolGroup: "xunlei_pikpak", AuthModes: []string{"web_login_capture", "manual_token"}, FastUploadInputs: []string{"gcid", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability),
		NewHashFamilyAdapter(Provider{Key: "pikpak", DisplayName: "PikPak", ProtocolGroup: "xunlei_pikpak", AuthModes: []string{"manual_token"}, FastUploadInputs: []string{"gcid", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability),
		NewOpenFamilyAdapter(Provider{Key: "123_open", DisplayName: "123Pan Open", ProtocolGroup: "aliyun_123_open", AuthModes: []string{"official_oauth", "manual_token"}, FastUploadInputs: []string{"md5", "size"}, FallbackModes: []string{"download_upload"}, ConflictPolicies: []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew}, SupportsOverwrite: false, SupportsAutoRename: true, OverwriteBehavior: "downgrade_to_auto_rename", Status: "planned"}, fullCapability, false),
	}
}
