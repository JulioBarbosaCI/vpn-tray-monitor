//go:build windows

package vpn

import (
	"os"
	"path/filepath"
)

// rasPhonebookPaths lista as agendas de conexao do Windows, da mais especifica
// para a mais geral: a do usuario primeiro, depois a compartilhada da maquina.
func rasPhonebookPaths() []string {
	const sufixo = `Microsoft\Network\Connections\Pbk\rasphone.pbk`

	var caminhos []string
	if appData := os.Getenv("APPDATA"); appData != "" {
		caminhos = append(caminhos, filepath.Join(appData, sufixo))
	}
	if programData := os.Getenv("PROGRAMDATA"); programData != "" {
		caminhos = append(caminhos, filepath.Join(programData, sufixo))
	}
	return caminhos
}
