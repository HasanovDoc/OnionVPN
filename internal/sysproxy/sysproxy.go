package sysproxy

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"golang.org/x/sys/windows/registry"
)

const internetSettingsPath = `Software\Microsoft\Windows\CurrentVersion\Internet Settings`

var (
	pacServer *http.Server
	pacPort   int
)

func GeneratePacFile(targetDir string, domains []string, socksPort int) (string, error) {
	pacPath := filepath.Join(targetDir, "proxy.pac")

	var sb strings.Builder
	sb.WriteString("function FindProxyForURL(url, host) {\n")
	sb.WriteString("    if (shExpMatch(host, '*.onion')) { return 'SOCKS 127.0.0.1:" + fmt.Sprintf("%d", socksPort) + "'; }\n")

	for _, domain := range domains {
		d := strings.TrimSpace(domain)
		if d != "" {
			sb.WriteString(fmt.Sprintf("    if (shExpMatch(host, '*%s*')) { return 'SOCKS 127.0.0.1:%d'; }\n", d, socksPort))
		}
	}

	sb.WriteString("    return 'DIRECT';\n")
	sb.WriteString("}\n")

	err := os.WriteFile(pacPath, []byte(sb.String()), 0644)
	return pacPath, err
}

func SetupSystemPac(pacPath string) error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to allocate port for PAC server: %w", err)
	}
	pacPort = listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	mux.HandleFunc("/proxy.pac", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/x-ns-proxy-autoconfig")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		http.ServeFile(w, r, pacPath)
	})

	pacServer = &http.Server{Handler: mux}

	go func() {
		_ = pacServer.Serve(listener)
	}()

	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.SET_VALUE)
	if err != nil {
		_ = StopPacServer()
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer k.Close()

	httpUrl := fmt.Sprintf("http://127.0.0.1:%d/proxy.pac", pacPort)
	if err := k.SetStringValue("AutoConfigURL", httpUrl); err != nil {
		_ = StopPacServer()
		return fmt.Errorf("failed to set AutoConfigURL: %w", err)
	}

	notifySystemOptionsChanged()
	return nil
}

func StopPacServer() error {
	if pacServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = pacServer.Shutdown(ctx)
		pacServer = nil
	}
	return nil
}

func OffSystemPac() error {
	_ = StopPacServer()

	k, err := registry.OpenKey(registry.CURRENT_USER, internetSettingsPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("failed to open registry key: %w", err)
	}
	defer k.Close()

	_ = k.DeleteValue("AutoConfigURL")

	notifySystemOptionsChanged()
	return nil
}

func notifySystemOptionsChanged() {
	modwininet := syscall.NewLazyDLL("wininet.dll")
	procInternetSetOption := modwininet.NewProc("InternetSetOptionW")
	_, _, _ = procInternetSetOption.Call(0, 39, 0, 0)
	_, _, _ = procInternetSetOption.Call(0, 37, 0, 0)
}
