package trayui

import (
	"fmt"

	"github.com/guibsu/vpn-tray-monitor/internal/supervisor"
)

// detailFor monta a segunda linha do menu: o erro quando existe, senao a
// hora da ultima verificacao.
func detailFor(snap supervisor.Snapshot) string {
	if snap.LastError != "" {
		return truncate(snap.LastError, detalheMaxChars)
	}
	if snap.LastCheck.IsZero() {
		return ""
	}
	linha := "Ultima verificacao: " + snap.LastCheck.Format("15:04:05")
	if snap.ReconnectCount > 0 {
		linha += fmt.Sprintf("  |  reconexoes: %d", snap.ReconnectCount)
	}
	return linha
}

// truncate encurta textos longos para caberem no menu da bandeja.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	if max <= 3 {
		return s[:max]
	}
	return s[:max-3] + "..."
}
