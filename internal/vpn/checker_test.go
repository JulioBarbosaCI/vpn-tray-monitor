package vpn

import (
	"context"
	"net"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/guibsu/vpn-tray-monitor/internal/config"
)

func TestTCPCheckerVeQuePortaAbertaEstaNoAr(t *testing.T) {
	// Arrange: um listener local faz o papel do host interno da VPN.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("abrindo o listener de teste: %v", err)
	}
	defer ln.Close()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()

	host, portaTexto, _ := net.SplitHostPort(ln.Addr().String())
	porta, _ := strconv.Atoi(portaTexto)
	checker := tcpChecker{host: host, port: porta}

	// Act
	connected, err := checker.Check(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("a verificacao nao deveria falhar: %v", err)
	}
	if !connected {
		t.Fatal("porta aberta deveria ser reportada como conectada")
	}
}

func TestTCPCheckerTrataPortaFechadaComoDesconectadaENaoComoErro(t *testing.T) {
	// Arrange: porta fechada e a situacao normal de VPN fora do ar.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("abrindo o listener de teste: %v", err)
	}
	host, portaTexto, _ := net.SplitHostPort(ln.Addr().String())
	porta, _ := strconv.Atoi(portaTexto)
	ln.Close() // fecha para garantir que ninguem atende

	checker := tcpChecker{host: host, port: porta}

	// Act
	connected, err := checker.Check(context.Background())

	// Assert: distinguir os dois casos importa, porque erro suspende a
	// reconexao e desconectado a dispara.
	if err != nil {
		t.Fatalf("porta fechada e resposta valida, nao erro: %v", err)
	}
	if connected {
		t.Fatal("porta fechada deveria ser reportada como desconectada")
	}
}

func TestTCPCheckerRespeitaOTimeoutDoContexto(t *testing.T) {
	// Arrange: endereco nao roteavel, que nao responde nem recusa.
	checker := tcpChecker{host: "10.255.255.1", port: 65000}
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Act
	inicio := time.Now()
	connected, err := checker.Check(ctx)

	// Assert
	if err != nil {
		t.Fatalf("timeout e resposta valida, nao erro: %v", err)
	}
	if connected {
		t.Fatal("host inalcancavel deveria ser reportado como desconectado")
	}
	if decorrido := time.Since(inicio); decorrido > 3*time.Second {
		t.Fatalf("a verificacao ignorou o timeout do contexto: levou %s", decorrido)
	}
}

func TestInterfaceCheckerEncontraAdaptadorDeLoopback(t *testing.T) {
	// Arrange: loopback existe e esta ativa em qualquer maquina.
	nome := "lo"
	if runtime.GOOS == "windows" {
		nome = "Loopback"
	}
	checker := interfaceChecker{match: nome}

	// Act
	connected, err := checker.Check(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("listar adaptadores nao deveria falhar: %v", err)
	}
	if !connected {
		t.Skipf("adaptador %q nao encontrado neste ambiente", nome)
	}
}

func TestInterfaceCheckerNaoEncontraAdaptadorInexistente(t *testing.T) {
	// Arrange
	checker := interfaceChecker{match: "adaptador-que-nao-existe-xyz"}

	// Act
	connected, err := checker.Check(context.Background())

	// Assert
	if err != nil {
		t.Fatalf("listar adaptadores nao deveria falhar: %v", err)
	}
	if connected {
		t.Fatal("adaptador inexistente nao deveria ser reportado como conectado")
	}
}

func TestCommandCheckerUsaOCodigoDeSaida(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("o teste usa binarios do shell POSIX")
	}

	// Arrange & Act & Assert: saida 0 significa conectada.
	sucesso := commandChecker{command: "true"}
	connected, err := sucesso.Check(context.Background())
	if err != nil {
		t.Fatalf("comando valido nao deveria dar erro: %v", err)
	}
	if !connected {
		t.Fatal("saida 0 deveria significar conectada")
	}

	// Saida diferente de 0 significa desconectada, e nao falha do monitor.
	falha := commandChecker{command: "false"}
	connected, err = falha.Check(context.Background())
	if err != nil {
		t.Fatalf("saida diferente de zero e resposta, nao erro: %v", err)
	}
	if connected {
		t.Fatal("saida diferente de 0 deveria significar desconectada")
	}
}

func TestCommandCheckerReportaErroQuandoOBinarioNaoExiste(t *testing.T) {
	// Arrange: binario ausente e falha de configuracao, nao VPN fora.
	checker := commandChecker{command: "binario-que-nao-existe-xyz"}

	// Act
	_, err := checker.Check(context.Background())

	// Assert
	if err == nil {
		t.Fatal("binario inexistente deveria produzir erro, para nao reconectar as cegas")
	}
}

