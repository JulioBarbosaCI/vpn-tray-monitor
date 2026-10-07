package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestValidateAceitaConfiguracaoCompleta(t *testing.T) {
	// Arrange
	cfg := Config{
		Check:   Check{Kind: CheckTCP, Host: "10.0.0.1", Port: 445},
		Connect: Connect{Kind: ConnectRasdial, Profile: "VPN"},
	}

	// Act
	err := cfg.Validate()

	// Assert
	if err != nil {
		t.Fatalf("configuracao valida foi recusada: %v", err)
	}
}

func TestValidateRecusaCheckSemHost(t *testing.T) {
	// Arrange
	cfg := Config{
		Check:   Check{Kind: CheckTCP, Port: 445},
		Connect: Connect{Kind: ConnectRasdial, Profile: "VPN"},
	}

	// Act
	err := cfg.Validate()

	// Assert
	if err == nil {
		t.Fatal("check.kind=tcp sem host deveria ser recusado")
	}
}

func TestValidateRecusaPortaForaDaFaixa(t *testing.T) {
	// Arrange
	cfg := Config{
		Check:   Check{Kind: CheckTCP, Host: "10.0.0.1", Port: 70000},
		Connect: Connect{Kind: ConnectRasdial, Profile: "VPN"},
	}

	// Act
	err := cfg.Validate()

	// Assert
	if err == nil {
		t.Fatal("porta acima de 65535 deveria ser recusada")
	}
}

func TestValidateRecusaConnectDesconhecido(t *testing.T) {
	// Arrange
	cfg := Config{
		Check:   Check{Kind: CheckTCP, Host: "10.0.0.1", Port: 445},
		Connect: Connect{Kind: "telepatia"},
	}

	// Act
	err := cfg.Validate()

	// Assert
	if err == nil {
		t.Fatal("connect.kind desconhecido deveria ser recusado")
	}
}

func TestIntervaloRespeitaOMinimo(t *testing.T) {
	// Arrange: 1 segundo martelaria a rede sem necessidade.
	cfg := Config{CheckIntervalSeconds: 1}

	// Act
	got := cfg.CheckInterval()

	// Assert
	if got != MinCheckInterval {
		t.Fatalf("intervalo = %s, esperado o minimo de %s", got, MinCheckInterval)
	}
}

func TestIntervaloUsaOPadraoQuandoAusente(t *testing.T) {
	// Arrange
	cfg := Config{}

	// Act
	got := cfg.CheckInterval()

	// Assert
	if got != DefaultCheckInterval {
		t.Fatalf("intervalo = %s, esperado o padrao de %s", got, DefaultCheckInterval)
	}
}

func TestTimeoutDeConexaoUsaOPadraoQuandoAusente(t *testing.T) {
	// Arrange
	c := Connect{}

	// Act & Assert
	if got := c.Timeout(); got != DefaultConnectTimeout {
		t.Fatalf("timeout = %s, esperado %s", got, DefaultConnectTimeout)
	}
}

func TestLoadLeArquivoValido(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	conteudo := `{
	  "displayName": "VPN Empresa",
	  "checkIntervalSeconds": 30,
	  "check":   {"kind": "tcp", "host": "10.0.0.1", "port": 445},
	  "connect": {"kind": "rasdial", "profile": "VPN Empresa"}
	}`
	if err := os.WriteFile(path, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("preparando o arquivo de teste: %v", err)
	}

	// Act
	cfg, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("Load devolveu erro: %v", err)
	}
	if cfg.DisplayName != "VPN Empresa" {
		t.Fatalf("displayName = %q", cfg.DisplayName)
	}
	if cfg.CheckInterval() != 30*time.Second {
		t.Fatalf("intervalo = %s, esperado 30s", cfg.CheckInterval())
	}
}

func TestLoadAceitaArquivoComBOM(t *testing.T) {
	// Arrange: o Bloco de Notas grava BOM, que quebraria o decoder JSON.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	conteudo := "\xEF\xBB\xBF" + `{
	  "check":   {"kind": "ping", "host": "10.0.0.1"},
	  "connect": {"kind": "service", "profile": "WireGuardTunnel"}
	}`
	if err := os.WriteFile(path, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("preparando o arquivo de teste: %v", err)
	}

	// Act
	_, err := Load(path)

	// Assert
	if err != nil {
		t.Fatalf("arquivo com BOM deveria ser aceito: %v", err)
	}
}

func TestLoadRecusaCampoDesconhecido(t *testing.T) {
	// Arrange: erro de digitacao silencioso seria pior que falhar na largada.
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	conteudo := `{
	  "checkIntervalSecond": 30,
	  "check":   {"kind": "tcp", "host": "10.0.0.1", "port": 445},
	  "connect": {"kind": "rasdial", "profile": "VPN"}
	}`
	if err := os.WriteFile(path, []byte(conteudo), 0o600); err != nil {
		t.Fatalf("preparando o arquivo de teste: %v", err)
	}

	// Act
	_, err := Load(path)

	// Assert
	if err == nil {
		t.Fatal("campo desconhecido deveria ser recusado para evitar erro de digitacao silencioso")
	}
}
