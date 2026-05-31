package provider

import "testing"

func TestDefaultCatalogOverwriteContracts(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	tests := []struct {
		key                   string
		wantSupportsOverwrite bool
		wantSupportsAutoRename bool
		wantOverwriteBehavior string
		wantConflictPolicies  []ConflictPolicy
	}{
		{
			key:                    "aliyundrive_open",
			wantSupportsOverwrite:  true,
			wantSupportsAutoRename: true,
			wantOverwriteBehavior:  "provider_managed",
			wantConflictPolicies:   []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew},
		},
		{
			key:                    "123_open",
			wantSupportsOverwrite:  false,
			wantSupportsAutoRename: true,
			wantOverwriteBehavior:  "downgrade_to_auto_rename",
			wantConflictPolicies:   []ConflictPolicy{ConflictPolicyOverwriteExisting, ConflictPolicyAutoRenameNew},
		},
		{
			key:                    "189cloud",
			wantSupportsOverwrite:  false,
			wantSupportsAutoRename: false,
			wantOverwriteBehavior:  "readonly_auth_blocked",
			wantConflictPolicies:   []ConflictPolicy{ConflictPolicyAutoRenameNew},
		},
		{
			key:                    "115_open",
			wantSupportsOverwrite:  false,
			wantSupportsAutoRename: false,
			wantOverwriteBehavior:  "not_implemented",
			wantConflictPolicies:   []ConflictPolicy{ConflictPolicyAutoRenameNew},
		},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			entry, ok := registry.Get(tt.key)
			if !ok {
				t.Fatalf("expected provider %s to exist", tt.key)
			}
			if entry.Meta.SupportsOverwrite != tt.wantSupportsOverwrite {
				t.Fatalf("expected SupportsOverwrite %v for %s, got %+v", tt.wantSupportsOverwrite, tt.key, entry.Meta)
			}
			if entry.Meta.SupportsAutoRename != tt.wantSupportsAutoRename {
				t.Fatalf("expected SupportsAutoRename %v for %s, got %+v", tt.wantSupportsAutoRename, tt.key, entry.Meta)
			}
			if entry.Meta.OverwriteBehavior != tt.wantOverwriteBehavior {
				t.Fatalf("expected OverwriteBehavior %s for %s, got %+v", tt.wantOverwriteBehavior, tt.key, entry.Meta)
			}
			if !equalConflictPolicies(entry.Meta.ConflictPolicies, tt.wantConflictPolicies) {
				t.Fatalf("expected conflict policies %#v for %s, got %#v", tt.wantConflictPolicies, tt.key, entry.Meta.ConflictPolicies)
			}
		})
	}
}

func equalConflictPolicies(left []ConflictPolicy, right []ConflictPolicy) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
