package supervisor

import "github.com/guibsu/vpn-tray-monitor/internal/vpn"

// SetConnector troca o reconector em execucao, sem reiniciar o programa.
// Usado quando o usuario escolhe outra VPN pelo menu da bandeja.
func (s *Supervisor) SetConnector(c vpn.Connector) {
	s.mu.Lock()
	s.conn = c
	s.mu.Unlock()
	s.log.Infof("conexao alterada para %s", c.Describe())
}

// connector devolve o reconector atual de forma segura para leitura
// concorrente, ja que o menu pode troca-lo enquanto o ciclo roda.
func (s *Supervisor) connector() vpn.Connector {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.conn
}
