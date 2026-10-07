// Package config carrega e valida a configuracao do monitor de VPN.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Limites e padroes. Evita numeros magicos espalhados pelo codigo.
const (
	DefaultCheckInterval   = 30 * time.Second
	DefaultConnectTimeout  = 60 * time.Second
	DefaultCheckTimeout    = 5 * time.Second
	DefaultGraceAfterUp    = 15 * time.Second
	DefaultFailuresToRetry = 2
	DefaultMaxBackoff      = 5 * time.Minute
	MinCheckInterval       = 5 * time.Second
	ConfigFileName         = "config.json"
)

// CheckKind identifica como o programa decide se a VPN esta no ar.
type CheckKind string

const (
	CheckTCP       CheckKind = "tcp"       // disca TCP em host:porta interno
	CheckPing      CheckKind = "ping"      // ICMP via utilitario ping do Windows
	CheckInterface CheckKind = "interface" // procura adaptador de rede pelo nome
	CheckCommand   CheckKind = "command"   // comando externo; exit code 0 = conectada
)

// ConnectKind identifica como o programa reconecta a VPN.
type ConnectKind string

const (
	ConnectRasdial  ConnectKind = "rasdial" // VPN nativa do Windows
	ConnectOpenVPN  ConnectKind = "openvpn" // OpenVPN GUI ou CLI
	ConnectService  ConnectKind = "service" // sobe um servico do Windows
	ConnectCommand  ConnectKind = "command" // comando livre
	ConnectSchedTsk ConnectKind = "task"    // tarefa agendada
)

