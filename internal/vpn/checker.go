// Package vpn decide se a VPN esta no ar e sabe restabelece-la.
package vpn

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/guibsu/vpn-tray-monitor/internal/config"
)

// Checker responde se a VPN esta conectada neste instante.
type Checker interface {
	// Check devolve true quando a VPN esta no ar. O erro descreve por que a
	// verificacao nao pode ser concluida, e nao a ausencia de conexao.
	Check(ctx context.Context) (bool, error)
	// Describe explica em uma linha o que esta sendo verificado.
	Describe() string
}

// NewChecker monta o verificador descrito na configuracao.
func NewChecker(c config.Check) (Checker, error) {
	switch c.Kind {
	case config.CheckTCP:
		return tcpChecker{host: c.Host, port: c.Port}, nil
	case config.CheckPing:
		return pingChecker{host: c.Host, timeout: c.Timeout()}, nil
	case config.CheckInterface:
		return interfaceChecker{match: c.InterfaceMatch}, nil
	case config.CheckCommand:
		return commandChecker{command: c.Command, args: c.Args}, nil
	default:
		return nil, fmt.Errorf("check.kind nao suportado: %q", c.Kind)
	}
}

// tcpChecker considera a VPN no ar quando um host interno aceita conexao TCP.
// E a verificacao mais confiavel: prova que a rota existe e que o outro lado responde.
type tcpChecker struct {
	host string
	port int
}

func (t tcpChecker) Check(ctx context.Context) (bool, error) {
	addr := net.JoinHostPort(t.host, strconv.Itoa(t.port))
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		// Recusa, timeout ou rota ausente sao respostas validas: VPN fora.
		return false, nil
	}
	_ = conn.Close()
	return true, nil
}

func (t tcpChecker) Describe() string {
	return fmt.Sprintf("conexao TCP em %s:%d", t.host, t.port)
}

// pingChecker envia um eco ICMP ao host interno. Serve quando o host responde
// ping mas nao expoe nenhuma porta TCP conhecida.
//
// A verificacao NAO usa o ping.exe: ele devolve codigo de saida zero quando um
// roteador responde "host de destino inacessivel", o que faria o monitor
// concluir que a VPN esta no ar justamente quando ela caiu. Veja icmp_windows.go.
type pingChecker struct {
	host    string
	timeout time.Duration
}

func (p pingChecker) Check(ctx context.Context) (bool, error) {
	return icmpEcho(ctx, p.host, p.timeout)
}

func (p pingChecker) Describe() string {
	return fmt.Sprintf("ping em %s", p.host)
}

// interfaceChecker procura um adaptador de rede ativo cujo nome contenha o
// trecho configurado. Detecta a VPN mesmo sem trafego, mas nao prova que a
// outra ponta responde: prefira tcp quando houver um host interno conhecido.
type interfaceChecker struct {
	match string
}

func (i interfaceChecker) Check(ctx context.Context) (bool, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return false, fmt.Errorf("listando adaptadores de rede: %w", err)
	}
	needle := strings.ToLower(i.match)
	for _, iface := range ifaces {
		if !strings.Contains(strings.ToLower(iface.Name), needle) {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		return true, nil
	}
	return false, nil
}

func (i interfaceChecker) Describe() string {
	return fmt.Sprintf("adaptador de rede contendo %q", i.match)
}

// commandChecker delega a decisao a um comando externo: saida 0 = conectada.
type commandChecker struct {
	command string
	args    []string
}

func (c commandChecker) Check(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, c.command, c.args...)
	hideWindow(cmd)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			return false, nil
		}
		return false, fmt.Errorf("executando %s: %w", c.command, err)
	}
	return true, nil
}

func (c commandChecker) Describe() string {
	return fmt.Sprintf("comando %s", c.command)
}
