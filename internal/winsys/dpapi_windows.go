//go:build windows

package winsys

import "github.com/billgraziano/dpapi"

// protect cifra com a DPAPI do usuario atual.
func protect(secret string) ([]byte, error) {
	blob, err := dpapi.Encrypt(secret)
	if err != nil {
		return nil, err
	}
	return []byte(blob), nil
}

// unprotect decifra o que protect gravou.
func unprotect(blob []byte) (string, error) {
	return dpapi.Decrypt(string(blob))
}
