package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"OnionVPN/internal/singbox"
	"OnionVPN/internal/sysproxy"
	"OnionVPN/internal/tor"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx            context.Context
	torManager     *tor.TorManager
	singBoxManager *singbox.SingBoxManager
	torDir         string
	baseDir        string
}

func NewApp() *App {
	return &App{}
}

func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx

	userConfig, err := os.UserConfigDir()
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to get user config dir: %v", err)
		return
	}

	a.baseDir = filepath.Join(userConfig, "OnionVPN")
	a.torDir = filepath.Join(a.baseDir, "tor_bundle")

	a.killOurProcesses()

	err = tor.ExtractBinaries(a.torDir)
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to extract Tor binaries: %v", err)
		return
	}

	a.torManager = tor.NewTorManager(a.torDir)
	a.singBoxManager = singbox.NewSingBoxManager(a.torDir)
}

func (a *App) DomReady(ctx context.Context) {}

func (a *App) Shutdown(ctx context.Context) {
	_ = sysproxy.OffSystemPac()
	if a.singBoxManager != nil {
		_ = a.singBoxManager.Stop()
	}
	if a.torManager != nil {
		_ = a.torManager.Stop()
	}
}

func (a *App) ConnectToTor(bridges []string, routedDomains []string, useSysProxy bool) string {
	if a.torManager == nil || a.singBoxManager == nil {
		return "Менеджеры не инициализированы"
	}

	a.killOurProcesses()

	torrcPath, err := tor.GenerateTorrc(a.torDir, bridges)
	if err != nil {
		return fmt.Sprintf("Ошибка конфигурации Tor: %v", err)
	}

	logCallback := func(logLine string) {
		runtime.EventsEmit(a.ctx, "tor:log", logLine)
	}

	err = a.torManager.Start(torrcPath, logCallback)
	if err != nil {
		return fmt.Sprintf("Не удалось запустить Tor: %v", err)
	}

	if useSysProxy {
		runtime.EventsEmit(a.ctx, "tor:log", "[System] Включение режима: Системный прокси (SysProxy PAC)")

		pacPath, err := sysproxy.GeneratePacFile(a.torDir, routedDomains, 9050)
		if err != nil {
			_ = a.torManager.Stop()
			return fmt.Sprintf("Ошибка конфигурации системного прокси: %v", err)
		}

		err = sysproxy.SetupSystemPac(pacPath)
		if err != nil {
			_ = a.torManager.Stop()
			return fmt.Sprintf("Не удалось активировать системный прокси: %v", err)
		}

		runtime.EventsEmit(a.ctx, "tor:log", "Bootstrapped 100% (Режим SysProxy успешно активирован)")
	} else {
		runtime.EventsEmit(a.ctx, "tor:log", "[System] Включение режима: Комплексный TUN (sing-box)")

		sbConfigPath, err := singbox.GenerateConfig(a.baseDir, routedDomains, 9050)
		if err != nil {
			_ = a.torManager.Stop()
			return fmt.Sprintf("Не удалось сгенерировать конфиг sing-box: %v", err)
		}

		err = a.singBoxManager.Start(sbConfigPath, logCallback)
		if err != nil {
			_ = a.torManager.Stop()
			return fmt.Sprintf("Не удалось запустить sing-box TUN: %v", err)
		}
	}

	return "success"
}

func (a *App) DisconnectFromTor() string {
	var errProxy, errSB, errTor error

	errProxy = sysproxy.OffSystemPac()

	if a.singBoxManager != nil {
		errSB = a.singBoxManager.Stop()
	}

	if a.torManager != nil {
		errTor = a.torManager.Stop()
	}

	if errProxy != nil || errSB != nil || errTor != nil {
		return fmt.Sprintf("Ошибки при остановке. Прокси: %v, SingBox: %v, Tor: %v", errProxy, errSB, errTor)
	}

	return "success"
}

func (a *App) OnBeforeClose(ctx context.Context) bool {
	_ = sysproxy.OffSystemPac()
	if a.singBoxManager != nil {
		_ = a.singBoxManager.Stop()
	}
	if a.torManager != nil {
		_ = a.torManager.Stop()
	}
	return false
}

func (a *App) killOurProcesses() {
	userConfig, err := os.UserConfigDir()
	if err != nil {
		return
	}
	ourAppDir := filepath.Clean(filepath.Join(userConfig, "OnionVPN"))

	cmd := exec.Command("wmic", "process", "get", "ExecutablePath,ProcessId")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Run(); err != nil {
		return
	}

	lines := strings.Split(out.String(), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "ExecutablePath") {
			continue
		}

		isTarget := strings.Contains(strings.ToLower(line), "tor.exe") ||
			strings.Contains(strings.ToLower(line), "sing-box.exe")

		if isTarget {
			if strings.HasPrefix(strings.ToLower(line), strings.ToLower(ourAppDir)) {
				fields := strings.Fields(line)
				if len(fields) < 2 {
					continue
				}
				pidStr := fields[len(fields)-1]

				killCmd := exec.Command("taskkill", "/F", "/PID", pidStr)
				killCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
				_ = killCmd.Run()

				time.Sleep(50 * time.Millisecond)
			}
		}
	}
}
