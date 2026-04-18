//go:build !windows

package testutil

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/creack/pty"
)

// PTYSession is a running command with stdin and stderr attached to a
// pseudo-terminal. Stdout remains separately captured so tests can assert the
// shell-facing output contract.
type PTYSession struct {
	cmd    *exec.Cmd
	pty    *os.File
	cancel context.CancelFunc

	stderrMu  sync.Mutex
	stderrBuf bytes.Buffer
	stdoutBuf *bytes.Buffer

	waitOnce sync.Once
	waitCh   chan error
	readCh   chan error
}

// RunWWPTY starts ww with stdin and stderr attached to a pseudo-terminal.
func (e *HostEnv) RunWWPTY(ctx context.Context, dir string, args ...string) (*PTYSession, error) {
	if ctx == nil {
		ctx = e.ctx
	}
	ctx, cancel := context.WithCancel(ctx)
	c := exec.CommandContext(ctx, e.wwBinaryPath, args...)
	if dir != "" {
		c.Dir = dir
	}
	c.Env = testEnv(e.gitConfigPath)
	stdout := &bytes.Buffer{}
	c.Stdout = stdout

	ptmx, err := pty.StartWithSize(c, &pty.Winsize{Rows: 32, Cols: 120})
	if err != nil {
		cancel()
		return nil, err
	}

	session := &PTYSession{
		cmd:       c,
		pty:       ptmx,
		cancel:    cancel,
		waitCh:    make(chan error, 1),
		readCh:    make(chan error, 1),
		stdoutBuf: stdout,
	}

	go func() {
		_, err := io.Copy(&lockedBuffer{mu: &session.stderrMu, buf: &session.stderrBuf}, ptmx)
		session.readCh <- err
	}()
	go func() {
		session.waitCh <- c.Wait()
	}()

	return session, nil
}

// WriteKeys sends terminal input to the running PTY.
func (s *PTYSession) WriteKeys(keys string) error {
	_, err := io.WriteString(s.pty, keys)
	return err
}

// WaitForOutput waits until the PTY-rendered stderr contains text.
func (s *PTYSession) WaitForOutput(text string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		if strings.Contains(s.Stderr(), text) {
			return nil
		}
		select {
		case err := <-s.waitCh:
			s.waitCh <- err
			return fmt.Errorf("process exited before %q appeared: %w\nstderr:\n%s", text, err, s.Stderr())
		default:
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("timed out waiting for %q\nstderr:\n%s", text, s.Stderr())
		}
		time.Sleep(20 * time.Millisecond)
	}
}

// Wait waits for the command to exit and returns captured stdout and PTY output.
func (s *PTYSession) Wait(timeout time.Duration) (string, string, error) {
	var waitErr error
	done := make(chan struct{})
	go func() {
		s.waitOnce.Do(func() {
			waitErr = <-s.waitCh
			_ = s.pty.Close()
			s.cancel()
		})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(timeout):
		s.cancel()
		_ = s.pty.Close()
		return s.Stdout(), s.Stderr(), fmt.Errorf("timed out waiting for process exit")
	}

	select {
	case <-s.readCh:
	case <-time.After(time.Second):
	}

	if waitErr != nil {
		if exitErr, ok := waitErr.(*exec.ExitError); ok {
			waitErr = fmt.Errorf("exit %d", exitErr.ExitCode())
		}
	}
	return s.Stdout(), s.Stderr(), waitErr
}

func (s *PTYSession) Stdout() string {
	return s.stdoutBuf.String()
}

func (s *PTYSession) Stderr() string {
	s.stderrMu.Lock()
	defer s.stderrMu.Unlock()
	return s.stderrBuf.String()
}

type lockedBuffer struct {
	mu  *sync.Mutex
	buf *bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func testEnv(gitConfigPath string) []string {
	baseEnv := os.Environ()
	env := make([]string, 0, len(baseEnv)+1)
	for _, v := range baseEnv {
		if strings.HasPrefix(v, "GIT_CONFIG_GLOBAL=") {
			continue
		}
		env = append(env, v)
	}
	env = append(env, "GIT_CONFIG_GLOBAL="+gitConfigPath)
	return env
}
