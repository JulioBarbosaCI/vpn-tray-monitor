// Comando vpnmon vigia uma conexao VPN e a restabelece quando ela cai,
// mostrando o estado na bandeja do sistema.
//
// Uso:
//
//	vpnmon.exe -ping 10.254.1.172 -vpn "VPN Empresa"
//	                                      monitora sem precisar de config.json
//	vpnmon.exe                            usa o config.json ao lado do executavel
//	vpnmon.exe -config C:\pasta\cfg.json  usa outro arquivo de configuracao
//	vpnmon.exe -check -ping 10.254.1.172  verifica uma vez e sai (diagnostico)
//	vpnmon.exe -set-password vpn          grava a senha da VPN no cofre local
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/guibsu/vpn-tray-monitor/internal/config"
	"github.com/guibsu/vpn-tray-monitor/internal/logging"
	"github.com/guibsu/vpn-tray-monitor/internal/state"
	"github.com/guibsu/vpn-tray-monitor/internal/supervisor"
	"github.com/guibsu/vpn-tray-monitor/internal/trayui"
	"github.com/guibsu/vpn-tray-monitor/internal/vpn"
	"github.com/guibsu/vpn-tray-monitor/internal/winsys"
)

const (
	appName     = "Monitor de VPN"
	logFileName = "vpnmon.log"
	secretsDir  = "secrets"
)

func main() {
	configPath := flag.String("config", "", "caminho do config.json (padrao: ao lado do executavel)")
	setPassword := flag.String("set-password", "", "grava a senha da VPN no cofre local sob o nome informado")
	checkOnce := flag.Bool("check", false, "verifica a VPN uma vez, imprime o resultado e sai")
	vpnProfile := flag.String("vpn", "", "nome da conexao VPN cadastrada no Windows")
	pingHost := flag.String("ping", "", "endereco que responde quando a VPN esta no ar")
	interval := flag.Int("intervalo", 0, "segundos entre verificacoes")
	flag.Parse()

	overrides := config.Overrides{
		Profile:         *vpnProfile,
		PingHost:        *pingHost,
		IntervalSeconds: *interval,
	}

	// Nos modos de linha de comando o programa precisa falar com o terminal;
	// no modo residente ele fica calado, sem abrir janela alguma.
	if *setPassword != "" || *checkOnce {
		winsys.AttachParentConsole()
	}

	if err := run(*configPath, *setPassword, *checkOnce, overrides); err != nil {
		fmt.Fprintln(os.Stderr, "erro:", err)
		os.Exit(1)
	}
}

// run concentra a inicializacao para que todo caminho de erro devolva um erro
// em vez de encerrar o processo no meio da montagem.
func run(configPath, setPassword string, checkOnce bool, overrides config.Overrides) error {
	baseDir, err := baseDirectory()
	if err != nil {
		return err
	}
	if configPath == "" {
		configPath = filepath.Join(baseDir, config.ConfigFileName)
	}

	store := winsys.NewSecretStore(filepath.Join(baseDir, secretsDir))

	// Gravar a senha nao exige configuracao valida: e justamente o passo que
	// se faz antes de tudo funcionar.
	if setPassword != "" {
		return storePassword(store, setPassword)
	}

	// A VPN escolhida no menu da bandeja vale entre execucoes, mas o que vier
	// pela linha de comando ainda tem a ultima palavra nesta execucao.
	statePath := state.Path(baseDir)
	saved, err := state.Load(statePath)
	if err != nil {
		return err
	}
	if overrides.Profile == "" && saved.VPNProfile != "" {
		overrides.Profile = saved.VPNProfile
	}

	cfg, err := config.Resolve(configPath, overrides)
	if err != nil {
		return err
	}

	checker, err := vpn.NewChecker(cfg.Check)
	if err != nil {
		return err
	}
	connector, err := vpn.NewConnector(cfg.Connect, store)
	if err != nil {
		return err
	}

	if checkOnce {
		return reportSingleCheck(checker)
	}

	log, err := logging.New(filepath.Join(baseDir, logFileName), cfg.LogMaxSizeKB, nil)
	if err != nil {
		return err
	}
	defer log.Close()

	// StartWithWindows na configuracao e a intencao declarada; aplica-la na
	// partida mantem o registro coerente mesmo apos copiar o programa de pasta.
	if cfg.StartWithWindows && !winsys.IsAutoStartEnabled() {
		if err := winsys.SetAutoStart(true); err != nil {
			log.Warnf("nao foi possivel registrar a inicializacao automatica: %v", err)
		}
	}

	name := cfg.DisplayName
	if name == "" {
		name = appName
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sup := supervisor.New(cfg, checker, connector, log, nil)
	go sup.Run(ctx)
	go waitForSignal(ctx, cancel, log)

	trayui.Run(trayui.Deps{
		AppName:        name,
		Supervisor:     sup,
		LogPath:        log.Path(),
		ConfigPath:     configPath,
		NotifyOnChange: cfg.NotifyOnChange,
		Log:            log,
		StatePath:      statePath,
		CurrentProfile: cfg.Connect.Profile,
		Quit:           cancel,
	})

	log.Infof("programa encerrado")
	return nil
}

// baseDirectory devolve a pasta do executavel, onde ficam config, log e cofre.
func baseDirectory() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("localizando o executavel: %w", err)
	}
	return filepath.Dir(exe), nil
}

// storePassword le a senha da entrada padrao e a grava cifrada. A senha vem
// pela entrada padrao, e nao por argumento, para nao ficar no historico do
// terminal nem visivel na lista de processos.
func storePassword(store *winsys.SecretStore, name string) error {
	fmt.Printf("Digite a senha da VPN para %q e tecle Enter: ", name)
	senha, err := readLine(os.Stdin)
	if err != nil {
		return fmt.Errorf("lendo a senha: %w", err)
	}
	if senha == "" {
		return fmt.Errorf("a senha nao pode ser vazia")
	}
	if err := store.Set(name, senha); err != nil {
		return err
	}
	fmt.Printf("Senha gravada com seguranca sob o nome %q.\n", name)
	fmt.Println("Aponte connect.secretName para esse nome no config.json.")
	return nil
}

// reportSingleCheck roda uma unica verificacao, util para diagnosticar a
// configuracao sem deixar o programa residente.
func reportSingleCheck(checker vpn.Checker) error {
	fmt.Println("Verificando:", checker.Describe())

	connected, err := checker.Check(context.Background())
	if err != nil {
		return fmt.Errorf("a verificacao nao pode ser executada: %w", err)
	}
	if connected {
		fmt.Println("Resultado: VPN CONECTADA")
		return nil
	}
	fmt.Println("Resultado: VPN DESCONECTADA")
	return nil
}

// waitForSignal encerra o programa de forma limpa em Ctrl+C ou logoff.
func waitForSignal(ctx context.Context, cancel context.CancelFunc, log *logging.Logger) {
	sinais := make(chan os.Signal, 1)
	signal.Notify(sinais, os.Interrupt, syscall.SIGTERM)
	select {
	case <-ctx.Done():
	case s := <-sinais:
		log.Infof("sinal %s recebido; encerrando", s)
		cancel()
	}
}

// readLine le uma linha completa, tolerando as duas convencoes de fim de linha.
func readLine(f *os.File) (string, error) {
	buf := make([]byte, 0, 128)
	one := make([]byte, 1)
	for {
		n, err := f.Read(one)
		if n == 0 || err != nil {
			break
		}
		if one[0] == '\n' {
			break
		}
		buf = append(buf, one[0])
	}
	return strings.TrimRight(string(buf), "\r"), nil
}
