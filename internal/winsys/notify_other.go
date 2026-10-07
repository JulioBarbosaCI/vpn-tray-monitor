//go:build !windows

package winsys

// Notify nao faz nada fora do Windows; existe para o projeto compilar em
// ambiente de desenvolvimento.
func Notify(_, _ string) {}
