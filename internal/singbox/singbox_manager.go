package singbox

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
)

type SingBoxManager struct {
	baseDir string
	cmd     *exec.Cmd
	mu      sync.Mutex
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewSingBoxManager(baseDir string) *SingBoxManager {
	return &SingBoxManager{
		baseDir: baseDir,
	}
}

func (m *SingBoxManager) Start(configPath string, logCallback func(string)) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd != nil && m.cmd.Process != nil {
		return fmt.Errorf("sing-box is already running")
	}

	m.ctx, m.cancel = context.WithCancel(context.Background())
	exePath := filepath.Join(m.baseDir, "sing-box.exe")

	wintunPath := filepath.Join(m.baseDir, "wintun.dll")
	if _, err := os.Stat(wintunPath); os.IsNotExist(err) {
		m.cancel()
		return fmt.Errorf("wintun.dll не найден в папке бинарников: %s", m.baseDir)
	}

	m.cmd = exec.CommandContext(m.ctx, exePath, "run", "-c", configPath)
	m.cmd.Dir = m.baseDir
	m.cmd.Env = append(os.Environ(), "ENABLE_DEPRECATED_LEGACY_DNS_SERVERS=true")
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		m.cancel()
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	m.cmd.Stderr = m.cmd.Stdout

	if err := m.cmd.Start(); err != nil {
		m.cancel()
		return fmt.Errorf("критическая ошибка cmd.Start(): %w. Проверьте права Администратора!", err)
	}

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			logCallback(fmt.Sprintf("[sing-box] %s", scanner.Text()))
		}
		_ = m.cmd.Wait()

		m.mu.Lock()
		m.cmd = nil
		m.mu.Unlock()
	}()

	return nil
}

func (m *SingBoxManager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cmd == nil || m.cmd.Process != nil {
		if m.cancel != nil {
			m.cancel()
		}
		if m.cmd != nil && m.cmd.Process != nil {
			killCmd := exec.Command("taskkill", "/F", "/T", "/PID", fmt.Sprintf("%d", m.cmd.Process.Pid))
			killCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
			_ = killCmd.Run()
		}
		m.cmd = nil
	}
	return nil
}
