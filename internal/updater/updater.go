package updater

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

func Run(oldExe, newExe string, logger *log.Logger) error {
	logger.Println("Waiting until executable becomes unlocked...")

	if err := waitUntilUnlocked(oldExe, logger); err != nil {
		return err
	}

	logger.Println("Executable unlocked.")
	backup := oldExe + ".bak"
	logger.Println("Creating backup...")

	if err := backupOldExe(oldExe, backup); err != nil {
		return err
	}

	logger.Println("Replacing executable...")

	if err := replaceExe(newExe, oldExe); err != nil {
		logger.Println(err)
		_ = os.Remove(oldExe + ".tmp")

		if _, statErr := os.Stat(oldExe); os.IsNotExist(statErr) {
			logger.Println("Executable missing. Restoring backup...")
			_ = restoreBackup(backup, oldExe)
		}

		return err
	}

	logger.Println("Verifying copied file...")

	if err := verifyCopy(oldExe, newExe); err != nil {
		logger.Println("Verification failed. Rolling back...")
		_ = os.Remove(oldExe)
		_ = restoreBackup(backup, oldExe)
		return err
	}

	logger.Println("Starting new version...")

	if err := startNewVersion(oldExe); err != nil {
		return err
	}

	logger.Println("Cleaning temporary files...")
	cleanup(newExe, backup)
	logger.Println("Done.")

	return nil
}

func waitUntilUnlocked(exe string, logger *log.Logger) error {
	const timeout = 60 * time.Second
	start := time.Now()
	tmp := exe + ".locktest"

	for {
		_ = os.Remove(tmp)
		err := os.Rename(exe, tmp)

		if err == nil {
			err = os.Rename(tmp, exe)

			if err == nil {
				return nil
			}
		}

		if time.Since(start) > timeout {
			return fmt.Errorf("timeout waiting executable unlock")
		}

		logger.Println("Waiting...")
		time.Sleep(300 * time.Millisecond)
	}
}

func backupOldExe(oldExe, backup string) error {
	_ = os.Remove(backup)
	src, err := os.Open(oldExe)

	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(backup)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return dst.Sync()
}

func restoreBackup(backup, oldExe string) error {
	_ = os.Remove(oldExe)
	src, err := os.Open(backup)

	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(oldExe)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return err
	}

	return dst.Sync()
}

func replaceExe(newExe, oldExe string) error {
	tmpExe := oldExe + ".tmp"
	_ = os.Remove(tmpExe)
	src, err := os.Open(newExe)

	if err != nil {
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(
		tmpExe,
		os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
		0755,
	)

	if err != nil {
		return err
	}

	if _, err = io.Copy(dst, src); err != nil {
		dst.Close()
		_ = os.Remove(tmpExe)
		return err
	}

	if err = dst.Sync(); err != nil {
		dst.Close()
		_ = os.Remove(tmpExe)
		return err
	}

	if err = dst.Close(); err != nil {
		_ = os.Remove(tmpExe)
		return err
	}

	if err = verifyCopy(tmpExe, newExe); err != nil {
		_ = os.Remove(tmpExe)
		return err
	}

	if err = os.Remove(oldExe); err != nil {
		_ = os.Remove(tmpExe)
		return err
	}

	if err = os.Rename(tmpExe, oldExe); err != nil {
		_ = os.Remove(tmpExe)
		return err
	}

	return nil
}

func verifyCopy(oldExe, newExe string) error {
	oldInfo, err := os.Stat(oldExe)
	if err != nil {
		return err
	}

	newInfo, err := os.Stat(newExe)
	if err != nil {
		return err
	}

	if oldInfo.Size() != newInfo.Size() {

		return fmt.Errorf(
			"copied file size mismatch (%d != %d)",
			oldInfo.Size(),
			newInfo.Size(),
		)

	}

	return nil
}

func startNewVersion(exe string) error {
	cmd := exec.Command(exe)
	cmd.Dir = filepath.Dir(exe)

	if err := cmd.Start(); err != nil {
		return err
	}

	return nil
}

func cleanup(newExe, backup string) {
	_ = os.Remove(newExe)
	_ = os.Remove(backup)
}
