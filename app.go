package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	currentVersion string
}

type DomainState struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
}

type UserConfig struct {
	Bridges      string        `json:"bridges"`
	DomainStates []DomainState `json:"domain_states"`
	UseSysProxy  bool          `json:"use_sys_proxy"`
}

func NewApp() *App {
	return &App{
		currentVersion: "1.7",
	}
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

func (a *App) CheckForUpdates() (map[string]interface{}, error) {
	url := "https://api.github.com/repos/HasanovDoc/OnionVPN/releases/latest"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("ошибка запроса обновлений: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("некорректный статус ответа: %d", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadUrl string `json:"browser_download_url"`
		} `json:"assets"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("ошибка декодирования json: %w", err)
	}

	latestVersion := strings.TrimPrefix(release.TagName, "v")
	current := strings.TrimPrefix(a.currentVersion, "v")

	hasUpdate := latestVersion != current

	var downloadUrl string
	for _, asset := range release.Assets {
		if strings.HasSuffix(strings.ToLower(asset.Name), ".exe") {
			downloadUrl = asset.BrowserDownloadUrl
			break
		}
	}

	return map[string]interface{}{
		"hasUpdate":   hasUpdate,
		"version":     release.TagName,
		"downloadUrl": downloadUrl,
	}, nil
}

func (a *App) ApplyUpdate(downloadUrl string) error {
	runtime.LogInfof(a.ctx, "Начало скачивания обновления: %s", downloadUrl)

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("не удалось определить путь к файлу: %w", err)
	}

	resp, err := http.Get(downloadUrl)
	if err != nil {
		return fmt.Errorf("не удалось скачать обновление: %w", err)
	}
	defer resp.Body.Close()

	tmpDir := os.TempDir()
	tmpNewExe := filepath.Join(tmpDir, "onionvpn_new.exe")

	out, err := os.Create(tmpNewExe)
	if err != nil {
		return fmt.Errorf("не удалось создать временный файл: %w", err)
	}
	if _, err = io.Copy(out, resp.Body); err != nil {
		out.Close()
		return fmt.Errorf("ошибка записи файла: %w", err)
	}
	out.Close()

	batPath := filepath.Join(tmpDir, "onionvpn_updater.bat")

	batContent := fmt.Sprintf(`@echo off
		chcp 65001 > nul
		:wait_process
		tasklist /FI "IMAGENAME eq %s" 2>NUL | find /I /N "%s">NUL
		if "%%ERRORLEVEL%%"=="0" (
			timeout /t 1 /nobreak > nul
			goto wait_process
		)

		del /f /q "%s"
		move /y "%s" "%s"
		start "" "%s"
		del /f /q "%%~f0"
		`, filepath.Base(currentExe), filepath.Base(currentExe), currentExe, tmpNewExe, currentExe, currentExe)

	err = os.WriteFile(batPath, []byte(batContent), 0755)
	if err != nil {
		return fmt.Errorf("не удалось создать скрипт обновления: %w", err)
	}

	cmd := exec.Command("cmd", "/c", batPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("не удалось запустить скрипт обновления: %w", err)
	}

	os.Exit(0)
	return nil
}

func (a *App) LoadConfig() map[string]interface{} {
	configPath := filepath.Join(a.baseDir, "config.json")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return map[string]interface{}{
			"bridges":       "",
			"domain_states": []interface{}{},
			"use_sys_proxy": false,
		}
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		runtime.LogErrorf(a.ctx, "Failed to read config file: %v", err)
		return nil
	}

	var config UserConfig
	if err := json.Unmarshal(data, &config); err != nil {
		runtime.LogErrorf(a.ctx, "Failed to unmarshal config: %v", err)
		return nil
	}

	return map[string]interface{}{
		"bridges":       config.Bridges,
		"domain_states": config.DomainStates,
		"use_sys_proxy": config.UseSysProxy,
	}
}

func (a *App) SaveConfig(bridges string, domains []interface{}, useSysProxy bool) string {
	configPath := filepath.Join(a.baseDir, "config.json")

	if err := os.MkdirAll(a.baseDir, 0755); err != nil {
		return fmt.Sprintf("Ошибка создания директории: %v", err)
	}

	domainBytes, _ := json.Marshal(domains)
	var domainStates []DomainState
	_ = json.Unmarshal(domainBytes, &domainStates)

	config := UserConfig{
		Bridges:      bridges,
		DomainStates: domainStates,
		UseSysProxy:  useSysProxy,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Sprintf("Ошибка маршалинга JSON: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Sprintf("Ошибка записи файла: %v", err)
	}

	return "success"
}

func (a *App) ToggleAutostart(enable bool) error {
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get exe path: %w", err)
	}

	taskName := "OnionVPN_Autostart"

	if enable {
		cmd := exec.Command("schtasks", "/Create", "/TN", taskName, "/TR", fmt.Sprintf(`"%s" --autostart`, currentExe), "/SC", "ONLOGON", "/RL", "HIGHEST", "/F")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create task: %w", err)
		}
	} else {
		cmd := exec.Command("schtasks", "/Delete", "/TN", taskName, "/F")
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
		_ = cmd.Run()
	}

	return nil
}

func (a *App) IsAutostartEnabled() bool {
	taskName := "OnionVPN_Autostart"
	cmd := exec.Command("schtasks", "/Query", "/TN", taskName)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run() == nil
}

func (a *App) MinimizeToTray() {
	runtime.WindowHide(a.ctx)
}
