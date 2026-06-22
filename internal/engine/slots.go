package engine

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/schaemi85/spire/internal/manifest"
	"github.com/schaemi85/spire/internal/tools"
)

// ResolveSlots iterates through the given slots, prompting or computing each value.
// Resolved values are stored in both the slot's Value field and rc.Slots.
func ResolveSlots(slots []manifest.Slot, rc *ResolveContext, nonInteractive bool) error {
	for i := range slots {
		slot := &slots[i]

		if v, ok := rc.Slots[slot.Key]; ok && v != "" {
			slot.Value = v
			continue
		}

		var value string
		var err error

		switch slot.Type {
		case manifest.PromptMandatory:
			if nonInteractive {
				return fmt.Errorf("slot %q requires a value; use --set %s=<value>", slot.Key, slot.Key)
			}
			value, err = promptMandatorySlot(slot)
		case manifest.PromptOptional:
			if nonInteractive {
				value = slot.DefaultValue
			} else {
				value, err = promptOptionalSlot(slot)
			}
		case manifest.PromptSecret:
			if nonInteractive {
				return fmt.Errorf("slot %q requires a secret value; use --set %s=<value>", slot.Key, slot.Key)
			}
			value, err = promptSecretSlot(slot)
		case manifest.DynamicValue:
			value, err = EvaluateExpression(slot.Expression, rc)
		default:
			return fmt.Errorf("unknown slot type %d for slot %q", slot.Type, slot.Key)
		}

		if err != nil {
			return fmt.Errorf("failed to resolve slot %q: %w", slot.Key, err)
		}

		value, err = applyPipelines(value, slot.Pipelines, rc)
		if err != nil {
			return fmt.Errorf("failed to apply pipelines for slot %q: %w", slot.Key, err)
		}

		slot.Value = value
		rc.Slots[slot.Key] = value
	}
	return nil
}

// ResolveDynamicSlots re-evaluates only DynamicValue slots.
func ResolveDynamicSlots(slots []manifest.Slot, rc *ResolveContext) error {
	for i := range slots {
		slot := &slots[i]
		if slot.Type != manifest.DynamicValue {
			continue
		}
		value, err := EvaluateExpression(slot.Expression, rc)
		if err != nil {
			return fmt.Errorf("failed to re-evaluate expression for slot %q: %w", slot.Key, err)
		}
		value, err = applyPipelines(value, slot.Pipelines, rc)
		if err != nil {
			return fmt.Errorf("failed to apply pipelines for slot %q: %w", slot.Key, err)
		}
		slot.Value = value
		rc.Slots[slot.Key] = value
	}
	return nil
}

// EvaluateExpression renders a Go template expression using the resolve context
// and Spire's custom delimiters ([[ ]]).
func EvaluateExpression(expr string, rc *ResolveContext) (string, error) {
	if expr == "" {
		return "", nil
	}
	tmpl, err := template.New("expr").Delims(SpireDelimLeft, SpireDelimRight).Funcs(PipelineFuncs()).Parse(expr)
	if err != nil {
		return "", fmt.Errorf("invalid expression %q: %w", expr, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, rc.templateData()); err != nil {
		return "", fmt.Errorf("failed to execute expression %q: %w", expr, err)
	}
	return strings.TrimSpace(buf.String()), nil
}

func applyPipelines(value string, pipelines []string, rc *ResolveContext) (string, error) {
	if len(pipelines) == 0 {
		return value, nil
	}
	chain := ".Value"
	for _, p := range pipelines {
		chain += " | " + p
	}
	expr := "{{ " + chain + " }}"
	tmpl, err := template.New("pipeline").Funcs(PipelineFuncs()).Parse(expr)
	if err != nil {
		return "", fmt.Errorf("invalid pipeline chain %q: %w", expr, err)
	}
	data := map[string]interface{}{
		"Value": value,
		"slots": rc.Slots,
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute pipeline chain: %w", err)
	}
	return buf.String(), nil
}

func promptMandatorySlot(slot *manifest.Slot) (string, error) {
	label := slot.Label
	if label == "" {
		label = slot.Key
	}
	if slot.Description != "" {
		fmt.Printf("  %s\n", slot.Description)
	}
	validator, err := ParseValidationRule(slot.Validation)
	if err != nil {
		return "", fmt.Errorf("slot %q: %w", slot.Key, err)
	}
	return tools.MandatoryPrompt(fmt.Sprintf("%s: ", label), validator), nil
}

func promptOptionalSlot(slot *manifest.Slot) (string, error) {
	label := slot.Label
	if label == "" {
		label = slot.Key
	}
	prompt := label
	if slot.DefaultValue != "" {
		prompt += fmt.Sprintf(" (press Enter for default: %s)", slot.DefaultValue)
	}
	prompt += ": "
	if slot.Description != "" {
		fmt.Printf("  %s\n", slot.Description)
	}
	validator, err := ParseValidationRule(slot.Validation)
	if err != nil {
		return "", fmt.Errorf("slot %q: %w", slot.Key, err)
	}
	return tools.OptionalPrompt(prompt, slot.DefaultValue, validator), nil
}

func promptSecretSlot(slot *manifest.Slot) (string, error) {
	label := slot.Label
	if label == "" {
		label = slot.Key
	}
	if slot.Description != "" {
		fmt.Printf("  %s\n", slot.Description)
	}
	validator, err := ParseValidationRule(slot.Validation)
	if err != nil {
		return "", fmt.Errorf("slot %q: %w", slot.Key, err)
	}
	return tools.SecretPrompt(fmt.Sprintf("%s: ", label), validator), nil
}
