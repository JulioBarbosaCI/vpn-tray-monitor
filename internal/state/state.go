// Package state guarda as escolhas que o usuario faz pelo menu da bandeja,
// para que sobrevivam ao fechar e abrir o programa.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileName e o arquivo gravado ao lado do executavel.
const FileName = "vpnmon.state.json"

const filePerm = 0o600

// State e o que o programa lembra entre execucoes.
type State struct {
	// VPNProfile e a conexao escolhida no menu da bandeja.
	VPNProfile string `json:"vpnProfile"`
}

// Path monta o caminho do arquivo de estado dentro da pasta informada.
func Path(dir string) string {
	return filepath.Join(dir, FileName)
}

// Load le o estado gravado. A ausencia do arquivo nao e erro: significa apenas
// que o usuario ainda nao escolheu nada.
func Load(path string) (State, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("lendo %s: %w", path, err)
	}

	var s State
	if err := json.Unmarshal(raw, &s); err != nil {
		// Estado corrompido nao pode impedir o programa de subir: ele volta ao
		// padrao e o usuario reescolhe pelo menu.
		return State{}, nil
	}
	return s, nil
}

// Save grava o estado, criando a pasta se preciso.
func Save(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("criando a pasta do estado: %w", err)
	}

	raw, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("serializando o estado: %w", err)
	}
	if err := os.WriteFile(path, raw, filePerm); err != nil {
		return fmt.Errorf("gravando %s: %w", path, err)
	}
	return nil
}
