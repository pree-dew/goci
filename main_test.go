package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
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

func mockCmdContext(ctx context.Context, exe string, arg ...string) *exec.Cmd {
	cs := []string{"-test.run=TestHelperProcess"}
	cs = append(cs, exe)
	cs = append(cs, arg...)

	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

func mockCmdContextTimeout(ctx context.Context, exe string, arg ...string) *exec.Cmd {
	cmd := mockCmdContext(ctx, exe, arg...)
	cmd.Env = append(cmd.Env, "GO_HELPER_TIMEOUT=1")
	return cmd
}

func TestHelperProcess(*testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	if os.Getenv("GO_HELPER_TIMEOUT") == "1" {
		time.Sleep(15 * time.Second)
	}

	if os.Args[2] == "git" {
		fmt.Fprintln(os.Stdout, "Everything up-to-date")
		os.Exit(0)
	}

	os.Exit(1)
}

func TestRun(t *testing.T) {
	_, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not found")
	}

	testdata := []struct {
		name        string
		proj        string
		out         string
		expErr      error
		setupGit    bool
		mockCommand func(context.Context, string, ...string) *exec.Cmd
	}{
		{name: "RunValid", proj: "./testdata/tool/", out: "go build: successful\ngo test: successful\ngofmt: successful\ngit push: successful\n", expErr: nil, setupGit: true, mockCommand: nil},
		{name: "SuccessMock", proj: "./testdata/tool/", out: "go build: successful\ngo test: successful\ngofmt: successful\ngit push: successful\n", expErr: nil, setupGit: false, mockCommand: mockCmdContext},
		{name: "RunInvalidDir", proj: "", out: "", expErr: fmt.Errorf("project dir is required: %w", ErrValidation), setupGit: false},
		{name: "RunFailedBuild", proj: "./testdata/toolErr/", out: "", expErr: &stepErr{step: "go build", msg: "failed to execute", cause: fmt.Errorf("exit status 1")}, setupGit: false},
		{name: "failTimeout", proj: "./testdata/tool/", out: "", expErr: &stepErr{step: "git push", msg: "failed timeout", cause: fmt.Errorf("context deadline exceeded")}, setupGit: false, mockCommand: mockCmdContextTimeout},
	}

	for _, tc := range testdata {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setupGit {
				teardown := setupGit(t, tc.proj)
				defer teardown()
			}

			if tc.mockCommand != nil {
				command = tc.mockCommand
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

				return
			}

			if buf.String() != tc.out {
				t.Errorf("expected %v, got %v", tc.out, buf.String())
			}
		})
	}
}

func TestRunKill(t *testing.T) {
	testcases := []struct {
		name   string
		proj   string
		sig    syscall.Signal
		expErr error
	}{
		{name: "SIGINT", proj: "./testdata/tool/", sig: syscall.SIGINT, expErr: fmt.Errorf("received signal interrupt: %w", ErrSignal)},
		{name: "SIGTERM", proj: "./testdata/tool/", sig: syscall.SIGTERM, expErr: fmt.Errorf("received signal terminated: %w", ErrSignal)},
		{name: "SIGQUIT", proj: "./testdata/tool/", sig: syscall.SIGQUIT, expErr: nil},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			command = mockCmdContextTimeout
			errCh := make(chan error)
			ignSigCh := make(chan os.Signal, 1)
			expSigCh := make(chan os.Signal, 1)

			signal.Notify(ignSigCh, syscall.SIGINT)
			defer signal.Stop(ignSigCh)

			signal.Notify(expSigCh, tc.sig)
			defer signal.Stop(expSigCh)

			go func() {
				errCh <- run(tc.proj, ioutil.Discard)
			}()

			go func() {
				time.Sleep(1 * time.Second)
				syscall.Kill(syscall.Getpid(), tc.sig)
			}()

			for {
				select {
				case err := <-errCh:
					if err != nil {
						if tc.expErr == nil {
							t.Fatalf("expected no error, got %v", err)
							return
						}

						if err.Error() != tc.expErr.Error() {
							t.Fatalf("expected error %v, got %v", tc.expErr, err)
						}
					}
					return
				case rec := <-expSigCh:
					if rec != tc.sig {
						t.Fatalf("expected signal %v, got %v", tc.sig, rec)
					}
					return
				case <-ignSigCh:
					return
				}
			}
		})
	}
}
