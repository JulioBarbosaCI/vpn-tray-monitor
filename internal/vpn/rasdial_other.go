//go:build !windows

package vpn

import "errors"

// dialWindowsVPN so existe no Windows; fora dele o projeto apenas compila
// para permitir rodar os testes em desenvolvimento.
func dialWindowsVPN(_, _, _ string) error {
	return errors.New("a conexao VPN nativa so funciona no Windows")
}
