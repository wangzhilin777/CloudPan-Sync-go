package provider

import "testing"

func TestDefaultCatalogIncludesTenProviders(t *testing.T) {
	registry := NewRegistry(DefaultCatalog()...)
	items := registry.List()
	if len(items) != 10 {
		t.Fatalf("expected 10 providers, got %d", len(items))
	}

	if _, ok := registry.Get("guangya"); !ok {
		t.Fatal("expected guangya provider to exist")
	}
}
