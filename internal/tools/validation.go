package tools

import (
	"fmt"
	"strings"
)

func ValidateBoolAnswer(r string) (bool, error) {
	switch strings.ToLower(r) {
	case "y":
		fallthrough
	case "yes":
		return true, nil
	case "n":
		fallthrough
	case "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid response, please use yes or no / y or n")
	}
}
