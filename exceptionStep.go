package main

import (
	"bytes"
	"fmt"
	"os/exec"
)

type exceptionStep struct {
	step
}

type executer interface {
	execute() (string, error)
}

func newExceptionStep(name, exe, proj, msg string, args []string) exceptionStep {
	s := exceptionStep{}
	s.step = newStep(name, exe, proj, msg, args)
	return s
}

func (s exceptionStep) execute() (string, error) {
	buff := &bytes.Buffer{}
	cmd := exec.Command(s.exe, s.args...)
	cmd.Dir = s.proj
	cmd.Stdout = buff

	if err := cmd.Run(); err != nil {
		return "", &stepErr{
			step:  s.name,
			msg:   "failed to execute",
			cause: err,
		}
	}

	if buff.Len() > 0 {
		return "", &stepErr{
			step:  s.name,
			msg:   fmt.Sprintf("invalid format: %q", buff.String()),
			cause: nil,
		}
	}

	return s.msg, nil
}
