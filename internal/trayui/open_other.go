//go:build !windows

package trayui

import "os/exec"

// openPath usa o abridor padrao do desktop, para desenvolvimento em Linux.
func openPath(path string) error {
	return exec.Command("xdg-open", path).Start()
}
