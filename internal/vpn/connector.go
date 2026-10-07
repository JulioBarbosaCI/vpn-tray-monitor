package vpn

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/guibsu/vpn-tray-monitor/internal/config"
)

// SecretResolver entrega a senha da VPN sob demanda. Fica como interface para
// que a senha so saia do cofre no instante da conexao e nunca seja mantida em
// memoria pelo resto do programa.
type SecretResolver interface {
	Secret(name string) (string, error)
}

// Connector restabelece a conexao da VPN.
type Connector interface {
	// Connect tenta conectar. Devolve erro quando a tentativa falha.
	Connect(ctx context.Context) error
	// Describe explica em uma linha o que sera executado.
	Describe() string
}

// NewConnector monta o reconector descrito na configuracao.
func NewConnector(c config.Connect, secrets SecretResolver) (Connector, error) {
	switch c.Kind {
	case config.ConnectRasdial:
		return rasdialConnector{cfg: c, secrets: secrets}, nil
	case config.ConnectOpenVPN:
		return openVPNConnector{cfg: c}, nil
	case config.ConnectService:
		return serviceConnector{name: c.Profile}, nil
	case config.ConnectSchedTsk:
		return taskConnector{name: c.Profile}, nil
	case config.ConnectCommand:
		return commandConnector{command: c.Command, args: c.Args}, nil
	default:
		return nil, fmt.Errorf("connect.kind nao suportado: %q", c.Kind)
	}
}

// runHidden executa o comando sem abrir janela e devolve a saida combinada,
// que e o que interessa no log quando a conexao falha.
func runHidden(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	hideWindow(cmd)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

// rasdialConnector usa a API de Acesso Remoto do Windows (RasDial).
type rasdialConnector struct {
	cfg     config.Connect
	secrets SecretResolver
}

func (r rasdialConnector) Connect(ctx context.Context) error {
	// O perfil pode vir vazio: nesse caso o proprio Windows informa qual e,
	// pela agenda de conexoes.
	perfil, err := resolveRasProfile(r.cfg.Profile)
	if err != nil {
		return err
	}

	// Sem usuario configurado valem as credenciais que o proprio Windows
	// guardou na conexao. Informar usuario e senha aqui as sobrescreve, para
	// quem prefere manter a senha no cofre deste programa.
	usuario := r.cfg.Username
	senha := ""
	if usuario != "" && r.cfg.SecretName != "" {
		if r.secrets == nil {
			return fmt.Errorf("cofre de senhas indisponivel para o segredo %q", r.cfg.SecretName)
		}
		var err error
		senha, err = r.secrets.Secret(r.cfg.SecretName)
		if err != nil {
			return fmt.Errorf("lendo a senha %q: %w", r.cfg.SecretName, err)
		}
	}

	if err := dialWindowsVPN(perfil, usuario, senha); err != nil {
		return fmt.Errorf("nao foi possivel conectar em %q: %w", perfil, err)
	}
	return nil
}

func (r rasdialConnector) Describe() string {
	if r.cfg.Profile == "" {
		return "conexao VPN cadastrada no Windows"
	}
	return fmt.Sprintf("conexao VPN %q do Windows", r.cfg.Profile)
}

// openVPNConnector aciona o OpenVPN GUI (ou o binario informado em command).
type openVPNConnector struct {
	cfg config.Connect
}

func (o openVPNConnector) Connect(ctx context.Context) error {
	binary := o.cfg.Command
	if binary == "" {
		binary = `C:\Program Files\OpenVPN\bin\openvpn-gui.exe`
	}
	args := o.cfg.Args
	if len(args) == 0 {
		args = []string{"--connect", o.cfg.Profile}
	}
	if out, err := runHidden(ctx, binary, args...); err != nil {
		return fmt.Errorf("openvpn falhou: %w: %s", err, out)
	}
	return nil
}

func (o openVPNConnector) Describe() string {
	return fmt.Sprintf("OpenVPN no perfil %q", o.cfg.Profile)
}

// serviceConnector sobe um servico do Windows (usado por WireGuard e por
// clientes corporativos que expoem a VPN como servico).
type serviceConnector struct {
	name string
}

func (s serviceConnector) Connect(ctx context.Context) error {
	if out, err := runHidden(ctx, "net", "start", s.name); err != nil {
		// O servico ja em execucao nao e falha: o proximo check confirma o estado.
		if strings.Contains(strings.ToLower(out), "already been started") ||
			strings.Contains(strings.ToLower(out), "ja foi iniciado") {
			return nil
		}
		return fmt.Errorf("net start %s falhou: %w: %s", s.name, err, out)
	}
	return nil
}

func (s serviceConnector) Describe() string {
	return fmt.Sprintf("servico do Windows %q", s.name)
}

// taskConnector dispara uma tarefa agendada ja existente. Util quando a
// conexao exige privilegio elevado e a tarefa foi criada com esse privilegio.
type taskConnector struct {
	name string
}

func (t taskConnector) Connect(ctx context.Context) error {
	if out, err := runHidden(ctx, "schtasks", "/run", "/tn", t.name); err != nil {
		return fmt.Errorf("schtasks /run %s falhou: %w: %s", t.name, err, out)
	}
	return nil
}

func (t taskConnector) Describe() string {
	return fmt.Sprintf("tarefa agendada %q", t.name)
}

// commandConnector executa um comando livre definido pelo operador.
type commandConnector struct {
	command string
	args    []string
}

func (c commandConnector) Connect(ctx context.Context) error {
	if out, err := runHidden(ctx, c.command, c.args...); err != nil {
		return fmt.Errorf("%s falhou: %w: %s", c.command, err, out)
	}
	return nil
}

func (c commandConnector) Describe() string {
	return fmt.Sprintf("comando %s", c.command)
}
