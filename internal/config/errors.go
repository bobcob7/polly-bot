package config

import "fmt"

type unexpectedTypeError struct {
	Type string
}

func (u unexpectedTypeError) Error() string {
	return fmt.Sprintf("unknown type: %q", u.Type)
}
