//go:build !windows

package winsys

import "errors"

// Fora do Windows nao ha registro. Serve so para o projeto compilar em
// ambiente de desenvolvimento.
func SetAutoStart(bool) error {
	return errors.New("inicializacao automatica so e suportada no Windows")
}

func IsAutoStartEnabled() bool { return false }
