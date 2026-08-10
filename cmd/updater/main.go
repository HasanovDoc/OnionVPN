package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"OnionVPN/internal/updater"
)

type Config struct {
	OldExe  string
	NewExe  string
	LogFile string
}

func main() {
	cfg, err := parseArgs()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	logFile, err := os.OpenFile(
		cfg.LogFile,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644,
	)

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	defer logFile.Close()
	logger := log.New(logFile, "", log.LstdFlags)
	logger.Println("===================================")
	logger.Println("Updater started")
	err = updater.Run(
		cfg.OldExe,
		cfg.NewExe,
		logger,
	)

	if err != nil {
		logger.Println("ERROR:", err)
		os.Exit(1)
	}

	logger.Println("Update completed successfully")
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
		return nil, fmt.Errorf("old exe is required")
	}

	if cfg.NewExe == "" {
		return nil, fmt.Errorf("new exe is required")
	}

	if cfg.LogFile == "" {
		cfg.LogFile = filepath.Join(
			os.TempDir(),
			"OnionVPNUpdate.log",
		)
	}

	cfg.OldExe, _ = filepath.Abs(cfg.OldExe)
	cfg.NewExe, _ = filepath.Abs(cfg.NewExe)
	cfg.LogFile, _ = filepath.Abs(cfg.LogFile)

	return cfg, nil
}
