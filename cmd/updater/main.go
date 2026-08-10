package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type Config struct {
	OldExe  string
	NewExe  string
	LogFile string
}

func main() {
	cfg, err := parseArgs()
	if err != nil {
		fmt.Println("Updater error:", err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(
		cfg.LogFile,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)
	if err != nil {
		fmt.Println("Failed to open log:", err)
		os.Exit(1)
	}
	defer logFile.Close()

	logger := log.New(logFile, "", log.LstdFlags)
	logger.Println("===================================")
	logger.Println("OnionVPN Updater started")
	logger.Printf("Old executable: %s", cfg.OldExe)
	logger.Printf("New executable: %s", cfg.NewExe)

	if err := runUpdate(cfg, logger); err != nil {
		logger.Println("UPDATE FAILED:", err)
		os.Exit(1)
	}

	logger.Println("UPDATE COMPLETED SUCCESSFULLY")
}

func parseArgs() (*Config, error) {
	cfg := &Config{}

	args := os.Args[1:]

	for i := 0; i < len(args); i++ {
		switch args[i] {

		case "--old":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --old")
			}

			cfg.OldExe = args[i+1]
			i++

		case "--new":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --new")
			}

			cfg.NewExe = args[i+1]
			i++

		case "--log":
			if i+1 >= len(args) {
				return nil, fmt.Errorf("missing value for --log")
			}

			cfg.LogFile = args[i+1]
			i++

		default:
			return nil, fmt.Errorf("unknown argument: %s", args[i])
		}
	}

	if cfg.OldExe == "" {
		return nil, fmt.Errorf("--old is required")
	}

	if cfg.NewExe == "" {
		return nil, fmt.Errorf("--new is required")
	}

	if cfg.LogFile == "" {
		cfg.LogFile = filepath.Join(
			os.TempDir(),
			"OnionVPNUpdate.log",
		)
	}

	oldExe, err := filepath.Abs(cfg.OldExe)
	if err != nil {
		return nil, fmt.Errorf("invalid old executable path: %w", err)
	}

	newExe, err := filepath.Abs(cfg.NewExe)
	if err != nil {
		return nil, fmt.Errorf("invalid new executable path: %w", err)
	}

	logFile, err := filepath.Abs(cfg.LogFile)
	if err != nil {
		return nil, fmt.Errorf("invalid log path: %w", err)
	}

	cfg.OldExe = oldExe
	cfg.NewExe = newExe
	cfg.LogFile = logFile

	return cfg, nil
}

func runUpdate(cfg *Config, logger *log.Logger) error {

	logger.Println("Waiting for OnionVPN.exe to exit...")

	if err := waitUntilUnlocked(cfg.OldExe, logger); err != nil {
		return err
	}

	logger.Println("OnionVPN.exe is no longer locked.")

	newInfo, err := os.Stat(cfg.NewExe)
	if err != nil {
		return fmt.Errorf("new executable does not exist: %w", err)
	}

	if newInfo.Size() == 0 {
		return fmt.Errorf("new executable is empty")
	}

	logger.Printf("New executable size: %d bytes", newInfo.Size())

	backup := cfg.OldExe + ".bak"

	logger.Println("Creating backup...")

	if err := backupOldExe(cfg.OldExe, backup); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	logger.Printf("Backup created: %s", backup)

	logger.Println("Replacing executable...")

	if err := replaceExe(cfg.NewExe, cfg.OldExe); err != nil {
		logger.Printf("Replacement failed: %v", err)
		logger.Println("Attempting rollback...")

		if restoreErr := restoreBackup(backup, cfg.OldExe); restoreErr != nil {
			return fmt.Errorf(
				"replacement failed (%v), rollback failed (%v)",
				err,
				restoreErr,
			)
		}

		return fmt.Errorf("replacement failed: %w", err)
	}

	logger.Println("Executable replaced.")

	logger.Println("Verifying installed executable...")

	if err := verifyCopy(cfg.OldExe, cfg.NewExe); err != nil {
		logger.Printf("Verification failed: %v", err)
		logger.Println("Attempting rollback...")

		_ = os.Remove(cfg.OldExe)

		if restoreErr := restoreBackup(backup, cfg.OldExe); restoreErr != nil {
			return fmt.Errorf(
				"verification failed (%v), rollback failed (%v)",
				err,
				restoreErr,
			)
		}

		return fmt.Errorf("verification failed: %w", err)
	}

	logger.Println("Verification successful.")
	logger.Println("Starting new OnionVPN version...")

	if err := startNewVersion(cfg.OldExe); err != nil {
		logger.Printf("Failed to start new version: %v", err)
		logger.Println("Attempting rollback...")

		_ = os.Remove(cfg.OldExe)

		if restoreErr := restoreBackup(backup, cfg.OldExe); restoreErr != nil {
			return fmt.Errorf(
				"new version failed to start (%v), rollback failed (%v)",
				err,
				restoreErr,
			)
		}

		return fmt.Errorf("failed to start new version: %w", err)
	}

	logger.Println("New version started successfully.")

	if err := os.Remove(cfg.NewExe); err != nil {
		logger.Printf(
			"Warning: failed to remove temporary update file: %v",
			err,
		)
	}

	if err := os.Remove(backup); err != nil {
		logger.Printf(
			"Warning: failed to remove backup: %v",
			err,
		)
	}

	logger.Println("Cleanup completed.")

	return nil
}

