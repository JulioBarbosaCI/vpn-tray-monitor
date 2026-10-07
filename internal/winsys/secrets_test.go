package winsys

import (
	"path/filepath"
	"testing"
)

func TestGravaELeSegredo(t *testing.T) {
	// Arrange
	store := NewSecretStore(t.TempDir())

	// Act
	if err := store.Set("vpn", "senha-secreta"); err != nil {
		t.Fatalf("gravando o segredo: %v", err)
	}
	got, err := store.Secret("vpn")

	// Assert
	if err != nil {
		t.Fatalf("lendo o segredo: %v", err)
	}
	if got != "senha-secreta" {
		t.Fatalf("segredo = %q, esperado %q", got, "senha-secreta")
	}
}

func TestSegredoNaoFicaEmTextoPlanoNoArquivo(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	store := NewSecretStore(dir)
	const senha = "SenhaMuitoSecreta123"

	// Act
	if err := store.Set("vpn", senha); err != nil {
		t.Fatalf("gravando o segredo: %v", err)
	}

	// Assert
	raw := readFile(t, filepath.Join(dir, "vpn.secret"))
	if contains(raw, senha) {
		t.Fatal("a senha aparece em texto plano no arquivo do cofre")
	}
}

func TestSegredoAusenteExplicaComoGravar(t *testing.T) {
	// Arrange
	store := NewSecretStore(t.TempDir())

	// Act
	_, err := store.Secret("inexistente")

	// Assert
	if err == nil {
		t.Fatal("ler um segredo ausente deveria falhar")
	}
	if !contains(err.Error(), "set-password") {
		t.Fatalf("a mensagem deveria ensinar a gravar o segredo; recebida: %v", err)
	}
}

func TestNomeDeSegredoComCaminhoEhRecusado(t *testing.T) {
	// Arrange: um nome com barra escaparia da pasta do cofre.
	store := NewSecretStore(t.TempDir())

	// Act
	err := store.Set(filepath.Join("..", "fora"), "x")

	// Assert
	if err == nil {
		t.Fatal("nome de segredo com caminho deveria ser recusado")
	}
}

func TestHasInformaSeOSegredoExiste(t *testing.T) {
	// Arrange
	store := NewSecretStore(t.TempDir())

	// Act & Assert
	if store.Has("vpn") {
		t.Fatal("Has deveria ser falso antes de gravar")
	}
	if err := store.Set("vpn", "x"); err != nil {
		t.Fatalf("gravando o segredo: %v", err)
	}
	if !store.Has("vpn") {
		t.Fatal("Has deveria ser verdadeiro depois de gravar")
	}
}
