package config

import (
	"errors"
	"fmt"
	"os"
)

// Overrides reune o que veio pela linha de comando.
//
// Passar a VPN e o alvo do ping por parametro permite implantar o programa sem
// distribuir arquivo de configuracao: basta o executavel e um atalho com os
// argumentos. O config.json continua valendo para quem quiser ajustar o resto.
type Overrides struct {
	// Profile e o nome da conexao VPN cadastrada no Windows.
	Profile string
	// PingHost e o endereco que responde quando a VPN esta no ar.
	PingHost string
	// IntervalSeconds, quando maior que zero, troca o intervalo entre verificacoes.
	IntervalSeconds int
}

// Empty informa que nada foi passado pela linha de comando.
func (o Overrides) Empty() bool {
	return o.Profile == "" && o.PingHost == "" && o.IntervalSeconds <= 0
}

// apply devolve uma copia da configuracao com os valores da linha de comando
// sobrepostos. O que veio por parametro vence o arquivo, porque foi digitado
// agora, para esta execucao.
func (c Config) apply(o Overrides) Config {
	next := c
	if o.PingHost != "" {
		next.Check = Check{
			Kind:           CheckPing,
			Host:           o.PingHost,
			TimeoutSeconds: c.Check.TimeoutSeconds,
		}
	}
	if o.Profile != "" {
		next.Connect.Kind = ConnectRasdial
		next.Connect.Profile = o.Profile
	}
	if o.IntervalSeconds > 0 {
		next.CheckIntervalSeconds = o.IntervalSeconds
	}
	return next
}

// fromOverrides monta uma configuracao completa apenas com o que veio da linha
// de comando, para o caso de nao existir config.json.
func fromOverrides(o Overrides) Config {
	return Config{
		DisplayName:              "VPN",
		CheckIntervalSeconds:     o.IntervalSeconds,
		FailuresBeforeReconnect:  DefaultFailuresToRetry,
		GraceAfterConnectSeconds: int(DefaultGraceAfterUp.Seconds()),
		MaxBackoffSeconds:        int(DefaultMaxBackoff.Seconds()),
		NotifyOnChange:           true,
		Check: Check{
			Kind: CheckPing,
			Host: o.PingHost,
		},
		Connect: Connect{
			Kind:    ConnectRasdial,
			Profile: o.Profile,
		},
	}.apply(o)
}

// Resolve decide a configuracao final combinando arquivo e linha de comando.
//
// Com config.json presente, ele e a base e os parametros o ajustam. Sem
// arquivo, os parametros sozinhos bastam — desde que digam ao menos o que pingar.
func Resolve(path string, o Overrides) (Config, error) {
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		cfg, parseErr := parse(raw, path)
		if parseErr != nil {
			return Config{}, parseErr
		}
		final := cfg.apply(o)
		if valErr := final.Validate(); valErr != nil {
			return Config{}, fmt.Errorf("configuracao invalida em %s: %w", path, valErr)
		}
		return final, nil

	case errors.Is(err, os.ErrNotExist):
		if o.Empty() {
			return Config{}, fmt.Errorf(
				"nao encontrei %s nem parametros de linha de comando; "+
					"informe -ping e -vpn, ou crie o arquivo de configuracao", path)
		}
		final := fromOverrides(o)
		if valErr := final.Validate(); valErr != nil {
			return Config{}, fmt.Errorf("parametros insuficientes: %w", valErr)
		}
		return final, nil

	default:
		return Config{}, fmt.Errorf("lendo %s: %w", path, err)
	}
}
