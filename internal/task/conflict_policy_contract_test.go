package task

import (
	"testing"

	"cloudpan-sync-go/internal/provider"
)

func TestResolveConflictPolicyContracts(t *testing.T) {
	tests := []struct {
		name          string
		meta          provider.Provider
		requested     provider.ConflictPolicy
		wantPolicy    provider.ConflictPolicy
		wantAction    string
	}{
		{
			name:       "empty policy defaults to auto rename",
			meta:       provider.Provider{Key: "demo", SupportsOverwrite: true, SupportsAutoRename: true},
			requested:  "",
			wantPolicy: provider.ConflictPolicyAutoRenameNew,
			wantAction: "",
		},
		{
			name:       "overwrite stays when provider supports it",
			meta:       provider.Provider{Key: "demo", SupportsOverwrite: true, SupportsAutoRename: true},
			requested:  provider.ConflictPolicyOverwriteExisting,
			wantPolicy: provider.ConflictPolicyOverwriteExisting,
			wantAction: "",
		},
		{
			name:       "overwrite downgrades to auto rename when provider lacks overwrite but supports auto rename",
			meta:       provider.Provider{Key: "demo", SupportsOverwrite: false, SupportsAutoRename: true},
			requested:  provider.ConflictPolicyOverwriteExisting,
			wantPolicy: provider.ConflictPolicyAutoRenameNew,
			wantAction: "downgrade_to_auto_rename",
		},
		{
			name:       "overwrite stays requested when provider supports neither overwrite nor auto rename",
			meta:       provider.Provider{Key: "demo", SupportsOverwrite: false, SupportsAutoRename: false},
			requested:  provider.ConflictPolicyOverwriteExisting,
			wantPolicy: provider.ConflictPolicyOverwriteExisting,
			wantAction: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPolicy, gotAction := resolveConflictPolicy(tt.meta, tt.requested)
			if gotPolicy != tt.wantPolicy || gotAction != tt.wantAction {
				t.Fatalf("resolveConflictPolicy(%+v, %s) = (%s, %s), want (%s, %s)", tt.meta, tt.requested, gotPolicy, gotAction, tt.wantPolicy, tt.wantAction)
			}
		})
	}
}
