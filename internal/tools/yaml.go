package tools

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// updateOrRemoveNodePath traverses the YAML node tree and either updates or removes a key (and its subtree).
// If newValue is nil, the key is removed. If newValue is not nil, the key is updated (or created).
func updateOrRemoveNodePath(node *yaml.Node, keyPath []string, newValue *string) {
	if len(keyPath) == 0 {
		return
	}
	switch node.Kind {
	case yaml.DocumentNode:
		for _, n := range node.Content {
			updateOrRemoveNodePath(n, keyPath, newValue)
		}
	case yaml.MappingNode:
		for i := 0; i < len(node.Content); i += 2 {
			k := node.Content[i]
			v := node.Content[i+1]
			if k.Value == keyPath[0] {
				if len(keyPath) == 1 {
					if newValue == nil {
						// Remove this key-value pair
						node.Content = append(node.Content[:i], node.Content[i+2:]...)
						i -= 2 // adjust index after removal
						continue
					} else {
						v.Value = *newValue
					}
				} else {
					updateOrRemoveNodePath(v, keyPath[1:], newValue)
				}
			}
			updateOrRemoveNodePath(v, keyPath, newValue)
		}
	case yaml.SequenceNode:
		for _, n := range node.Content {
			updateOrRemoveNodePath(n, keyPath, newValue)
		}
	}
}

// SetOrRemoveYAMLKey updates or removes a key in a YAML file using dot notation for nested keys.
// If newValue is nil, the key is removed. If newValue is not nil, the key is updated (or created).
func SetOrRemoveYAMLKey(filePath, key string, newValue *string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return err
	}
	keyPath := strings.Split(key, ".")
	updateOrRemoveNodePath(&node, keyPath, newValue)

	data, err = yaml.Marshal(&node)
	if err != nil {
		return err
	}
	return os.WriteFile(filePath, data, 0644)
}
