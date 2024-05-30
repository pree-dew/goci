package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

func run(proj string, out io.Writer) error {
	if proj == "" {
		return fmt.Errorf("project dir is required: %w", ErrValidation)
	}

	pipeline := make([]executer, 4)
	pipeline[0] = newStep("go build", "go", proj, "go build: successful", []string{"build", ".", "errors"})
	pipeline[1] = newStep("go test", "go", proj, "go test: successful", []string{"test", "-v"})
	pipeline[2] = newExceptionStep("go fmt", "gofmt", proj, "gofmt: successful", []string{"-l", "."})
	pipeline[3] = newTimeoutStep("git push", "git", proj, "git push: successful", []string{"push", "origin", "main"}, 10*time.Second)

	for _, s := range pipeline {
		msg, err := s.execute()
		if err != nil {
			return err
		}

		_, err = fmt.Fprintln(out, msg)
		if err != nil {
			return err
		}
	}

	return nil
}

func main() {
	proj := flag.String("proj", "", "project directory")
	flag.Parse()

	if err := run(*proj, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
