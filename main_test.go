package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func setupGit(t *testing.T, proj string) func() {
	t.Helper()

	// check git is installed or not
	gitExe, err := exec.LookPath("git")
	if err != nil {
		t.Fatal("git not found")
	}

	tempDir, err := ioutil.TempDir("", "gocitest")
	if err != nil {
		t.Fatal(err)
	}

	projPath, err := filepath.Abs(proj)
	if err != nil {
		t.Fatal(err)
	}

	remoteURI := fmt.Sprintf("file://%s", tempDir)

	gitCmdList := []struct {
		args []string
		dir  string
		env  []string
	}{
		{args: []string{"init", "--bare"}, dir: tempDir, env: nil},
		{args: []string{"init"}, dir: projPath, env: nil},
		{args: []string{"remote", "add", "origin", remoteURI}, dir: projPath, env: nil},
		{args: []string{"add", "."}, dir: projPath, env: nil},
		{
			args: []string{"commit", "-m", "test"}, dir: projPath,
			env: []string{
				"GIT_AUTHOR_NAME=Test",
				"GIT_COMMITTER_NAME=Test",
				"GIT_COMMITTER_EMAIL=test@example.com",
				"GIT_AUTHOR_EMAIL=test@example.com",
			},
		},
	}

	for _, g := range gitCmdList {
		gitCmd := exec.Command(gitExe, g.args...)
		gitCmd.Dir = g.dir

		if g.env != nil {
			gitCmd.Env = append(os.Environ(), g.env...)
		}

		if err := gitCmd.Run(); err != nil {
			t.Fatal(err)
		}

	}

	return func() {
		os.RemoveAll(tempDir)
		os.RemoveAll(filepath.Join(projPath, ".git"))
	}
}

func TestRun(t *testing.T) {
	_, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not found")
	}

	testdata := []struct {
		name     string
		proj     string
		out      string
		expErr   error
		setupGit bool
	}{
		{name: "RunValid", proj: "./testdata/tool/", out: "go build: successful\ngo test: successful\ngofmt: successful\ngit push: successful\n", expErr: nil, setupGit: true},
		{name: "RunInvalidDir", proj: "", out: "", expErr: fmt.Errorf("project dir is required: %w", ErrValidation), setupGit: false},
		{name: "RunFailedBuild", proj: "./testdata/toolErr/", out: "", expErr: &stepErr{step: "go build", msg: "failed to execute", cause: fmt.Errorf("exit status 1")}, setupGit: false},
	}

	for _, tc := range testdata {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupGit {
				teardown := setupGit(t, tc.proj)
				defer teardown()
			}

			var buf bytes.Buffer
			err := run(tc.proj, &buf)
			if err != nil {
				if tc.expErr == nil {
					t.Fatalf("expected no error, got %v", err)
				}

				if err.Error() != tc.expErr.Error() {
					t.Fatalf("expected error %v, got %v", tc.expErr, err)
				}
			}

			if buf.String() != tc.out {
				t.Errorf("expected %v, got %v", tc.out, buf.String())
			}
		})
	}
}
