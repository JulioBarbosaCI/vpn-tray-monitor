package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// escreverConfig grava um config.json temporario e devolve o caminho.
func escreverConfig(t *testing.T, conteudo string) string {
	t.Helper()
	caminho := filepath.Join(t.TempDir(), ConfigFileName)
	if err := os.WriteFile(caminho, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("gravando o config de teste: %v", err)
	}
	return caminho
}

func TestResolveFuncionaSoComParametrosQuandoNaoHaArquivo(t *testing.T) {
	// Arrange: implantacao sem distribuir config.json.
	ausente := filepath.Join(t.TempDir(), "nao-existe.json")

	// Act
	cfg, err := Resolve(ausente, Overrides{
		PingHost: "10.254.1.172",
		Profile:  "VPN Empresa",
	})

	// Assert
	if err != nil {
		t.Fatalf("os parametros deveriam bastar: %v", err)
	}
	if cfg.Check.Kind != CheckPing || cfg.Check.Host != "10.254.1.172" {
		t.Fatalf("check = %+v", cfg.Check)
	}
	if cfg.Connect.Kind != ConnectRasdial || cfg.Connect.Profile != "VPN Empresa" {
		t.Fatalf("connect = %+v", cfg.Connect)
	}
}

func TestResolveExigeAlgoQuandoNaoHaArquivoNemParametros(t *testing.T) {
	// Arrange
	ausente := filepath.Join(t.TempDir(), "nao-existe.json")

	// Act
	_, err := Resolve(ausente, Overrides{})

	// Assert
	if err == nil {
		t.Fatal("sem arquivo e sem parametros a resolucao deveria falhar")
	}
	if !strings.Contains(err.Error(), "-ping") {
		t.Fatalf("a mensagem deveria ensinar a saida: %v", err)
	}
}

func TestParametroVenceOArquivo(t *testing.T) {
	// Arrange: o arquivo diz uma coisa, a linha de comando diz outra.
	caminho := escreverConfig(t, `{
	  "displayName": "VPN Empresa",
	  "check":   {"kind": "tcp", "host": "10.0.0.1", "port": 445},
	  "connect": {"kind": "rasdial", "profile": "Antiga"}
	}`)

	// Act
	cfg, err := Resolve(caminho, Overrides{
		PingHost: "10.254.1.172",
		Profile:  "Nova",
	})

	// Assert
	if err != nil {
		t.Fatalf("Resolve devolveu erro: %v", err)
	}
	if cfg.Check.Kind != CheckPing || cfg.Check.Host != "10.254.1.172" {
		t.Fatalf("o parametro -ping deveria vencer o arquivo; check = %+v", cfg.Check)
	}
	if cfg.Connect.Profile != "Nova" {
		t.Fatalf("o parametro -vpn deveria vencer o arquivo; profile = %q", cfg.Connect.Profile)
	}
}

func TestArquivoValeQuandoNaoHaParametro(t *testing.T) {
	// Arrange
	caminho := escreverConfig(t, `{
	  "displayName": "VPN Empresa",
	  "checkIntervalSeconds": 45,
	  "check":   {"kind": "ping", "host": "10.254.1.172"},
	  "connect": {"kind": "rasdial", "profile": "VPN Empresa"}
	}`)

	// Act
	cfg, err := Resolve(caminho, Overrides{})

	// Assert
	if err != nil {
		t.Fatalf("Resolve devolveu erro: %v", err)
	}
	if cfg.Check.Host != "10.254.1.172" || cfg.Connect.Profile != "VPN Empresa" {
		t.Fatalf("o arquivo deveria valer integralmente; cfg = %+v", cfg)
	}
	if cfg.CheckIntervalSeconds != 45 {
		t.Fatalf("intervalo = %d, esperado 45", cfg.CheckIntervalSeconds)
	}
}

func TestParametroDeIntervaloSobrepoeOArquivo(t *testing.T) {
	// Arrange
	caminho := escreverConfig(t, `{
	  "checkIntervalSeconds": 45,
	  "check":   {"kind": "ping", "host": "10.254.1.172"},
	  "connect": {"kind": "rasdial", "profile": "VPN"}
	}`)

	// Act
	cfg, err := Resolve(caminho, Overrides{IntervalSeconds: 10})

	// Assert
	if err != nil {
		t.Fatalf("Resolve devolveu erro: %v", err)
	}
	if cfg.CheckIntervalSeconds != 10 {
		t.Fatalf("intervalo = %d, esperado 10", cfg.CheckIntervalSeconds)
	}
}

func TestParametroPreservaOTimeoutJaConfigurado(t *testing.T) {
	// Arrange: trocar o alvo do ping nao deve descartar o ajuste fino do arquivo.
	caminho := escreverConfig(t, `{
	  "check":   {"kind": "tcp", "host": "10.0.0.1", "port": 445, "timeoutSeconds": 9},
	  "connect": {"kind": "rasdial", "profile": "VPN"}
	}`)

	// Act
	cfg, err := Resolve(caminho, Overrides{PingHost: "10.254.1.172"})

	// Assert
	if err != nil {
		t.Fatalf("Resolve devolveu erro: %v", err)
	}
	if cfg.Check.TimeoutSeconds != 9 {
		t.Fatalf("timeout = %d, esperado 9 preservado do arquivo", cfg.Check.TimeoutSeconds)
	}
}

func TestResolveRecusaArquivoInvalido(t *testing.T) {
	// Arrange
	caminho := escreverConfig(t, `{ isso nao e json }`)

	// Act
	_, err := Resolve(caminho, Overrides{PingHost: "10.254.1.172"})

	// Assert
	if err == nil {
		t.Fatal("arquivo invalido deveria ser recusado, mesmo com parametros validos")
	}
}

func TestOverridesVaziosSaoDetectados(t *testing.T) {
	// Arrange & Act & Assert
	if !(Overrides{}).Empty() {
		t.Fatal("Overrides sem campos deveria ser considerado vazio")
	}
	if (Overrides{PingHost: "10.254.1.172"}).Empty() {
		t.Fatal("Overrides com host nao deveria ser vazio")
	}
}
