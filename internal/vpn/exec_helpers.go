package vpn

import (
	"errors"
	"os/exec"
)

// asExitError separa "o comando rodou e devolveu erro" de "o comando nem rodou".
// A primeira situacao e uma resposta; a segunda e uma falha do monitor.
func asExitError(err error, target **exec.ExitError) bool {
	return errors.As(err, target)
}
