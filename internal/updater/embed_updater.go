package updater

import (
	_ "embed"
)

//go:embed bin/windows/OnionVPNUpdater.exe
var UpdaterBinary []byte
