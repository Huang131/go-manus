package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
)

func TestDefaultPromptCatalogExposesVersionedDefinitions(t *testing.T) {
	catalog := DefaultPromptCatalog()
	wantNames := []PromptName{
		PromptCreatePlan,
		PromptExecution,
		PromptPlannerSystem,
		PromptReActSystem,
		PromptSummarize,
		PromptSystem,
		PromptUpdatePlan,
	}
	if catalog.Version() != "v1" {
		t.Fatalf("Version() = %q, want v1", catalog.Version())
	}
	if got := catalog.Names(); !reflect.DeepEqual(got, wantNames) {
		t.Fatalf("Names() = %v, want %v", got, wantNames)
	}
	for _, name := range wantNames {
		definition, ok := catalog.Get(name)
		if !ok {
			t.Fatalf("Get(%q) missing", name)
		}
		if definition.Name != name || definition.Version != catalog.Version() || definition.Content == "" {
			t.Fatalf("Get(%q) = %+v", name, definition)
		}
		digest := sha256.Sum256([]byte(definition.Content))
		if definition.Hash != hex.EncodeToString(digest[:]) {
			t.Fatalf("Get(%q).Hash = %q, want content SHA-256", name, definition.Hash)
		}
	}
	if catalog.Hash() == "" {
		t.Fatal("Hash() is empty")
	}
}

func TestPromptCatalogCopiesInputAndHashesDeterministically(t *testing.T) {
	contents := map[PromptName]string{PromptSystem: "system", PromptExecution: "execution"}
	first, err := NewPromptCatalog("test-v1", contents)
	if err != nil {
		t.Fatal(err)
	}
	contents[PromptSystem] = "mutated"
	second, err := NewPromptCatalog("test-v1", map[PromptName]string{
		PromptExecution: "execution",
		PromptSystem:    "system",
	})
	if err != nil {
		t.Fatal(err)
	}

	if first.Hash() != second.Hash() {
		t.Fatalf("catalog hash depends on map order: %q != %q", first.Hash(), second.Hash())
	}
	changed, err := NewPromptCatalog("test-v1", map[PromptName]string{
		PromptExecution: "changed",
		PromptSystem:    "system",
	})
	if err != nil {
		t.Fatal(err)
	}
	if changed.Hash() == first.Hash() {
		t.Fatal("catalog hash did not change with prompt content")
	}
	definition, _ := first.Get(PromptSystem)
	if definition.Content != "system" {
		t.Fatalf("catalog retained mutable input: %q", definition.Content)
	}
}

func TestNewPromptCatalogRejectsIncompleteDefinition(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		contents map[PromptName]string
	}{
		{name: "missing definitions", version: "v1", contents: map[PromptName]string{}},
		{name: "missing version", contents: map[PromptName]string{PromptSystem: "system"}},
		{name: "missing name", version: "v1", contents: map[PromptName]string{"": "system"}},
		{name: "missing content", version: "v1", contents: map[PromptName]string{PromptSystem: ""}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := NewPromptCatalog(tt.version, tt.contents); err == nil {
				t.Fatal("NewPromptCatalog() error = nil")
			}
		})
	}
}
