package tor

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

//go:embed bin/windows/*
var torBinaries embed.FS

func ExtractBinaries(targetDir string) error {
	baseEmbedPath := "bin/windows"

	return extractDir(baseEmbedPath, targetDir)
}

func extractDir(srcDir, destDir string) error {
	entries, err := torBinaries.ReadDir(srcDir)
	if err != nil {
		return fmt.Errorf("failed to read embed dir %s: %w", srcDir, err)
	}

	for _, entry := range entries {
		srcPath := srcDir + "/" + entry.Name()
		relPath := strings.TrimPrefix(srcPath, "bin/windows/")
		destPath := filepath.Join(destDir, relPath)

		if entry.IsDir() {
			if err := os.MkdirAll(destPath, 0755); err != nil {
				return fmt.Errorf("failed to create dir %s: %w", destPath, err)
			}
			if err := extractDir(srcPath, destDir); err != nil {
				return err
			}
		} else {
			if err := extractFile(srcPath, destPath); err != nil {
				return err
			}
		}
	}
	return nil
}

func extractFile(srcPath, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	srcFile, err := torBinaries.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open embed file %s: %w", srcPath, err)
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create target file %s: %w", destPath, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("failed to write file %s: %w", destPath, err)
	}

	return nil
}

func GenerateTorrc(baseDir string, bridges []string) (string, error) {
	torrcPath := filepath.Join(baseDir, "torrc")

	// dataDir := filepath.ToSlash(filepath.Join(baseDir, "data"))
	lyrebirdPath := filepath.ToSlash(filepath.Join(baseDir, "tor", "pluggable_transports", "lyrebird.exe"))
	// conjurePath := filepath.ToSlash(filepath.Join(baseDir, "tor", "pluggable_transports", "conjure-client.exe"))

	var sb strings.Builder

	sb.WriteString("SocksPort 127.0.0.1:9050\n\n")
	sb.WriteString("ClientUseIPv6 0\n")
	sb.WriteString("ClientUseIPv4 1\n\n")
	sb.WriteString("UseBridges 1\n\n")
	sb.WriteString(fmt.Sprintf("ClientTransportPlugin obfs4 exec %s\n", lyrebirdPath))
	sb.WriteString(fmt.Sprintf("ClientTransportPlugin webtunnel exec %s\n", lyrebirdPath))

	if len(bridges) > 0 {
		for _, bridge := range bridges {
			bridgeLine := strings.TrimSpace(bridge)
			if bridgeLine != "" {
				if !strings.HasPrefix(bridgeLine, "Bridge ") {
					sb.WriteString(fmt.Sprintf("Bridge %s\n", bridgeLine))
				} else {
					sb.WriteString(fmt.Sprintf("%s\n", bridgeLine))
				}
			}
		}
	}

	err := os.WriteFile(torrcPath, []byte(sb.String()), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write torrc: %w", err)
	}

	return torrcPath, nil
}
