// Package supervisor mantem a VPN conectada: verifica, decide e reconecta.
package supervisor

import "time"

// State e o estado observavel da VPN, exibido no icone da bandeja.
type State int

const (
	// StateUnknown vale antes da primeira verificacao concluir.
	StateUnknown State = iota
	// StateConnected: a ultima verificacao respondeu que a VPN esta no ar.
	StateConnected
	// StateDisconnected: a VPN caiu e o monitor vai tentar reconectar.
	StateDisconnected
	// StateConnecting: uma tentativa de reconexao esta em andamento.
	StateConnecting
	// StateError: a propria verificacao nao pode ser executada
	// (binario ausente, permissao negada). Nao e o mesmo que VPN fora.
	StateError
	// StatePaused: o usuario suspendeu o monitoramento pelo menu.
	StatePaused
)

// String devolve o rotulo em portugues exibido na interface.
func (s State) String() string {
	switch s {
	case StateConnected:
		return "Conectada"
	case StateDisconnected:
		return "Desconectada"
	case StateConnecting:
		return "Conectando..."
	case StateError:
		return "Erro na verificacao"
	case StatePaused:
		return "Monitoramento pausado"
	default:
		return "Verificando..."
	}
}

// Snapshot e a fotografia do monitor entregue a interface.
type Snapshot struct {
	State            State
	Since            time.Time
	LastCheck        time.Time
	LastError        string
	ConsecutiveFails int
	ReconnectCount   int
	NextAttemptIn    time.Duration
}
