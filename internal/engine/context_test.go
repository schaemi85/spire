package engine

import (
	"testing"

	"github.com/schaemi85/spire/internal/manifest"
)

func TestClearSecretSlotValues(t *testing.T) {
	slots := []manifest.Slot{
		{Key: "Name", Type: manifest.PromptMandatory, Value: "visible"},
		{Key: "Password", Type: manifest.PromptSecret, Value: "s3cret"},
		{Key: "Slug", Type: manifest.DynamicValue, Value: "computed"},
		{Key: "Token", Type: manifest.PromptSecret, Value: "tok3n"},
	}

	ClearSecretSlotValues(slots)

	if slots[0].Value != "visible" {
		t.Errorf("non-secret slot value cleared: %q", slots[0].Value)
	}
	if slots[1].Value != "" {
		t.Errorf("secret slot not cleared: %q", slots[1].Value)
	}
	if slots[2].Value != "computed" {
		t.Errorf("dynamic slot value cleared: %q", slots[2].Value)
	}
	if slots[3].Value != "" {
		t.Errorf("secret slot not cleared: %q", slots[3].Value)
	}
}

func TestBuildResolveContextFromManifest(t *testing.T) {
	m := &manifest.SpireManifest{
		AppSlots: []manifest.Slot{
			{Key: "Name", Value: "my-app"},
			{Key: "Empty", Value: ""},
			{Key: "Version", Value: "1.0.0"},
		},
		Services: []manifest.Service{
			{Name: "svc1", SlugName: "svc1"},
		},
	}

	rc := BuildResolveContextFromManifest(m)

	if rc.Slots["Name"] != "my-app" {
		t.Errorf("Slots[Name] = %q", rc.Slots["Name"])
	}
	if _, ok := rc.Slots["Empty"]; ok {
		t.Error("empty slot should not be in resolve context")
	}
	if rc.Slots["Version"] != "1.0.0" {
		t.Errorf("Slots[Version] = %q", rc.Slots["Version"])
	}
	if len(rc.Services) != 1 {
		t.Errorf("Services length = %d, want 1", len(rc.Services))
	}
}

func TestPopulateManifestFromSlots(t *testing.T) {
	m := &manifest.SpireManifest{}
	rc := NewResolveContext()
	rc.Slots["GitRepository"] = "https://github.com/example/repo.git"

	PopulateManifestFromSlots(m, rc)

	if m.GitRepository != "https://github.com/example/repo.git" {
		t.Errorf("GitRepository = %q", m.GitRepository)
	}
}

func TestPopulateManifestFromSlotsNoKey(t *testing.T) {
	m := &manifest.SpireManifest{GitRepository: "original"}
	rc := NewResolveContext()

	PopulateManifestFromSlots(m, rc)

	if m.GitRepository != "original" {
		t.Errorf("GitRepository changed to %q, should stay %q", m.GitRepository, "original")
	}
}
