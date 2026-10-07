package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGravaMensagemComNivel(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "vpnmon.log")
	log, err := New(path, 64, nil)
	if err != nil {
		t.Fatalf("criando o logger: %v", err)
	}
	defer log.Close()

	// Act
	log.Infof("VPN %s", "conectada")

	// Assert
	conteudo, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("lendo o log: %v", err)
	}
	if !strings.Contains(string(conteudo), "VPN conectada") {
		t.Fatalf("mensagem ausente no log: %q", conteudo)
	}
	if !strings.Contains(string(conteudo), "INFO") {
		t.Fatalf("nivel ausente no log: %q", conteudo)
	}
}

func TestRotacionaAoUltrapassarOLimite(t *testing.T) {
	// Arrange: limite de 1 KB, com escrita suficiente para estourar.
	dir := t.TempDir()
	path := filepath.Join(dir, "vpnmon.log")
	log, err := New(path, 1, nil)
	if err != nil {
		t.Fatalf("criando o logger: %v", err)
	}
	defer log.Close()

	// Act
	for i := 0; i < 60; i++ {
		log.Infof("linha de preenchimento numero %d com texto suficiente para encher", i)
	}

	// Assert
	if _, err := os.Stat(path + backupSuffix); err != nil {
		t.Fatalf("o arquivo rotacionado deveria existir: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("o log atual deveria existir: %v", err)
	}
	if info.Size() > 2*1024 {
		t.Fatalf("o log atual deveria ter sido truncado; tamanho = %d", info.Size())
	}
}

func TestPathDevolveOCaminhoDoArquivo(t *testing.T) {
	// Arrange
	path := filepath.Join(t.TempDir(), "vpnmon.log")
	log, err := New(path, 64, nil)
	if err != nil {
		t.Fatalf("criando o logger: %v", err)
	}
	defer log.Close()

	// Act & Assert
	if got := log.Path(); got != path {
		t.Fatalf("Path() = %q, esperado %q", got, path)
	}
}
