package trayui

import (
	"strings"
	"testing"
	"time"

	"github.com/guibsu/vpn-tray-monitor/internal/supervisor"
)

func TestDetalheMostraOErroQuandoExiste(t *testing.T) {
	// Arrange
	snap := supervisor.Snapshot{LastError: "rasdial falhou: codigo 691"}

	// Act
	got := detailFor(snap)

	// Assert
	if !strings.Contains(got, "691") {
		t.Fatalf("detalhe = %q, deveria mostrar o erro", got)
	}
}

func TestDetalheMostraHoraDaUltimaVerificacao(t *testing.T) {
	// Arrange
	snap := supervisor.Snapshot{LastCheck: time.Date(2026, 9, 2, 14, 35, 7, 0, time.UTC)}

	// Act
	got := detailFor(snap)

	// Assert
	if !strings.Contains(got, "14:35:07") {
		t.Fatalf("detalhe = %q, deveria conter a hora", got)
	}
}

func TestDetalheIncluiContagemDeReconexoes(t *testing.T) {
	// Arrange
	snap := supervisor.Snapshot{
		LastCheck:      time.Date(2026, 9, 2, 14, 35, 7, 0, time.UTC),
		ReconnectCount: 3,
	}

	// Act
	got := detailFor(snap)

	// Assert
	if !strings.Contains(got, "3") {
		t.Fatalf("detalhe = %q, deveria mostrar as reconexoes", got)
	}
}

func TestDetalheVazioAntesDaPrimeiraVerificacao(t *testing.T) {
	// Arrange
	snap := supervisor.Snapshot{}

	// Act
	got := detailFor(snap)

	// Assert
	if got != "" {
		t.Fatalf("detalhe = %q, esperado vazio antes da primeira verificacao", got)
	}
}

func TestTruncateEncurtaTextoLongo(t *testing.T) {
	// Arrange
	longo := strings.Repeat("a", 100)

	// Act
	got := truncate(longo, 20)

	// Assert
	if len(got) != 20 {
		t.Fatalf("tamanho = %d, esperado 20", len(got))
	}
	if !strings.HasSuffix(got, "...") {
		t.Fatalf("texto truncado deveria terminar em reticencias: %q", got)
	}
}

func TestTruncateNaoAlteraTextoCurto(t *testing.T) {
	// Arrange & Act
	got := truncate("curto", 20)

	// Assert
	if got != "curto" {
		t.Fatalf("texto = %q, esperado inalterado", got)
	}
}

func TestIconeMudaConformeOEstado(t *testing.T) {
	// Arrange & Act & Assert: estados distintos nao podem compartilhar icone,
	// senao o usuario nao distingue conectada de caida na barra.
	if string(iconFor(supervisor.StateConnected)) == string(iconFor(supervisor.StateDisconnected)) {
		t.Fatal("conectada e desconectada deveriam ter icones diferentes")
	}
	if string(iconFor(supervisor.StateConnecting)) == string(iconFor(supervisor.StateConnected)) {
		t.Fatal("conectando e conectada deveriam ter icones diferentes")
	}
	if len(iconFor(supervisor.StatePaused)) == 0 {
		t.Fatal("o estado pausado precisa de um icone")
	}
}
