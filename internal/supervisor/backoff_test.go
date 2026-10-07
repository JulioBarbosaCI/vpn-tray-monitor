package supervisor

import (
	"testing"
	"time"
)

func TestBackoffDobraAteOTeto(t *testing.T) {
	// Arrange
	bo := newBackoff(10*time.Second, 40*time.Second)

	// Act & Assert: dobra a cada chamada e para no teto.
	esperados := []time.Duration{
		10 * time.Second,
		20 * time.Second,
		40 * time.Second,
		40 * time.Second,
	}
	for i, esperado := range esperados {
		if got := bo.next(); got != esperado {
			t.Fatalf("chamada %d: intervalo = %s, esperado %s", i+1, got, esperado)
		}
	}
}

func TestBackoffVoltaAoNormalAposReset(t *testing.T) {
	// Arrange
	bo := newBackoff(10*time.Second, 60*time.Second)
	bo.next()
	bo.next()

	// Act
	bo.reset()

	// Assert
	if got := bo.peek(); got != 10*time.Second {
		t.Fatalf("apos reset o intervalo = %s, esperado 10s", got)
	}
}

func TestBackoffNuncaUltrapassaOTetoMesmoComBaseMaior(t *testing.T) {
	// Arrange: configuracao incoerente, base acima do teto.
	bo := newBackoff(90*time.Second, 60*time.Second)

	// Act
	bo.next()

	// Assert
	if got := bo.peek(); got > 60*time.Second {
		t.Fatalf("intervalo = %s, nao deveria ultrapassar o teto de 60s", got)
	}
}
