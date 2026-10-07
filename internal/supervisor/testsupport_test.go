package supervisor

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/guibsu/vpn-tray-monitor/internal/config"
)

// fakeChecker devolve respostas roteirizadas, para que o teste controle
// exatamente a sequencia de estados que o supervisor observa.
type fakeChecker struct {
	mu      sync.Mutex
	results []checkResult
	calls   int
}

type checkResult struct {
	connected bool
	err       error
}

func (f *fakeChecker) Check(context.Context) (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.calls >= len(f.results) {
		// Depois do roteiro, mantem a ultima resposta.
		last := f.results[len(f.results)-1]
		f.calls++
		return last.connected, last.err
	}
	r := f.results[f.calls]
	f.calls++
	return r.connected, r.err
}

func (f *fakeChecker) Describe() string { return "checker de teste" }

func (f *fakeChecker) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

// fakeConnector conta tentativas e pode ser configurado para falhar.
type fakeConnector struct {
	mu       sync.Mutex
	attempts int
	err      error
}

func (f *fakeConnector) Connect(context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.attempts++
	return f.err
}

func (f *fakeConnector) Describe() string { return "connector de teste" }

func (f *fakeConnector) attemptCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.attempts
}

// silentLogger descarta as mensagens: o teste verifica estado, nao texto.
type silentLogger struct{}

func (silentLogger) Infof(string, ...any)  {}
func (silentLogger) Warnf(string, ...any)  {}
func (silentLogger) Errorf(string, ...any) {}

// instantClock nao espera de verdade, para o teste rodar em milissegundos.
type instantClock struct {
	mu    sync.Mutex
	now   time.Time
	slept []time.Duration
}

func newInstantClock() *instantClock {
	return &instantClock{now: time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)}
}

func (c *instantClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *instantClock) Sleep(ctx context.Context, d time.Duration) bool {
	c.mu.Lock()
	c.slept = append(c.slept, d)
	c.now = c.now.Add(d)
	c.mu.Unlock()
	return ctx.Err() == nil
}

// baseConfig e a configuracao minima valida usada pelos testes.
func baseConfig() config.Config {
	return config.Config{
		DisplayName:              "VPN de teste",
		CheckIntervalSeconds:     5,
		FailuresBeforeReconnect:  2,
		GraceAfterConnectSeconds: 1,
		MaxBackoffSeconds:        60,
		Check: config.Check{
			Kind: config.CheckTCP,
			Host: "10.0.0.1",
			Port: 445,
		},
		Connect: config.Connect{
			Kind:    config.ConnectRasdial,
			Profile: "VPN Empresa",
		},
	}
}

var errCheckFailed = errors.New("ferramenta de verificacao indisponivel")
