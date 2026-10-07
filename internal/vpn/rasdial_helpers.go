package vpn

import "github.com/guibsu/vpn-tray-monitor/internal/config"

// rasdialConfigFor monta a configuracao minima de uma conexao rasdial sem
// credenciais, que e o caso quando o Windows ja guarda usuario e senha.
func rasdialConfigFor(profile string) config.Connect {
	return config.Connect{
		Kind:    config.ConnectRasdial,
		Profile: profile,
	}
}
