package tools

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"
)

// readMaskedLine reads a line of input from the terminal, displaying a '*'
// for each character typed. Supports backspace to delete characters.
// The actual input is never displayed on screen.
func readMaskedLine() (string, error) {
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("failed to set terminal to raw mode: %w", err)
	}
	defer func() { _ = term.Restore(fd, oldState) }()

	var input []byte
	buf := make([]byte, 1)
	for {
		_, err := os.Stdin.Read(buf)
		if err != nil {
			return "", fmt.Errorf("error reading input: %w", err)
		}
		c := buf[0]
		switch {
		case c == '\n' || c == '\r':
			fmt.Print("\r\n")
			return string(input), nil
		case c == 127 || c == 8: // backspace or delete
			if len(input) > 0 {
				input = input[:len(input)-1]
				fmt.Print("\b \b")
			}
		case c == 3: // Ctrl+C
			fmt.Print("\r\n")
			_ = term.Restore(fd, oldState)
			os.Exit(1)
		case c >= 32 && c < 127: // printable ASCII
			input = append(input, c)
			fmt.Print("*")
		}
	}
}

// SecretPrompt prompts the user for a secret value, masking each typed character
// with a '*'. If isValid is non-nil, the prompt repeats until validation passes.
// This is suitable for passwords, tokens, and other sensitive inputs.
func SecretPrompt(prompt string, isValid func(string) error) string {
	for {
		fmt.Print(prompt)
		value, err := readMaskedLine()
		if err != nil {
			fmt.Printf("Error reading input: %v\n", err)
			continue
		}
		if isValid != nil {
			if err := isValid(value); err != nil {
				fmt.Println(err.Error())
				continue
			}
		}
		return value
	}
}

func ReadLine(prompt string) string {
	fmt.Print(prompt)
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

// MandatoryPrompt prompts the user for a value until isValid returns true.
// If isValid is nil, validation is skipped.
func MandatoryPrompt(prompt string, isValid func(string) error) string {
	var value string
	for {
		value = ReadLine(prompt)
		if isValid == nil {
			return value
		}
		err := isValid(value)
		if err == nil {
			return value
		} else {
			fmt.Println(err.Error())
		}
	}
}

// OptionalPrompt prompts the user for a optional value, return the default value if not entered by the user.
// If isValid is nil, validation is skipped.
func OptionalPrompt(prompt, defaultValue string, isValid func(string) error) string {
	var value string
	for {
		value = ReadLine(prompt)
		if value != "" {
			if isValid == nil {
				return value
			}
			err := isValid(value)
			if err == nil {
				return value
			} else {
				fmt.Println(err.Error())
			}
		} else {
			return defaultValue
		}
	}
}

// ConfirmAction prompts the user for confirmation and returns true if confirmed.
func ConfirmAction(prompt string) (bool, error) {
	fmt.Printf("%s (yes/no): ", prompt)
	var response string
	_, err := fmt.Scanln(&response)
	if err != nil {
		return false, err
	}
	return ValidateBoolAnswer(response)
}
