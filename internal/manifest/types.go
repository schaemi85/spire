package manifest

import "fmt"

// SpireManifest is the central manifest for a Spire project.
type SpireManifest struct {
	SpireVersion    string         `yaml:"spireVersion"`
	TemplateVersion string         `yaml:"templateVersion"`
	GitRepository   string         `yaml:"gitRepository,omitempty"`
	AppSlots        []Slot         `yaml:"appSlots,omitempty"`
	Services        []Service      `yaml:"services,omitempty"`
	TemplateFiles   []TemplateFile `yaml:"templateFiles,omitempty"`
	PathRenames     []PathRename   `yaml:"pathRenames,omitempty"`
	IgnorePaths     []string       `yaml:"ignorePaths,omitempty"`
	ServiceConfig   ServiceConfig  `yaml:"serviceConfig,omitempty"`
}

// Slot represents a single configurable placeholder in a template.
type Slot struct {
	Key          string   `yaml:"key"`
	Label        string   `yaml:"label,omitempty"`
	Description  string   `yaml:"description,omitempty"`
	Type         SlotType `yaml:"type"`
	DefaultValue string   `yaml:"defaultValue,omitempty"`
	Expression   string   `yaml:"expression,omitempty"`
	Pipelines    []string `yaml:"pipelines,omitempty"`
	Validation   string   `yaml:"validation,omitempty"`
	Value        string   `yaml:"value,omitempty"`
}

// Service represents a generated service within the project.
type Service struct {
	Name     string `yaml:"name"`
	SlugName string `yaml:"slugName"`
	Slots    []Slot `yaml:"slots,omitempty"`
}

// ServiceConfig is the blueprint used when generating new services.
type ServiceConfig struct {
	OriginalPath  string       `yaml:"originalPath"`
	ServicesSlots []Slot       `yaml:"servicesSlots,omitempty"`
	PathRenames   []PathRename `yaml:"pathRenames,omitempty"`
	PostHooks     []PostHook   `yaml:"postHooks,omitempty"`
}

// PostHook defines post-generation operations with an optional condition.
type PostHook struct {
	Condition   string       `yaml:"condition,omitempty"`
	RemovePaths []string     `yaml:"removePaths,omitempty"`
	PathRenames []PathRename `yaml:"pathRenames,omitempty"`
}

// TemplateFile defines a Go template file to render during generation.
type TemplateFile struct {
	Source                    string `yaml:"source"`
	Destination               string `yaml:"destination"`
	RegenerateOnServiceChange bool   `yaml:"regenerateOnServiceChange,omitempty"`
}

// PathRename defines a file/directory rename rule driven by slot values.
type PathRename struct {
	Pattern    string `yaml:"pattern"`
	Expression string `yaml:"expression"`
}

// SlotType determines how a slot value is collected during generation.
type SlotType int

const (
	PromptOptional  SlotType = iota // user is prompted, may leave empty
	PromptMandatory                 // user is prompted, value required
	PromptSecret                    // user is prompted, input is masked
	DynamicValue                    // value is computed from Expression
)

var (
	slotTypeToString = map[SlotType]string{
		PromptOptional:  "PromptOptional",
		PromptMandatory: "PromptMandatory",
		PromptSecret:    "PromptSecret",
		DynamicValue:    "DynamicValue",
	}
	stringToSlotType = map[string]SlotType{
		"PromptOptional":  PromptOptional,
		"PromptMandatory": PromptMandatory,
		"PromptSecret":    PromptSecret,
		"DynamicValue":    DynamicValue,
	}
)

func (st SlotType) String() string {
	if s, ok := slotTypeToString[st]; ok {
		return s
	}
	return "Unknown"
}

func (st SlotType) MarshalYAML() (interface{}, error) {
	if s, ok := slotTypeToString[st]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("unknown SlotType: %d", st)
}

func (st *SlotType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if v, ok := stringToSlotType[s]; ok {
		*st = v
		return nil
	}
	return fmt.Errorf("unknown SlotType: %q", s)
}

// GetSlotValue returns the resolved value of a slot by key, or empty string if not found.
func (m *SpireManifest) GetSlotValue(key string) string {
	for _, s := range m.AppSlots {
		if s.Key == key {
			return s.Value
		}
	}
	return ""
}

// SetSlotValue sets the value of an app-level slot by key. Returns false if not found.
func (m *SpireManifest) SetSlotValue(key, value string) bool {
	for i, s := range m.AppSlots {
		if s.Key == key {
			m.AppSlots[i].Value = value
			return true
		}
	}
	return false
}

// GetServiceSlotValue returns the resolved value of a slot within a specific service.
func (m *SpireManifest) GetServiceSlotValue(serviceName, key string) string {
	for _, svc := range m.Services {
		if svc.Name == key || svc.SlugName == serviceName {
			for _, s := range svc.Slots {
				if s.Key == key {
					return s.Value
				}
			}
		}
	}
	return ""
}
