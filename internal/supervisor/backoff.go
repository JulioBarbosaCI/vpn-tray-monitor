package supervisor

import "time"

// backoff cresce o intervalo entre tentativas de reconexao para nao martelar
// um concentrador de VPN que ja esta fora do ar, e volta ao intervalo normal
// assim que a conexao se restabelece.
type backoff struct {
	base    time.Duration
	max     time.Duration
	current time.Duration
}

func newBackoff(base, max time.Duration) *backoff {
	return &backoff{base: base, max: max, current: base}
}

// next dobra o intervalo, respeitando o teto configurado.
func (b *backoff) next() time.Duration {
	current := b.current
	doubled := b.current * 2
	if doubled > b.max {
		doubled = b.max
	}
	b.current = doubled
	return current
}

// reset devolve o intervalo ao valor normal apos uma conexao bem-sucedida.
func (b *backoff) reset() {
	b.current = b.base
}

// peek informa o intervalo que sera usado na proxima espera, sem avanca-lo.
func (b *backoff) peek() time.Duration {
	return b.current
}
