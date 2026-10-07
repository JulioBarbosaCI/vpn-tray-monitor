//go:build !windows

package winsys

import (
	"encoding/base64"
	"errors"
)

// Fora do Windows nao existe DPAPI. Estas implementacoes servem apenas para
// compilar e testar a logica de arquivo em ambiente de desenvolvimento; elas
// NAO protegem nada e por isso o binario de producao e sempre o de Windows.
func protect(secret string) ([]byte, error) {
	return []byte(base64.StdEncoding.EncodeToString([]byte(secret))), nil
}

func unprotect(blob []byte) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(string(blob))
	if err != nil {
		return "", errors.New("segredo corrompido")
	}
	return string(raw), nil
}
