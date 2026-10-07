package config

import (
	"bytes"
	"io"
)

// newTrimmer entrega o JSON sem o BOM que o Bloco de Notas do Windows
// costuma gravar, que faria o decoder falhar no primeiro byte.
func newTrimmer(raw []byte) io.Reader {
	return bytes.NewReader(bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF}))
}
