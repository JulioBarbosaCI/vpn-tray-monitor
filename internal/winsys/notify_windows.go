//go:build windows

package winsys

import "fyne.io/systray"

// Notify mostra um balao na area de notificacao. A falha em notificar nao
// pode derrubar o monitor: perder um aviso e menos grave que perder a VPN.
func Notify(titulo, mensagem string) {
	defer func() { _ = recover() }()
	systray.SetTooltip(titulo + ": " + mensagem)
}
