//go:build !windows

package vpn

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"time"
)

// icmpEcho usa o ping do sistema fora do Windows. Serve apenas ao ambiente de
// desenvolvimento: o binario de producao e sempre o de Windows, onde a
// verificacao vai direto na API ICMP.
//
// Aqui o codigo de saida e confiavel -- o ping do Linux devolve 1 tanto para
// timeout quanto para "inacessivel", ao contrario do ping.exe do Windows.
func icmpEcho(ctx context.Context, host string, timeout time.Duration) (bool, error) {
	segundos := int(timeout.Seconds())
	if segundos < 1 {
		segundos = 1
	}

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", strconv.Itoa(segundos), host)
	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
