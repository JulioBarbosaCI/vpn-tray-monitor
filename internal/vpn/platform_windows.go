//go:build windows

package vpn

import (
	"os/exec"

	"golang.org/x/sys/windows"
)

// hideWindow impede que cada verificacao pisque um console preto na tela do
// usuario. Sem isso o programa fica visualmente insuportavel rodando a cada 30s.
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &windows.SysProcAttr{
		HideWindow:    true,
		CreationFlags: windows.CREATE_NO_WINDOW,
	}
}