func TestNewCheckerMontaCadaTipoConfigurado(t *testing.T) {
	// Arrange
	casos := []struct {
		nome     string
		check    config.Check
		esperado string
	}{
		{"tcp", config.Check{Kind: config.CheckTCP, Host: "10.0.0.1", Port: 445}, "TCP"},
		{"ping", config.Check{Kind: config.CheckPing, Host: "10.0.0.1"}, "ping"},
		{"interface", config.Check{Kind: config.CheckInterface, InterfaceMatch: "VPN"}, "adaptador"},
		{"command", config.Check{Kind: config.CheckCommand, Command: "x"}, "comando"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			// Act
			checker, err := NewChecker(caso.check)

			// Assert
			if err != nil {
				t.Fatalf("NewChecker devolveu erro: %v", err)
			}
			if !strings.Contains(checker.Describe(), caso.esperado) {
				t.Fatalf("Describe() = %q, deveria conter %q", checker.Describe(), caso.esperado)
			}
		})
	}
}

func TestNewCheckerRecusaTipoDesconhecido(t *testing.T) {
	// Arrange & Act
	_, err := NewChecker(config.Check{Kind: "telepatia"})

	// Assert
	if err == nil {
		t.Fatal("tipo desconhecido deveria ser recusado")
	}
}

func TestNewConnectorMontaCadaTipoConfigurado(t *testing.T) {
	// Arrange
	casos := []struct {
		nome     string
		connect  config.Connect
		esperado string
	}{
		{"rasdial", config.Connect{Kind: config.ConnectRasdial, Profile: "VPN"}, "conexao VPN"},
		{"openvpn", config.Connect{Kind: config.ConnectOpenVPN, Profile: "perfil"}, "OpenVPN"},
		{"service", config.Connect{Kind: config.ConnectService, Profile: "Tunel"}, "servico"},
		{"task", config.Connect{Kind: config.ConnectSchedTsk, Profile: "Tarefa"}, "tarefa"},
		{"command", config.Connect{Kind: config.ConnectCommand, Command: "x"}, "comando"},
	}

	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			// Act
			connector, err := NewConnector(caso.connect, nil)

			// Assert
			if err != nil {
				t.Fatalf("NewConnector devolveu erro: %v", err)
			}
			if !strings.Contains(connector.Describe(), caso.esperado) {
				t.Fatalf("Describe() = %q, deveria conter %q", connector.Describe(), caso.esperado)
			}
		})
	}
}

func TestNewConnectorRecusaTipoDesconhecido(t *testing.T) {
	// Arrange & Act
	_, err := NewConnector(config.Connect{Kind: "telepatia"}, nil)

	// Assert
	if err == nil {
		t.Fatal("tipo desconhecido deveria ser recusado")
	}
}

func TestRasdialFalhaQuandoOSegredoNaoPodeSerLido(t *testing.T) {
	// Arrange: usuario configurado com segredo, mas sem cofre disponivel.
	// Sem usuario, valeriam as credenciais guardadas pelo proprio Windows.
	connector := rasdialConnector{
		cfg: config.Connect{
			Kind:       config.ConnectRasdial,
			Profile:    "VPN",
			Username:   "usuario",
			SecretName: "vpn",
		},
		secrets: nil,
	}

	// Act
	err := connector.Connect(context.Background())

	// Assert
	if err == nil {
		t.Fatal("sem cofre de senhas a conexao deveria falhar de forma explicita")
	}
	if !strings.Contains(err.Error(), "cofre") {
		t.Fatalf("a mensagem deveria explicar o problema do cofre; recebida: %v", err)
	}
}

func TestRedactRemoveASenhaDaSaida(t *testing.T) {
	// Arrange: o rasdial recebe a senha como ultimo argumento e pode
	// devolve-la na mensagem de erro.
	args := []string{"VPN Empresa", "usuario", "SenhaSecreta123"}
	saida := "Erro ao conectar com a senha SenhaSecreta123 informada"

	// Act
	limpo := redact(saida, args)

	// Assert
	if strings.Contains(limpo, "SenhaSecreta123") {
		t.Fatalf("a senha continua visivel apos redact: %q", limpo)
	}
	if !strings.Contains(limpo, redactMask) {
		t.Fatalf("a senha deveria ter sido substituida pela mascara: %q", limpo)
	}
}

func TestRedactNaoAlteraSaidaSemSenha(t *testing.T) {
	// Arrange: conexao sem usuario nao tem senha entre os argumentos.
	args := []string{"VPN Empresa"}
	saida := "Conectando..."

	// Act
	limpo := redact(saida, args)

	// Assert
	if limpo != saida {
		t.Fatalf("saida = %q, esperado inalterada %q", limpo, saida)
	}
}
