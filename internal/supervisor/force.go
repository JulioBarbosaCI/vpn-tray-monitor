package supervisor

import "context"

// ForceReconnect executa o comando de conexao imediatamente, a pedido do
// usuario pelo menu, sem esperar o limite de falhas consecutivas. Roda em
// segundo plano para nao travar a bandeja enquanto a conexao e negociada.
func (s *Supervisor) ForceReconnect() {
	go func() {
		s.log.Infof("reconexao solicitada pelo usuario")
		s.reconnect(context.Background())
		s.CheckNow()
	}()
}
