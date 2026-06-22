package engine

import (
	"github.com/schaemi85/spire/internal/manifest"
)

// ResolveContext holds the state available during slot resolution and rendering.
type ResolveContext struct {
	Slots    map[string]string
	Services []manifest.Service
}

// NewResolveContext creates a ResolveContext with an empty Slots map.
func NewResolveContext() *ResolveContext {
	return &ResolveContext{
		Slots: make(map[string]string),
	}
}

// templateData returns the data map used for Go template evaluation.
func (rc *ResolveContext) templateData() map[string]interface{} {
	return map[string]interface{}{
		"slots":    rc.Slots,
		"services": rc.Services,
	}
}

// BuildResolveContextFromManifest creates a ResolveContext pre-populated with
// stored slot values from an existing manifest.
func BuildResolveContextFromManifest(m *manifest.SpireManifest) *ResolveContext {
	rc := NewResolveContext()
	for _, slot := range m.AppSlots {
		if slot.Value != "" {
			rc.Slots[slot.Key] = slot.Value
		}
	}
	rc.Services = m.Services
	return rc
}

// PopulateManifestFromSlots sets the GitRepository field on a SpireManifest
// from the resolved slot value if present.
func PopulateManifestFromSlots(m *manifest.SpireManifest, rc *ResolveContext) {
	if v, ok := rc.Slots["GitRepository"]; ok {
		m.GitRepository = v
	}
}

// ClearSecretSlotValues removes the Value from PromptSecret slots so secrets
// are never persisted to the manifest file on disk.
func ClearSecretSlotValues(slots []manifest.Slot) {
	for i := range slots {
		if slots[i].Type == manifest.PromptSecret {
			slots[i].Value = ""
		}
	}
}