func waitUntilUnlocked(exe string, logger *log.Logger) error {

	const timeout = 60 * time.Second
	const interval = 300 * time.Millisecond

	start := time.Now()

	for {
		file, err := os.OpenFile(
			exe,
			os.O_RDWR,
			0,
		)

		if err == nil {
			_ = file.Close()
			return nil
		}

		if time.Since(start) >= timeout {
			return fmt.Errorf(
				"timeout waiting for executable to unlock: %s",
				exe,
			)
		}

		logger.Println("Executable is still locked, waiting...")

		time.Sleep(interval)
	}
}

func backupOldExe(oldExe, backup string) error {

	_ = os.Remove(backup)

	src, err := os.Open(oldExe)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(
		backup,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0755,
	)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}

	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		return err
	}

	return dst.Close()
}

func restoreBackup(backup, oldExe string) error {
	src, err := os.Open(backup)
	if err != nil {
		return err
	}
	defer src.Close()

	_ = os.Remove(oldExe)

	dst, err := os.OpenFile(
		oldExe,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0755,
	)
	if err != nil {
		return err
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		return err
	}

	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		return err
	}

	return dst.Close()
}

func replaceExe(newExe, oldExe string) error {
	tmpExe := oldExe + ".tmp"
	_ = os.Remove(tmpExe)

	src, err := os.Open(newExe)
	if err != nil {
		return fmt.Errorf("failed to open new executable: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(
		tmpExe,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0755,
	)
	if err != nil {
		return fmt.Errorf("failed to create temporary executable: %w", err)
	}

	if _, err := io.Copy(dst, src); err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpExe)

		return fmt.Errorf(
			"failed to copy new executable: %w",
			err,
		)
	}

	if err := dst.Sync(); err != nil {
		_ = dst.Close()
		_ = os.Remove(tmpExe)

		return fmt.Errorf(
			"failed to flush temporary executable: %w",
			err,
		)
	}

	if err := dst.Close(); err != nil {
		_ = os.Remove(tmpExe)

		return fmt.Errorf(
			"failed to close temporary executable: %w",
			err,
		)
	}

	if err := verifyCopy(tmpExe, newExe); err != nil {
		_ = os.Remove(tmpExe)

		return fmt.Errorf(
			"temporary executable verification failed: %w",
			err,
		)
	}

	if err := os.Remove(oldExe); err != nil {
		_ = os.Remove(tmpExe)

		return fmt.Errorf(
			"failed to remove old executable: %w",
			err,
		)
	}

	if err := os.Rename(tmpExe, oldExe); err != nil {
		_ = os.Remove(tmpExe)

		return fmt.Errorf(
			"failed to rename temporary executable: %w",
			err,
		)
	}

	return nil
}

func verifyCopy(installedExe, sourceExe string) error {
	installedInfo, err := os.Stat(installedExe)
	if err != nil {
		return fmt.Errorf(
			"failed to stat installed executable: %w",
			err,
		)
	}

	sourceInfo, err := os.Stat(sourceExe)
	if err != nil {
		return fmt.Errorf(
			"failed to stat source executable: %w",
			err,
		)
	}

	if installedInfo.Size() != sourceInfo.Size() {
		return fmt.Errorf(
			"file size mismatch: installed=%d, source=%d",
			installedInfo.Size(),
			sourceInfo.Size(),
		)
	}

	if installedInfo.Size() == 0 {
		return fmt.Errorf("installed executable is empty")
	}

	return nil
}

func startNewVersion(exe string) error {
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf(
			"failed to start %s: %w",
			exe,
			err,
		)
	}

	return nil
}
