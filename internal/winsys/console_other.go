//go:build !windows

package winsys

// AttachParentConsole nao e necessario fora do Windows: o terminal ja esta
// conectado a saida padrao.
func AttachParentConsole() {}
