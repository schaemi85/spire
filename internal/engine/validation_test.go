package engine

import (
	"testing"
)

func TestParseValidationRule_Empty(t *testing.T) {
	fn, err := ParseValidationRule("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fn != nil {
		t.Fatal("expected nil validator for empty rule")
	}
}

func TestParseValidationRule_Pattern(t *testing.T) {
	fn, err := ParseValidationRule("pattern:^[a-z][a-z0-9-]*$")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fn("my-app"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := fn("My-App"); err == nil {
		t.Fatal("expected error for uppercase")
	}
	if err := fn("123"); err == nil {
		t.Fatal("expected error for leading digit")
	}
}

func TestParseValidationRule_Pattern_InvalidPattern(t *testing.T) {
	_, err := ParseValidationRule("pattern:[invalid")
	if err == nil {
		t.Fatal("expected error for invalid regex")
	}
}

func TestParseValidationRule_Pattern_MissingPattern(t *testing.T) {
	_, err := ParseValidationRule("pattern:")
	if err == nil {
		t.Fatal("expected error for missing pattern")
	}
}

func TestParseValidationRule_MinLength(t *testing.T) {
	fn, err := ParseValidationRule("minLength:3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fn("abc"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := fn("ab"); err == nil {
		t.Fatal("expected error for too short")
	}
}

func TestParseValidationRule_MaxLength(t *testing.T) {
	fn, err := ParseValidationRule("maxLength:5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fn("hello"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := fn("toolong"); err == nil {
		t.Fatal("expected error for too long")
	}
}

func TestParseValidationRule_Enum(t *testing.T) {
	fn, err := ParseValidationRule("enum:yes,no")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fn("yes"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := fn("no"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := fn("maybe"); err == nil {
		t.Fatal("expected error for invalid option")
	}
}

func TestParseValidationRule_Email(t *testing.T) {
	fn, err := ParseValidationRule("email")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fn("user@example.com"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := fn("invalid"); err == nil {
		t.Fatal("expected error for invalid email")
	}
	if err := fn(""); err == nil {
		t.Fatal("expected error for empty email")
	}
	if err := fn("@example.com"); err == nil {
		t.Fatal("expected error for missing username")
	}
}

func TestParseValidationRule_Unknown(t *testing.T) {
	_, err := ParseValidationRule("foobar:123")
	if err == nil {
		t.Fatal("expected error for unknown rule")
	}
}

func TestParseValidationRule_MinLength_InvalidArg(t *testing.T) {
	_, err := ParseValidationRule("minLength:abc")
	if err == nil {
		t.Fatal("expected error for non-numeric argument")
	}
}

func TestParseValidationRule_Enum_MissingOptions(t *testing.T) {
	_, err := ParseValidationRule("enum:")
	if err == nil {
		t.Fatal("expected error for missing options")
	}
}

func TestParseValidationRule_Slug(t *testing.T) {
	fn, err := ParseValidationRule("slug")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, valid := range []string{"my-app", "app", "a1", "hello-world-42"} {
		if err := fn(valid); err != nil {
			t.Fatalf("expected %q to be valid: %v", valid, err)
		}
	}
	for _, invalid := range []string{"My-App", "123", "-app", "my app", "APP", ""} {
		if err := fn(invalid); err == nil {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}
}

func TestParseValidationRule_Port(t *testing.T) {
	fn, err := ParseValidationRule("port")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, valid := range []string{"1", "80", "443", "8080", "65535"} {
		if err := fn(valid); err != nil {
			t.Fatalf("expected %q to be valid: %v", valid, err)
		}
	}
	for _, invalid := range []string{"0", "65536", "-1", "abc", "", "99999"} {
		if err := fn(invalid); err == nil {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}
}

func TestParseValidationRule_URL(t *testing.T) {
	fn, err := ParseValidationRule("url")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, valid := range []string{"https://example.com", "http://localhost:8080", "git://github.com/org/repo.git"} {
		if err := fn(valid); err != nil {
			t.Fatalf("expected %q to be valid: %v", valid, err)
		}
	}
	for _, invalid := range []string{"not-a-url", "example.com", "", "/path/only"} {
		if err := fn(invalid); err == nil {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}
}

func TestParseValidationRule_StartsWith(t *testing.T) {
	fn, err := ParseValidationRule("startsWith:https://")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fn("https://example.com"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := fn("http://example.com"); err == nil {
		t.Fatal("expected error for wrong prefix")
	}
}

func TestParseValidationRule_StartsWith_MissingArg(t *testing.T) {
	_, err := ParseValidationRule("startsWith:")
	if err == nil {
		t.Fatal("expected error for missing prefix")
	}
}

func TestParseValidationRule_EndsWith(t *testing.T) {
	fn, err := ParseValidationRule("endsWith:.git")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := fn("repo.git"); err != nil {
		t.Fatalf("expected valid: %v", err)
	}
	if err := fn("repo.tar"); err == nil {
		t.Fatal("expected error for wrong suffix")
	}
}

func TestParseValidationRule_EndsWith_MissingArg(t *testing.T) {
	_, err := ParseValidationRule("endsWith:")
	if err == nil {
		t.Fatal("expected error for missing suffix")
	}
}

func TestParseValidationRule_Semver(t *testing.T) {
	fn, err := ParseValidationRule("semver")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, valid := range []string{"1.0.0", "v1.0.0", "0.1.0", "v12.34.56"} {
		if err := fn(valid); err != nil {
			t.Fatalf("expected %q to be valid: %v", valid, err)
		}
	}
	for _, invalid := range []string{"1.0", "v1", "latest", "1.0.0-beta", "abc", ""} {
		if err := fn(invalid); err == nil {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}
}

func TestParseValidationRule_Numeric(t *testing.T) {
	fn, err := ParseValidationRule("numeric")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, valid := range []string{"0", "42", "-1", "99999"} {
		if err := fn(valid); err != nil {
			t.Fatalf("expected %q to be valid: %v", valid, err)
		}
	}
	for _, invalid := range []string{"abc", "12.5", "", "3a"} {
		if err := fn(invalid); err == nil {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}
}

func TestParseValidationRule_Minimum(t *testing.T) {
	fn, err := ParseValidationRule("minimum:1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, valid := range []string{"1", "100", "65535"} {
		if err := fn(valid); err != nil {
			t.Fatalf("expected %q to be valid: %v", valid, err)
		}
	}
	for _, invalid := range []string{"0", "-1"} {
		if err := fn(invalid); err == nil {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}
	if err := fn("abc"); err == nil {
		t.Fatal("expected error for non-numeric input")
	}
}

func TestParseValidationRule_Minimum_InvalidArg(t *testing.T) {
	_, err := ParseValidationRule("minimum:abc")
	if err == nil {
		t.Fatal("expected error for non-numeric argument")
	}
}

func TestParseValidationRule_Maximum(t *testing.T) {
	fn, err := ParseValidationRule("maximum:100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, valid := range []string{"0", "50", "100"} {
		if err := fn(valid); err != nil {
			t.Fatalf("expected %q to be valid: %v", valid, err)
		}
	}
	for _, invalid := range []string{"101", "9999"} {
		if err := fn(invalid); err == nil {
			t.Fatalf("expected %q to be invalid", invalid)
		}
	}
	if err := fn("abc"); err == nil {
		t.Fatal("expected error for non-numeric input")
	}
}

func TestParseValidationRule_Maximum_InvalidArg(t *testing.T) {
	_, err := ParseValidationRule("maximum:abc")
	if err == nil {
		t.Fatal("expected error for non-numeric argument")
	}
}