// Check descreve a verificacao de estado da VPN.
type Check struct {
	Kind CheckKind `json:"kind"`
	// TCP
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty"`
	// Interface: trecho do nome/descricao do adaptador (case-insensitive).
	InterfaceMatch string `json:"interfaceMatch,omitempty"`
	// Command
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	// TimeoutSeconds limita cada tentativa de verificacao.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

// Timeout devolve o timeout da verificacao ja com o padrao aplicado.
func (c Check) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return DefaultCheckTimeout
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// Connect descreve como restabelecer a conexao.
type Connect struct {
	Kind ConnectKind `json:"kind"`
	// Rasdial / OpenVPN / Service: nome do perfil, conexao ou servico.
	Profile string `json:"profile,omitempty"`
	// Usuario da VPN. A senha NUNCA fica aqui: veja SecretName.
	Username string `json:"username,omitempty"`
	// SecretName e a chave do cofre local (DPAPI) onde a senha foi gravada
	// por `vpnmon.exe -set-password`. Vazio = conexao sem senha.
	SecretName string `json:"secretName,omitempty"`
	// Command / OpenVPN: executavel e argumentos.
	Command string   `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	// TimeoutSeconds limita a tentativa de conexao.
	TimeoutSeconds int `json:"timeoutSeconds,omitempty"`
}

// Timeout devolve o timeout de conexao ja com o padrao aplicado.
func (c Connect) Timeout() time.Duration {
	if c.TimeoutSeconds <= 0 {
		return DefaultConnectTimeout
	}
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// Config e o documento completo lido de config.json.
type Config struct {
	// DisplayName aparece no tooltip e nas notificacoes.
	DisplayName string `json:"displayName"`
	// CheckIntervalSeconds e o intervalo entre verificacoes.
	CheckIntervalSeconds int `json:"checkIntervalSeconds"`
	// FailuresBeforeReconnect evita reconectar por causa de uma falha isolada.
	FailuresBeforeReconnect int `json:"failuresBeforeReconnect"`
	// GraceAfterConnectSeconds e a espera apos conectar antes de verificar de novo.
	GraceAfterConnectSeconds int `json:"graceAfterConnectSeconds"`
	// MaxBackoffSeconds limita o crescimento do intervalo entre tentativas.
	MaxBackoffSeconds int `json:"maxBackoffSeconds"`
	// NotifyOnChange mostra balao do Windows quando o estado muda.
	NotifyOnChange bool `json:"notifyOnChange"`
	// StartWithWindows registra o programa no boot do usuario.
	StartWithWindows bool `json:"startWithWindows"`
	// LogMaxSizeKB limita o tamanho do arquivo de log antes de rotacionar.
	LogMaxSizeKB int `json:"logMaxSizeKB"`

	Check   Check   `json:"check"`
	Connect Connect `json:"connect"`
}

// CheckInterval devolve o intervalo entre verificacoes ja com o padrao aplicado.
func (c Config) CheckInterval() time.Duration {
	if c.CheckIntervalSeconds <= 0 {
		return DefaultCheckInterval
	}
	d := time.Duration(c.CheckIntervalSeconds) * time.Second
	if d < MinCheckInterval {
		return MinCheckInterval
	}
	return d
}

// GraceAfterConnect devolve a carencia pos-conexao ja com o padrao aplicado.
func (c Config) GraceAfterConnect() time.Duration {
	if c.GraceAfterConnectSeconds <= 0 {
		return DefaultGraceAfterUp
	}
	return time.Duration(c.GraceAfterConnectSeconds) * time.Second
}

// MaxBackoff devolve o teto do backoff ja com o padrao aplicado.
func (c Config) MaxBackoff() time.Duration {
	if c.MaxBackoffSeconds <= 0 {
		return DefaultMaxBackoff
	}
	return time.Duration(c.MaxBackoffSeconds) * time.Second
}

// FailuresToReconnect devolve quantas falhas seguidas disparam a reconexao.
func (c Config) FailuresToReconnect() int {
	if c.FailuresBeforeReconnect <= 0 {
		return DefaultFailuresToRetry
	}
	return c.FailuresBeforeReconnect
}

// Validate recusa configuracoes que nao teriam como funcionar em execucao.
func (c Config) Validate() error {
	switch c.Check.Kind {
	case CheckTCP:
		if c.Check.Host == "" {
			return fmt.Errorf("check.host e obrigatorio quando check.kind=tcp")
		}
		if c.Check.Port <= 0 || c.Check.Port > 65535 {
			return fmt.Errorf("check.port deve estar entre 1 e 65535, recebido %d", c.Check.Port)
		}
	case CheckPing:
		if c.Check.Host == "" {
			return fmt.Errorf("check.host e obrigatorio quando check.kind=ping")
		}
	case CheckInterface:
		if c.Check.InterfaceMatch == "" {
			return fmt.Errorf("check.interfaceMatch e obrigatorio quando check.kind=interface")
		}
	case CheckCommand:
		if c.Check.Command == "" {
			return fmt.Errorf("check.command e obrigatorio quando check.kind=command")
		}
	case "":
		return fmt.Errorf("check.kind e obrigatorio")
	default:
		return fmt.Errorf("check.kind desconhecido: %q", c.Check.Kind)
	}

	switch c.Connect.Kind {
	case ConnectRasdial:
		// profile e opcional: quando ausente, o programa descobre o nome da
		// conexao na agenda do proprio Windows.
	case ConnectService, ConnectSchedTsk:
		if c.Connect.Profile == "" {
			return fmt.Errorf("connect.profile e obrigatorio quando connect.kind=%s", c.Connect.Kind)
		}
	case ConnectOpenVPN:
		if c.Connect.Profile == "" && c.Connect.Command == "" {
			return fmt.Errorf("connect.profile ou connect.command e obrigatorio quando connect.kind=openvpn")
		}
	case ConnectCommand:
		if c.Connect.Command == "" {
			return fmt.Errorf("connect.command e obrigatorio quando connect.kind=command")
		}
	case "":
		return fmt.Errorf("connect.kind e obrigatorio")
	default:
		return fmt.Errorf("connect.kind desconhecido: %q", c.Connect.Kind)
	}

	return nil
}

// parse interpreta o conteudo de um config.json, sem validar. Fica separado de
// Load porque Resolve precisa aplicar os parametros da linha de comando antes
// de validar: um arquivo incompleto pode ser completado por eles.
func parse(raw []byte, path string) (Config, error) {
	var cfg Config
	dec := json.NewDecoder(newTrimmer(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("interpretando %s: %w", path, err)
	}
	return cfg, nil
}

// Load le e valida o config.json do caminho informado.
func Load(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("lendo %s: %w", path, err)
	}

	cfg, err := parse(raw, path)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("configuracao invalida em %s: %w", path, err)
	}
	return cfg, nil
}

// DefaultPath devolve o config.json ao lado do executavel.
func DefaultPath() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("localizando o executavel: %w", err)
	}
	return filepath.Join(filepath.Dir(exe), ConfigFileName), nil
}
