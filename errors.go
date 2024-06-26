package main

import "fmt"

var (
	ErrValidation = fmt.Errorf("validation failed")
	ErrSignal     = fmt.Errorf("signal received")
)

type stepErr struct {
	step  string
	msg   string
	cause error
}

func (s *stepErr) Error() string {
	return fmt.Sprintf("Step: %q: %s: Cause: %v", s.step, s.msg, s.cause)
}

func (s *stepErr) Is(target error) bool {
	t, ok := target.(*stepErr)
	if !ok {
		return false
	}

	return s.step == t.step
}

func (s *stepErr) Unwrap() error {
	return s.cause
}
