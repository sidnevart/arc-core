package provider

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

func runProviderCommand(binary string, args []string, dir string, timeout time.Duration, stdoutPath string, stderrPath string, extraEnv ...string) ([]byte, []byte, error, bool) {
	resolvedBinary, _, err := resolveBinary(binary)
	if err != nil {
		return nil, nil, err, false
	}
	cmd := exec.Command(resolvedBinary, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Env = providerCommandEnv(resolvedBinary, extraEnv...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), err, false
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), err, false
	}

	stdoutFile, err := createOutputFile(stdoutPath)
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), err, false
	}
	if stdoutFile != nil {
		defer stdoutFile.Close()
	}
	stderrFile, err := createOutputFile(stderrPath)
	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), err, false
	}
	if stderrFile != nil {
		defer stderrFile.Close()
	}

	if err := cmd.Start(); err != nil {
		return stdout.Bytes(), stderr.Bytes(), err, false
	}

	stdoutDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(io.MultiWriter(&stdout, stdoutFileOrDiscard(stdoutFile)), stdoutPipe)
		stdoutDone <- copyErr
	}()

	stderrDone := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(io.MultiWriter(&stderr, stdoutFileOrDiscard(stderrFile)), stderrPipe)
		stderrDone <- copyErr
	}()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	if timeout <= 0 {
		err := <-done
		_ = <-stdoutDone
		_ = <-stderrDone
		return stdout.Bytes(), stderr.Bytes(), err, false
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		_ = <-stdoutDone
		_ = <-stderrDone
		return stdout.Bytes(), stderr.Bytes(), err, false
	case <-timer.C:
		_ = killProcessGroup(cmd)
		waitErr := <-done
		_ = <-stdoutDone
		_ = <-stderrDone
		if waitErr == nil {
			waitErr = errors.New("process terminated after timeout")
		}
		return stdout.Bytes(), stderr.Bytes(), fmt.Errorf("%s timed out after %s: %w", binary, timeout, waitErr), true
	}
}

func createOutputFile(path string) (*os.File, error) {
	if path == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func stdoutFileOrDiscard(file *os.File) io.Writer {
	if file == nil {
		return io.Discard
	}
	return file
}

func killProcessGroup(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		_ = cmd.Process.Kill()
		return err
	}
	return nil
}
