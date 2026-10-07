package vpn

// ListWindowsVPNs devolve os nomes das conexoes VPN cadastradas no Windows,
// na ordem em que aparecem nas agendas do usuario e da maquina.
//
// A bandeja usa esta lista para deixar o usuario escolher a VPN pelo menu, em
// vez de exigir que ele digite o nome exato em algum lugar.
func ListWindowsVPNs() []string {
	return discoverRasProfiles()
}

// NewRasdialFor monta um reconector para a conexao informada, usado quando o
// usuario troca de VPN pelo menu sem reiniciar o programa.
func NewRasdialFor(profile string) Connector {
	return rasdialConnector{cfg: rasdialConfigFor(profile)}
}
