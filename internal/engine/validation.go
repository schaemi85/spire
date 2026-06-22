package engine

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

// ParseValidationRule parses a slot validation rule string and returns a
// validator function. Returns nil when the rule is empty (no validation).
//
// Supported rules:
//
//	"pattern:<regex>"      – value must match the regular expression
//	"minLength:<n>"        – value must be at least n characters
//	"maxLength:<n>"        – value must be at most n characters
//	"enum:<a>,<b>,<c>"    – value must be one of the listed options
//	"email"                – value must be a valid email address
//	"slug"                 – lowercase alphanumeric with hyphens, starts with a letter
//	"port"                 – valid TCP port number (1–65535)
//	"url"                  – valid URL with scheme
//	"startsWith:<prefix>"  – value must start with the given prefix
//	"endsWith:<suffix>"    – value must end with the given suffix
//	"numeric"              – value must be a valid integer
//	"minimum:<n>"          – numeric value must be >= n
//	"maximum:<n>"          – numeric value must be <= n
//	"semver"               – semantic version (v1.2.3 or 1.2.3)
func ParseValidationRule(rule string) (func(string) error, error) {
	if rule == "" {
		return nil, nil
	}

	parts := strings.SplitN(rule, ":", 2)
	name := parts[0]
	arg := ""
	if len(parts) == 2 {
		arg = parts[1]
	}

	switch name {
	case "pattern":
		if arg == "" {
			return nil, fmt.Errorf("validation rule \"pattern\" requires a regex (e.g. pattern:^[a-z]+$)")
		}
		re, err := regexp.Compile(arg)
		if err != nil {
			return nil, fmt.Errorf("invalid regex pattern %q: %w", arg, err)
		}
		return func(value string) error {
			if !re.MatchString(value) {
				return fmt.Errorf("value %q does not match pattern %s", value, arg)
			}
			return nil
		}, nil

	case "minLength":
		n, err := strconv.Atoi(arg)
		if err != nil {
			return nil, fmt.Errorf("validation rule \"minLength\" requires a number (e.g. minLength:3)")
		}
		return func(value string) error {
			if len(value) < n {
				return fmt.Errorf("value must be at least %d characters", n)
			}
			return nil
		}, nil

	case "maxLength":
		n, err := strconv.Atoi(arg)
		if err != nil {
			return nil, fmt.Errorf("validation rule \"maxLength\" requires a number (e.g. maxLength:30)")
		}
		return func(value string) error {
			if len(value) > n {
				return fmt.Errorf("value must be at most %d characters", n)
			}
			return nil
		}, nil

	case "enum":
		if arg == "" {
			return nil, fmt.Errorf("validation rule \"enum\" requires comma-separated options (e.g. enum:yes,no)")
		}
		options := strings.Split(arg, ",")
		for i := range options {
			options[i] = strings.TrimSpace(options[i])
		}
		return func(value string) error {
			for _, opt := range options {
				if value == opt {
					return nil
				}
			}
			return fmt.Errorf("value must be one of: %s", strings.Join(options, ", "))
		}, nil

	case "email":
		return func(value string) error {
			if value == "" {
				return fmt.Errorf("email cannot be empty")
			}
			at := strings.Index(value, "@")
			dot := strings.LastIndex(value, ".")
			if at <= 0 || dot <= at+1 || dot >= len(value)-1 {
				return fmt.Errorf("invalid email address")
			}
			return nil
		}, nil

	case "slug":
		slugRe := regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
		return func(value string) error {
			if !slugRe.MatchString(value) {
				return fmt.Errorf("value %q is not a valid slug (lowercase letters, digits, and hyphens; must start with a letter)", value)
			}
			return nil
		}, nil

	case "port":
		return func(value string) error {
			n, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("value %q is not a valid port number", value)
			}
			if n < 1 || n > 65535 {
				return fmt.Errorf("port must be between 1 and 65535, got %d", n)
			}
			return nil
		}, nil

	case "url":
		return func(value string) error {
			u, err := url.Parse(value)
			if err != nil || u.Scheme == "" || u.Host == "" {
				return fmt.Errorf("value %q is not a valid URL (must include scheme and host)", value)
			}
			return nil
		}, nil

	case "startsWith":
		if arg == "" {
			return nil, fmt.Errorf("validation rule \"startsWith\" requires a prefix (e.g. startsWith:https://)")
		}
		return func(value string) error {
			if !strings.HasPrefix(value, arg) {
				return fmt.Errorf("value must start with %q", arg)
			}
			return nil
		}, nil

	case "endsWith":
		if arg == "" {
			return nil, fmt.Errorf("validation rule \"endsWith\" requires a suffix (e.g. endsWith:.git)")
		}
		return func(value string) error {
			if !strings.HasSuffix(value, arg) {
				return fmt.Errorf("value must end with %q", arg)
			}
			return nil
		}, nil

	case "numeric":
		return func(value string) error {
			if _, err := strconv.Atoi(value); err != nil {
				return fmt.Errorf("value %q is not a valid number", value)
			}
			return nil
		}, nil

	case "minimum":
		n, err := strconv.Atoi(arg)
		if err != nil {
			return nil, fmt.Errorf("validation rule \"minimum\" requires a number (e.g. minimum:1)")
		}
		return func(value string) error {
			v, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("value %q is not a valid number", value)
			}
			if v < n {
				return fmt.Errorf("value must be at least %d, got %d", n, v)
			}
			return nil
		}, nil

	case "maximum":
		n, err := strconv.Atoi(arg)
		if err != nil {
			return nil, fmt.Errorf("validation rule \"maximum\" requires a number (e.g. maximum:100)")
		}
		return func(value string) error {
			v, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("value %q is not a valid number", value)
			}
			if v > n {
				return fmt.Errorf("value must be at most %d, got %d", n, v)
			}
			return nil
		}, nil

	case "semver":
		semverRe := regexp.MustCompile(`^v?(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)\.(?:0|[1-9]\d*)$`)
		return func(value string) error {
			if !semverRe.MatchString(value) {
				return fmt.Errorf("value %q is not a valid semantic version (expected format: v1.2.3 or 1.2.3)", value)
			}
			return nil
		}, nil

	default:
		return nil, fmt.Errorf("unknown validation rule %q", name)
	}
}
