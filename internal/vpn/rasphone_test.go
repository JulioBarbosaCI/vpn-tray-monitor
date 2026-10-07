package vpn

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"
)

// pbkANSI monta uma agenda de conexoes como o Windows grava em ANSI.
func pbkANSI(perfis ...string) []byte {
	var b strings.Builder
	for _, p := range perfis {
		b.WriteString("[" + p + "]\r\n")
		b.WriteString("Encoding=1\r\n")
		b.WriteString("Type=2\r\n")
		b.WriteString("\r\n")
	}
	return []byte(b.String())
}

// pbkUTF16 monta a mesma agenda em UTF-16LE com BOM, a outra forma que o
// Windows usa conforme a versao.
func pbkUTF16(perfis ...string) []byte {
	texto := string(pbkANSI(perfis...))
	unidades := utf16.Encode([]rune(texto))

	out := []byte{0xff, 0xfe}
	for _, u := range unidades {
		out = append(out, byte(u), byte(u>>8))
	}
	return out
}

func TestParsePhonebookLeNomesDePerfilEmANSI(t *testing.T) {
	// Arrange
	raw := pbkANSI("VPN Empresa", "VPN Filial")

	// Act
	perfis := parsePhonebook(raw)

	// Assert
	if len(perfis) != 2 {
		t.Fatalf("perfis = %v, esperado 2", perfis)
	}
	if perfis[0] != "VPN Empresa" || perfis[1] != "VPN Filial" {
		t.Fatalf("perfis = %v", perfis)
	}
}

func TestParsePhonebookLeNomesDePerfilEmUTF16(t *testing.T) {
	// Arrange: sem tratar UTF-16 o parser veria bytes nulos e nao acharia nada.
	raw := pbkUTF16("VPN Empresa")

	// Act
	perfis := parsePhonebook(raw)

	// Assert
	if len(perfis) != 1 || perfis[0] != "VPN Empresa" {
		t.Fatalf("perfis = %v, esperado [VPN Empresa]", perfis)
	}
}

func TestParsePhonebookIgnoraLinhasDePropriedade(t *testing.T) {
	// Arrange: so as secoes sao perfis; as chaves de configuracao nao.
	raw := []byte("[VPN Empresa]\r\nEncoding=1\r\nDevice=WAN Miniport\r\n")

	// Act
	perfis := parsePhonebook(raw)

	// Assert
	if len(perfis) != 1 {
		t.Fatalf("perfis = %v, esperado apenas a secao", perfis)
	}
}

func TestParsePhonebookDevolveVazioParaArquivoSemPerfil(t *testing.T) {
	// Arrange & Act
	perfis := parsePhonebook([]byte("\r\n\r\n"))

	// Assert
	if len(perfis) != 0 {
		t.Fatalf("perfis = %v, esperado vazio", perfis)
	}
}

// escreverAgenda grava uma agenda temporaria e a aponta para o descobridor.
func escreverAgenda(t *testing.T, raw []byte) {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), "rasphone.pbk")
	if err := os.WriteFile(caminho, raw, 0o600); err != nil {
		t.Fatalf("gravando a agenda de teste: %v", err)
	}
	t.Setenv("VPNMON_PBK", caminho)
}

func TestResolveUsaOPerfilDoConfigQuandoInformado(t *testing.T) {
	// Arrange: mesmo havendo outros cadastrados, o config manda.
	escreverAgenda(t, pbkANSI("Outra VPN"))

	// Act
	perfil, err := resolveRasProfile("VPN Empresa")

	// Assert
	if err != nil {
		t.Fatalf("resolveRasProfile devolveu erro: %v", err)
	}
	if perfil != "VPN Empresa" {
		t.Fatalf("perfil = %q, esperado o do config", perfil)
	}
}

func TestResolveDescobreOUnicoPerfilCadastrado(t *testing.T) {
	// Arrange
	escreverAgenda(t, pbkANSI("VPN Empresa"))

	// Act
	perfil, err := resolveRasProfile("")

	// Assert
	if err != nil {
		t.Fatalf("com um unico perfil a descoberta deveria funcionar: %v", err)
	}
	if perfil != "VPN Empresa" {
		t.Fatalf("perfil = %q, esperado %q", perfil, "VPN Empresa")
	}
}

func TestResolvePedeEscolhaQuandoHaVariosPerfis(t *testing.T) {
	// Arrange: adivinhar qual VPN conectar seria pior que uma mensagem clara.
	escreverAgenda(t, pbkANSI("VPN Empresa", "VPN Filial"))

	// Act
	_, err := resolveRasProfile("")

	// Assert
	if err == nil {
		t.Fatal("com varios perfis a escolha deveria ser exigida")
	}
	if !strings.Contains(err.Error(), "VPN Filial") {
		t.Fatalf("a mensagem deveria listar os perfis encontrados: %v", err)
	}
}

func TestResolveExplicaQuandoNaoHaNenhumaConexao(t *testing.T) {
	// Arrange
	escreverAgenda(t, []byte(""))

	// Act
	_, err := resolveRasProfile("")

	// Assert
	if err == nil {
		t.Fatal("sem conexao cadastrada a resolucao deveria falhar")
	}
	if !strings.Contains(err.Error(), "menu do icone") {
		t.Fatalf("a mensagem deveria ensinar a saida: %v", err)
	}
}
