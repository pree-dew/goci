package main

import "os/exec"

type step struct {
	name string
	exe  string
	args []string
	proj string
	msg  string
}

func (s step) execute() (string, error) {
	cmd := exec.Command(s.exe, s.args...)
	cmd.Dir = s.proj

	if err := cmd.Run(); err != nil {
		return "", &stepErr{
			step:  s.name,
			msg:   "failed to execute",
			cause: err,
		}
	}

	return s.msg, nil
}

func newStep(name, exe, proj, msg string, args []string) step {
	return step{
		name: name,
		exe:  exe,
		args: args,
		proj: proj,
		msg:  msg,
	}
}
