//go:build !windows

package vpn

import "os"

// rasPhonebookPaths permite apontar uma agenda de teste fora do Windows,
// para exercitar a descoberta de perfis em desenvolvimento.
func rasPhonebookPaths() []string {
	if caminho := os.Getenv("VPNMON_PBK"); caminho != "" {
		return []string{caminho}
	}
	return nil
}
