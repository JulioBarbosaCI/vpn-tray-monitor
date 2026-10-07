//go:build windows

package winsys

import (
	"os"

	"golang.org/x/sys/windows"
)

// attachParentProcess e o valor (DWORD)-1 que AttachConsole interpreta como
// "use o console do processo que me chamou".
const attachParentProcess = ^uintptr(0)

// AttachParentConsole religa a saida padrao ao console que chamou o programa.
//
// O executavel e compilado como aplicacao grafica (-H=windowsgui) para que o
// monitor residente nao abra uma janela preta. O efeito colateral e que os
// modos de linha de comando (-check, -set-password) ficariam mudos. Anexar o
// console do processo pai devolve a saida ao terminal do usuario, sem
// reintroduzir a janela no modo residente.
func AttachParentConsole() {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	if r, _, _ := kernel32.NewProc("AttachConsole").Call(attachParentProcess); r == 0 {
		return // nao havia console pai: execucao por atalho ou pelo Explorer
	}
	if f, err := os.OpenFile("CONOUT$", os.O_RDWR, 0); err == nil {
		os.Stdout = f
		os.Stderr = f
	}
	if f, err := os.OpenFile("CONIN$", os.O_RDWR, 0); err == nil {
		os.Stdin = f
	}
}
