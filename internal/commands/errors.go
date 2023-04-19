package commands

import (
	"errors"
	"fmt"
)

type failedResponseInteractionError struct {
	err error
}

func (f failedResponseInteractionError) Unwrap() error {
	return f.err
}

func (f failedResponseInteractionError) Error() string {
	return fmt.Sprintf("failed to send response interation: %v", f.err)
}

type unexpectedCategoryError struct {
	category string
}

func (u unexpectedCategoryError) Error() string {
	return fmt.Sprintf("unknown category: %q", u.category)
}

type unexpectedNumberOfTorrentsError struct {
	want int
	got  int
}

func (u unexpectedNumberOfTorrentsError) Error() string {
	return fmt.Sprintf("scraped %d torrents intead of %d", u.got, u.want)
}

var errFailedTypeAssertion = errors.New("failed type assertion")
