package supervisor

import (
	"context"
	"sync"
	"time"

	"github.com/guibsu/vpn-tray-monitor/internal/config"
	"github.com/guibsu/vpn-tray-monitor/internal/vpn"
)

// Logger e o minimo que o supervisor precisa registrar.
type Logger interface {
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
}

// Observer recebe cada mudanca de estado, para atualizar icone e notificacoes.
type Observer func(Snapshot)

// Supervisor executa o ciclo verificar -> decidir -> reconectar.
type Supervisor struct {
	cfg     config.Config
	checker vpn.Checker
	conn    vpn.Connector
	log     Logger
	clock   Clock

	mu       sync.RWMutex
	snapshot Snapshot
	paused   bool

	observers []Observer
	forceNow  chan struct{}
}

// Clock isola a passagem do tempo para que os testes nao precisem esperar de verdade.
type Clock interface {
	Now() time.Time
	Sleep(ctx context.Context, d time.Duration) bool
}

// New monta um supervisor pronto para rodar.
func New(cfg config.Config, checker vpn.Checker, conn vpn.Connector, log Logger, clock Clock) *Supervisor {
	if clock == nil {
		clock = realClock{}
	}
	return &Supervisor{
		cfg:      cfg,
		checker:  checker,
		conn:     conn,
		log:      log,
		clock:    clock,
		snapshot: Snapshot{State: StateUnknown, Since: clock.Now()},
		forceNow: make(chan struct{}, 1),
	}
}

// Subscribe registra um observador chamado a cada mudanca de estado.
func (s *Supervisor) Subscribe(o Observer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.observers = append(s.observers, o)
}

// Snapshot devolve o estado atual, seguro para leitura concorrente.
func (s *Supervisor) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot
}

// SetPaused suspende ou retoma o monitoramento a pedido do usuario.
func (s *Supervisor) SetPaused(paused bool) {
	s.mu.Lock()
	s.paused = paused
	s.mu.Unlock()

	if paused {
		s.setState(StatePaused, "")
		s.log.Infof("monitoramento pausado pelo usuario")
		return
	}
	s.log.Infof("monitoramento retomado pelo usuario")
	s.CheckNow()
}

// CheckNow interrompe a espera e forca uma verificacao imediata.
func (s *Supervisor) CheckNow() {
	select {
	case s.forceNow <- struct{}{}:
	default: // ja existe um pedido pendente; um basta
	}
}

// Run executa o ciclo ate o contexto ser cancelado.
func (s *Supervisor) Run(ctx context.Context) {
	s.log.Infof("monitorando %s (%s), intervalo de %s",
		s.cfg.DisplayName, s.checker.Describe(), s.cfg.CheckInterval())

	bo := newBackoff(s.cfg.CheckInterval(), s.cfg.MaxBackoff())

	for {
		if ctx.Err() != nil {
			s.log.Infof("monitoramento encerrado")
			return
		}

		wait := s.cfg.CheckInterval()
		if !s.isPaused() {
			wait = s.runCycle(ctx, bo)
		}

		s.setNextAttempt(wait)
		if !s.waitOrWake(ctx, wait) {
			s.log.Infof("monitoramento encerrado")
			return
		}
	}
}

// runCycle faz uma verificacao e, se preciso, uma tentativa de reconexao.
// Devolve quanto esperar antes do proximo ciclo.
func (s *Supervisor) runCycle(ctx context.Context, bo *backoff) time.Duration {
	checkCtx, cancel := context.WithTimeout(ctx, s.cfg.Check.Timeout())
	connected, err := s.checker.Check(checkCtx)
	cancel()

	s.markChecked()

	if err != nil {
		// A verificacao nao pode ser executada. Nao presuma que a VPN caiu:
		// reconectar as cegas por causa de um erro de ferramenta so piora.
		s.setState(StateError, err.Error())
		s.log.Errorf("nao foi possivel verificar a VPN: %v", err)
		return bo.next()
	}

	if connected {
		if s.previousState() != StateConnected {
			s.log.Infof("VPN conectada")
		}
		s.resetFailures()
		s.setState(StateConnected, "")
		bo.reset()
		return s.cfg.CheckInterval()
	}

	fails := s.incrementFailures()
	s.setState(StateDisconnected, "")

	if fails < s.cfg.FailuresToReconnect() {
		// Uma falha isolada pode ser oscilacao momentanea. Confirme antes de agir.
		s.log.Warnf("VPN sem resposta (%d de %d verificacoes)", fails, s.cfg.FailuresToReconnect())
		return s.cfg.CheckInterval()
	}

	s.reconnect(ctx)
	return bo.next()
}

// reconnect executa uma tentativa de reconexao e registra o resultado.
func (s *Supervisor) reconnect(ctx context.Context) {
	conn := s.connector()

	s.setState(StateConnecting, "")
	s.log.Infof("reconectando via %s", conn.Describe())

	connCtx, cancel := context.WithTimeout(ctx, s.cfg.Connect.Timeout())
	defer cancel()

	if err := conn.Connect(connCtx); err != nil {
		s.setState(StateDisconnected, err.Error())
		s.log.Errorf("falha ao reconectar: %v", err)
		return
	}

	s.countReconnect()
	s.log.Infof("comando de conexao executado; aguardando %s para confirmar", s.cfg.GraceAfterConnect())
	s.clock.Sleep(ctx, s.cfg.GraceAfterConnect())
	s.resetFailures()
}

// waitOrWake espera o intervalo, mas acorda antes se o usuario pedir
// verificacao imediata. Devolve false quando o contexto foi cancelado.
func (s *Supervisor) waitOrWake(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-s.forceNow:
		return true
	case <-timer.C:
		return true
	}
}
