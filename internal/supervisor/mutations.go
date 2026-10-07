package supervisor

import (
	"context"
	"time"
)

// Este arquivo concentra as escritas no snapshot. Cada funcao produz um novo
// snapshot em vez de alterar o existente no lugar, de modo que quem leu uma
// copia continue com um retrato coerente.

func (s *Supervisor) setState(state State, lastError string) {
	s.mu.Lock()
	previous := s.snapshot.State
	next := s.snapshot
	next.State = state
	next.LastError = lastError
	if previous != state {
		next.Since = s.clock.Now()
	}
	s.snapshot = next
	observers := append([]Observer(nil), s.observers...)
	s.mu.Unlock()

	if previous == state {
		return
	}
	for _, observe := range observers {
		observe(next)
	}
}

func (s *Supervisor) markChecked() {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.snapshot
	next.LastCheck = s.clock.Now()
	s.snapshot = next
}

func (s *Supervisor) incrementFailures() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.snapshot
	next.ConsecutiveFails++
	s.snapshot = next
	return next.ConsecutiveFails
}

func (s *Supervisor) resetFailures() {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.snapshot
	next.ConsecutiveFails = 0
	s.snapshot = next
}

func (s *Supervisor) countReconnect() {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.snapshot
	next.ReconnectCount++
	s.snapshot = next
}

func (s *Supervisor) setNextAttempt(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := s.snapshot
	next.NextAttemptIn = d
	s.snapshot = next
}

func (s *Supervisor) previousState() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.snapshot.State
}

func (s *Supervisor) isPaused() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.paused
}

// realClock e o relogio usado em producao.
type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

// Sleep dorme o periodo pedido, mas desiste se o contexto for cancelado.
// Devolve false quando foi interrompido.
func (realClock) Sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
