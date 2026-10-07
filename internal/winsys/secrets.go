// Package winsys concentra o que depende de APIs do Windows: cofre de senhas,
// inicializacao automatica e notificacoes.
package winsys

import (
	"fmt"
	"os"
	"path/filepath"
)

// SecretStore guarda a senha da VPN cifrada em disco.
//
// A senha e protegida pela DPAPI do Windows, amarrada ao usuario que a gravou:
// o arquivo copiado para outra maquina ou aberto por outro usuario nao pode ser
// decifrado. Isso evita a senha em texto plano no config.json, que e o erro
// mais comum nesse tipo de utilitario.
type SecretStore struct {
	dir string
}

// NewSecretStore aponta o cofre para a pasta informada.
func NewSecretStore(dir string) *SecretStore {
	return &SecretStore{dir: dir}
}

// secretPath monta o caminho do arquivo de um segredo, recusando nomes que
// tentem escapar da pasta do cofre.
func (s *SecretStore) secretPath(name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("o nome do segredo nao pode ser vazio")
	}
	if name != filepath.Base(name) || name == "." || name == ".." {
		return "", fmt.Errorf("nome de segredo invalido: %q", name)
	}
	return filepath.Join(s.dir, name+".secret"), nil
}

// Set cifra e grava a senha sob o nome informado.
func (s *SecretStore) Set(name, secret string) error {
	path, err := s.secretPath(name)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return fmt.Errorf("criando a pasta do cofre: %w", err)
	}
	blob, err := protect(secret)
	if err != nil {
		return fmt.Errorf("cifrando o segredo: %w", err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		return fmt.Errorf("gravando o segredo: %w", err)
	}
	return nil
}

// Secret le e decifra a senha gravada. Satisfaz vpn.SecretResolver.
func (s *SecretStore) Secret(name string) (string, error) {
	path, err := s.secretPath(name)
	if err != nil {
		return "", err
	}
	blob, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("segredo %q nao foi gravado; rode: vpnmon.exe -set-password %s", name, name)
		}
		return "", fmt.Errorf("lendo o segredo: %w", err)
	}
	secret, err := unprotect(blob)
	if err != nil {
		return "", fmt.Errorf("decifrando o segredo %q (foi gravado por outro usuario?): %w", name, err)
	}
	return secret, nil
}

// Has informa se o segredo ja foi gravado.
func (s *SecretStore) Has(name string) bool {
	path, err := s.secretPath(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}
