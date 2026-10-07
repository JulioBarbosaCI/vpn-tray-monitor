package vpn

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"strings"
	"unicode/utf16"
)

// O Windows guarda as conexoes de rede discada e VPN em um arquivo de agenda
// chamado rasphone.pbk, no formato INI: cada secao e um perfil.
//
//	[VPN Empresa]
//	Encoding=1
//	Type=2
//
// Ler esse arquivo permite descobrir o nome do perfil sozinho, sem exigir que
// alguem digite no config.json um nome que precisa bater exatamente com o
// cadastrado no Windows -- fonte classica de erro silencioso.

// parsePhonebook extrai os nomes de perfil de um rasphone.pbk.
func parsePhonebook(raw []byte) []string {
	texto := decodeMaybeUTF16(raw)

	var perfis []string
	scanner := bufio.NewScanner(strings.NewReader(texto))
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		linha := strings.TrimSpace(scanner.Text())
		if len(linha) < 3 || !strings.HasPrefix(linha, "[") || !strings.HasSuffix(linha, "]") {
			continue
		}
		nome := strings.TrimSpace(linha[1 : len(linha)-1])
		if nome != "" {
			perfis = append(perfis, nome)
		}
	}
	return perfis
}

// decodeMaybeUTF16 converte o conteudo para texto. O Windows grava o pbk ora
// em ANSI, ora em UTF-16LE conforme a versao; sem tratar os dois, o parser
// enxergaria bytes nulos entre cada letra e nao acharia perfil algum.
func decodeMaybeUTF16(raw []byte) string {
	const bomUTF16LE = "\xff\xfe"
	if !bytes.HasPrefix(raw, []byte(bomUTF16LE)) {
		return string(bytes.TrimPrefix(raw, []byte("\xEF\xBB\xBF")))
	}

	corpo := raw[len(bomUTF16LE):]
	unidades := make([]uint16, 0, len(corpo)/2)
	for i := 0; i+1 < len(corpo); i += 2 {
		unidades = append(unidades, uint16(corpo[i])|uint16(corpo[i+1])<<8)
	}
	return string(utf16.Decode(unidades))
}

// discoverRasProfiles reune os perfis de todas as agendas conhecidas, sem
// repetir nomes que aparecam na agenda do usuario e na da maquina.
func discoverRasProfiles() []string {
	vistos := make(map[string]bool)
	var perfis []string

	for _, caminho := range rasPhonebookPaths() {
		raw, err := os.ReadFile(caminho)
		if err != nil {
			continue // agenda inexistente e normal: nem todo caminho existe
		}
		for _, nome := range parsePhonebook(raw) {
			if vistos[nome] {
				continue
			}
			vistos[nome] = true
			perfis = append(perfis, nome)
		}
	}
	return perfis
}

// resolveRasProfile decide qual perfil usar quando o config.json nao informa um.
//
// Com um unico perfil cadastrado a escolha e obvia e o programa segue sozinho.
// Com varios, ele para e pede que se escolha: adivinhar qual VPN conectar seria
// pior do que uma mensagem clara.
func resolveRasProfile(configurado string) (string, error) {
	if configurado != "" {
		return configurado, nil
	}

	perfis := discoverRasProfiles()
	switch len(perfis) {
	case 0:
		return "", fmt.Errorf("nenhuma conexao VPN encontrada no Windows; " +
			"cadastre a VPN em Configuracoes > Rede > VPN e escolha-a no menu do icone")
	case 1:
		return perfis[0], nil
	default:
		return "", fmt.Errorf("ha %d conexoes cadastradas no Windows (%s); "+
			"escolha qual usar no menu do icone, em \"Conexao VPN\"",
			len(perfis), strings.Join(perfis, ", "))
	}
}
