package tor

import (
	"OnionVPN/internal/winjob"
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
)

type TorManager struct {
	baseDir string
	cmd     *exec.Cmd
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
	job     *winjob.Job
}

func NewTorManager(baseDir string) *TorManager {
	return &TorManager{
		baseDir: baseDir,
	}
}

func (m *TorManager) Start(torrcPath string, logCallback func(string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		return fmt.Errorf("tor is already running")
	}

	job, err := winjob.CreateJob()
	if err != nil {
		return fmt.Errorf("failed to create tor winjob: %w", err)
	}
	m.job = job

	m.ctx, m.cancel = context.WithCancel(context.Background())
	torExePath := filepath.Join(m.baseDir, "tor", "tor.exe")

	m.cmd = exec.CommandContext(m.ctx, torExePath, "-f", torrcPath)

	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		m.job.Close()
		m.job = nil
		m.cancel()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	m.cmd.Stderr = m.cmd.Stdout

	if err := m.cmd.Start(); err != nil {
		m.job.Close()
		m.job = nil
		m.cancel()
		return fmt.Errorf("failed to start tor process: %w", err)
	}

	if err := m.job.AssignProcess(m.cmd.Process.Pid); err != nil {
		_ = m.cmd.Process.Kill()
		_ = m.cmd.Wait()
		m.job.Close()
		m.job = nil
		m.cancel()
		return fmt.Errorf("failed to assign tor pid to job object: %w", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			logCallback(fmt.Sprintf("[tor] %s", scanner.Text()))
		}
	}()

	return nil
}

func (m *TorManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
	}

	if m.cmd != nil && m.cmd.Process != nil {
		_ = m.cmd.Process.Kill()
		_ = m.cmd.Wait()
	}

	if m.job != nil {
		m.job.Close()
		m.job = nil
	}

	m.cmd = nil
	return nil
}
