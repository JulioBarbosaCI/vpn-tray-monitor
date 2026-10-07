package supervisor

import (
	"context"
	"testing"
	"time"
)

// runCycles executa exatamente n ciclos do supervisor e devolve o controle.
// Evita depender do laco Run, que so termina por cancelamento.
func runCycles(t *testing.T, s *Supervisor, n int) {
	t.Helper()
	bo := newBackoff(s.cfg.CheckInterval(), s.cfg.MaxBackoff())
	for i := 0; i < n; i++ {
		s.runCycle(context.Background(), bo)
	}
}

func TestVPNNoArNaoDisparaReconexao(t *testing.T) {
	// Arrange
	checker := &fakeChecker{results: []checkResult{{connected: true}}}
	connector := &fakeConnector{}
	s := New(baseConfig(), checker, connector, silentLogger{}, newInstantClock())

	// Act
	runCycles(t, s, 3)

	// Assert
	if got := connector.attemptCount(); got != 0 {
		t.Fatalf("nao deveria reconectar com a VPN no ar; tentativas = %d", got)
	}
	if got := s.Snapshot().State; got != StateConnected {
		t.Fatalf("estado = %v, esperado %v", got, StateConnected)
	}
}

func TestUmaFalhaIsoladaNaoReconecta(t *testing.T) {
	// Arrange: uma unica falha, com o limite configurado em 2.
	checker := &fakeChecker{results: []checkResult{{connected: false}, {connected: true}}}
	connector := &fakeConnector{}
	s := New(baseConfig(), checker, connector, silentLogger{}, newInstantClock())

	// Act
	runCycles(t, s, 1)

	// Assert
	if got := connector.attemptCount(); got != 0 {
		t.Fatalf("uma falha isolada nao deveria reconectar; tentativas = %d", got)
	}
	if got := s.Snapshot().ConsecutiveFails; got != 1 {
		t.Fatalf("falhas consecutivas = %d, esperado 1", got)
	}
}

func TestFalhasSeguidasDisparamReconexao(t *testing.T) {
	// Arrange: duas falhas seguidas atingem o limite.
	checker := &fakeChecker{results: []checkResult{{connected: false}, {connected: false}}}
	connector := &fakeConnector{}
	s := New(baseConfig(), checker, connector, silentLogger{}, newInstantClock())

	// Act
	runCycles(t, s, 2)

	// Assert
	if got := connector.attemptCount(); got != 1 {
		t.Fatalf("tentativas de reconexao = %d, esperado 1", got)
	}
	if got := s.Snapshot().ReconnectCount; got != 1 {
		t.Fatalf("reconexoes contabilizadas = %d, esperado 1", got)
	}
}

func TestReconexaoBemSucedidaZeraContadorDeFalhas(t *testing.T) {
	// Arrange
	checker := &fakeChecker{results: []checkResult{{connected: false}, {connected: false}}}
	connector := &fakeConnector{}
	s := New(baseConfig(), checker, connector, silentLogger{}, newInstantClock())

	// Act
	runCycles(t, s, 2)

	// Assert
	if got := s.Snapshot().ConsecutiveFails; got != 0 {
		t.Fatalf("falhas consecutivas apos reconectar = %d, esperado 0", got)
	}
}

func TestErroDeVerificacaoNaoReconectaAsCegas(t *testing.T) {
	// Arrange: a verificacao nem pode ser executada. Isso nao prova que a VPN caiu.
	checker := &fakeChecker{results: []checkResult{{err: errCheckFailed}, {err: errCheckFailed}}}
	connector := &fakeConnector{}
	s := New(baseConfig(), checker, connector, silentLogger{}, newInstantClock())

	// Act
	runCycles(t, s, 2)

	// Assert
	if got := connector.attemptCount(); got != 0 {
		t.Fatalf("erro de verificacao nao deveria disparar reconexao; tentativas = %d", got)
	}
	if got := s.Snapshot().State; got != StateError {
		t.Fatalf("estado = %v, esperado %v", got, StateError)
	}
	if s.Snapshot().LastError == "" {
		t.Fatal("o erro da verificacao deveria ficar registrado no snapshot")
	}
}

func TestFalhaAoConectarMantemEstadoDesconectado(t *testing.T) {
	// Arrange
	checker := &fakeChecker{results: []checkResult{{connected: false}, {connected: false}}}
	connector := &fakeConnector{err: errCheckFailed}
	s := New(baseConfig(), checker, connector, silentLogger{}, newInstantClock())

	// Act
	runCycles(t, s, 2)

	// Assert
	snap := s.Snapshot()
	if snap.State != StateDisconnected {
		t.Fatalf("estado = %v, esperado %v", snap.State, StateDisconnected)
	}
	if snap.ReconnectCount != 0 {
		t.Fatalf("reconexao falha nao deveria ser contabilizada; contagem = %d", snap.ReconnectCount)
	}
	if snap.LastError == "" {
		t.Fatal("a falha de conexao deveria ficar registrada no snapshot")
	}
}

func TestPausarImpedeVerificacao(t *testing.T) {
	// Arrange
	checker := &fakeChecker{results: []checkResult{{connected: false}}}
	connector := &fakeConnector{}
	s := New(baseConfig(), checker, connector, silentLogger{}, newInstantClock())

	// Act
	s.SetPaused(true)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { s.Run(ctx); close(done) }()
	time.Sleep(50 * time.Millisecond)
	cancel()
	<-done

	// Assert
	if got := checker.callCount(); got != 0 {
		t.Fatalf("pausado nao deveria verificar; chamadas = %d", got)
	}
	if got := s.Snapshot().State; got != StatePaused {
		t.Fatalf("estado = %v, esperado %v", got, StatePaused)
	}
}

func TestObservadorRecebeMudancaDeEstado(t *testing.T) {
	// Arrange
	checker := &fakeChecker{results: []checkResult{{connected: true}}}
	s := New(baseConfig(), checker, &fakeConnector{}, silentLogger{}, newInstantClock())

	received := make([]State, 0, 2)
	s.Subscribe(func(snap Snapshot) { received = append(received, snap.State) })

	// Act
	runCycles(t, s, 2)

	// Assert: apenas a transicao gera notificacao, nao cada verificacao.
	if len(received) != 1 {
		t.Fatalf("notificacoes = %d, esperado 1 (so a transicao)", len(received))
	}
	if received[0] != StateConnected {
		t.Fatalf("estado notificado = %v, esperado %v", received[0], StateConnected)
	}
}
