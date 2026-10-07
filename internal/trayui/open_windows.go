//go:build windows

package trayui

import (
	"os/exec"

	"golang.org/x/sys/windows"
)

// openPath entrega o arquivo ao programa associado no Windows.
func openPath(path string) error {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", path)
	cmd.SysProcAttr = &windows.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
	return cmd.Start()
}
