//go:build windows

package winsys

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

// runKeyPath e a chave que o Windows le no login do usuario. Usar a chave do
// usuario (HKCU) e nao a da maquina evita exigir privilegio de administrador.
const (
	runKeyPath = `Software\Microsoft\Windows\CurrentVersion\Run`
	runValue   = "VPNTrayMonitor"
)

// SetAutoStart liga ou desliga a inicializacao junto com o Windows.
func SetAutoStart(enabled bool) error {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.SET_VALUE)
	if err != nil {
		return fmt.Errorf("abrindo a chave de inicializacao: %w", err)
	}
	defer key.Close()

	if !enabled {
		err := key.DeleteValue(runValue)
		if err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("removendo a inicializacao automatica: %w", err)
		}
		return nil
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("localizando o executavel: %w", err)
	}
	// As aspas protegem caminhos com espaco, como "C:\Arquivos de Programas".
	if err := key.SetStringValue(runValue, `"`+exe+`"`); err != nil {
		return fmt.Errorf("gravando a inicializacao automatica: %w", err)
	}
	return nil
}

// IsAutoStartEnabled informa se o programa ja esta registrado no login.
func IsAutoStartEnabled() bool {
	key, err := registry.OpenKey(registry.CURRENT_USER, runKeyPath, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer key.Close()

	value, _, err := key.GetStringValue(runValue)
	return err == nil && value != ""
}
