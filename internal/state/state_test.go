package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGravaELeAEscolhaDeVPN(t *testing.T) {
	// Arrange
	caminho := Path(t.TempDir())

	// Act
	if err := Save(caminho, State{VPNProfile: "VPN Empresa"}); err != nil {
		t.Fatalf("gravando o estado: %v", err)
	}
	lido, err := Load(caminho)

	// Assert
	if err != nil {
		t.Fatalf("lendo o estado: %v", err)
	}
	if lido.VPNProfile != "VPN Empresa" {
		t.Fatalf("perfil = %q, esperado %q", lido.VPNProfile, "VPN Empresa")
	}
}

func TestArquivoAusenteNaoEhErro(t *testing.T) {
	// Arrange: na primeira execucao o arquivo ainda nao existe.
	caminho := filepath.Join(t.TempDir(), "nao-existe.json")

	// Act
	lido, err := Load(caminho)

	// Assert
	if err != nil {
		t.Fatalf("arquivo ausente nao deveria ser erro: %v", err)
	}
	if lido.VPNProfile != "" {
		t.Fatalf("perfil = %q, esperado vazio", lido.VPNProfile)
	}
}

func TestArquivoCorrompidoNaoImpedeOProgramaDeSubir(t *testing.T) {
	// Arrange: estado ilegivel nao pode travar a partida; o usuario reescolhe.
	caminho := Path(t.TempDir())
	if err := os.WriteFile(caminho, []byte("{ isso nao e json"), 0o600); err != nil {
		t.Fatalf("preparando o arquivo: %v", err)
	}

	// Act
	lido, err := Load(caminho)

	// Assert
	if err != nil {
		t.Fatalf("estado corrompido deveria ser ignorado, nao virar erro: %v", err)
	}
	if lido.VPNProfile != "" {
		t.Fatalf("perfil = %q, esperado vazio", lido.VPNProfile)
	}
}

func TestSaveSobrescreveAEscolhaAnterior(t *testing.T) {
	// Arrange
	caminho := Path(t.TempDir())
	if err := Save(caminho, State{VPNProfile: "Antiga"}); err != nil {
		t.Fatalf("gravando o primeiro estado: %v", err)
	}

	// Act
	if err := Save(caminho, State{VPNProfile: "Nova"}); err != nil {
		t.Fatalf("gravando o segundo estado: %v", err)
	}
	lido, err := Load(caminho)

	// Assert
	if err != nil {
		t.Fatalf("lendo o estado: %v", err)
	}
	if lido.VPNProfile != "Nova" {
		t.Fatalf("perfil = %q, esperado %q", lido.VPNProfile, "Nova")
	}
}

func TestPathUsaOArquivoPadrao(t *testing.T) {
	// Arrange & Act
	got := Path("/opt/vpnmon")

	// Assert
	if got != filepath.Join("/opt/vpnmon", FileName) {
		t.Fatalf("Path = %q", got)
	}
}
