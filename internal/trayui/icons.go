// Package trayui desenha o icone da bandeja e o menu de contexto.
package trayui

import (
	_ "embed"

	"github.com/guibsu/vpn-tray-monitor/internal/supervisor"
)

// Os icones sao embutidos no executavel para que o programa continue sendo um
// arquivo unico, sem pasta de recursos ao lado. Gere-os com: go run ./tools/geniconos
var (
	//go:embed icons/conectada.ico
	iconConectada []byte
	//go:embed icons/desconectada.ico
	iconDesconectada []byte
	//go:embed icons/conectando.ico
	iconConectando []byte
	//go:embed icons/inativa.ico
	iconInativa []byte
)

// iconFor devolve o icone correspondente ao estado atual.
func iconFor(state supervisor.State) []byte {
	switch state {
	case supervisor.StateConnected:
		return iconConectada
	case supervisor.StateDisconnected:
		return iconDesconectada
	case supervisor.StateConnecting:
		return iconConectando
	default:
		// Desconhecido, erro e pausado compartilham o icone neutro: em todos
		// eles o monitor nao esta afirmando nada sobre a VPN.
		return iconInativa
	}
}
