//go:build !windows

package vpn

import "os/exec"

func hideWindow(_ *exec.Cmd) {}
