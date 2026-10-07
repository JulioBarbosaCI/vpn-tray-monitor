package vpn

import "strings"

// redactMask substitui qualquer segredo antes de o texto chegar ao log.
const redactMask = "***"

// redact remove do texto os argumentos sensiveis passados ao processo.
// O rasdial recebe a senha na linha de comando, e a saida de erro pode
// ecoa-la de volta; sem isso a senha da VPN acabaria gravada em disco.
func redact(text string, args []string) string {
	if text == "" {
		return text
	}
	// A senha do rasdial e sempre o ultimo argumento quando ha usuario.
	const minArgsWithPassword = 3
	if len(args) >= minArgsWithPassword {
		if secret := args[len(args)-1]; secret != "" {
			text = strings.ReplaceAll(text, secret, redactMask)
		}
	}
	return text
}
