// Comando geniconos desenha os icones da bandeja e grava um .ico por estado.
//
// Os icones sao gerados em vez de versionados como binarios opacos: assim a
// paleta fica visivel no codigo e uma mudanca de cor e uma linha de diff.
//
// Uso: go run ./tools/geniconos
package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

// Tamanhos embutidos em cada .ico. O Windows escolhe conforme o DPI da barra.
var tamanhos = []int{16, 20, 24, 32, 48}

// icone descreve um estado e a cor que o representa.
type icone struct {
	nome string
	cor  color.RGBA
}

func main() {
	destino := filepath.Join("assets")
	if err := os.MkdirAll(destino, 0o750); err != nil {
		fatal(err)
	}

	icones := []icone{
		{"conectada", color.RGBA{R: 0x22, G: 0xc5, B: 0x5e, A: 0xff}},    // verde
		{"desconectada", color.RGBA{R: 0xef, G: 0x44, B: 0x44, A: 0xff}}, // vermelho
		{"conectando", color.RGBA{R: 0xf5, G: 0x9e, B: 0x0b, A: 0xff}},   // ambar
		{"inativa", color.RGBA{R: 0x9c, G: 0xa3, B: 0xaf, A: 0xff}},      // cinza
	}

	for _, ic := range icones {
		blob, err := montarICO(ic.cor)
		if err != nil {
			fatal(fmt.Errorf("gerando %s: %w", ic.nome, err))
		}
		caminho := filepath.Join(destino, ic.nome+".ico")
		if err := os.WriteFile(caminho, blob, 0o640); err != nil {
			fatal(err)
		}
		fmt.Printf("gerado %s (%d bytes)\n", caminho, len(blob))
	}
}

// desenhar produz um escudo circular preenchido com a cor do estado, com um
// anel branco por fora para o icone nao sumir em barras claras nem escuras.
func desenhar(tamanho int, cor color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, tamanho, tamanho))
	centro := float64(tamanho-1) / 2
	raioExterno := float64(tamanho)/2 - float64(tamanho)*0.06
	raioInterno := raioExterno - math.Max(1, float64(tamanho)*0.14)
	branco := color.RGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}

	for y := 0; y < tamanho; y++ {
		for x := 0; x < tamanho; x++ {
			dx, dy := float64(x)-centro, float64(y)-centro
			dist := math.Hypot(dx, dy)
			switch {
			case dist <= raioInterno:
				img.Set(x, y, cor)
			case dist <= raioExterno:
				img.Set(x, y, branco)
			default:
				img.Set(x, y, color.RGBA{})
			}
		}
	}
	return img
}

// montarICO empacota os varios tamanhos em um unico arquivo .ico.
// Cada entrada guarda um PNG, formato aceito pelo Windows desde o Vista.
func montarICO(cor color.RGBA) ([]byte, error) {
	type entrada struct {
		tamanho int
		dados   []byte
	}

	entradas := make([]entrada, 0, len(tamanhos))
	for _, t := range tamanhos {
		var buf bytes.Buffer
		if err := png.Encode(&buf, desenhar(t, cor)); err != nil {
			return nil, fmt.Errorf("codificando png %dx%d: %w", t, t, err)
		}
		entradas = append(entradas, entrada{tamanho: t, dados: buf.Bytes()})
	}

	const cabecalhoBytes = 6
	const entradaBytes = 16

	var out bytes.Buffer
	// ICONDIR: reservado, tipo 1 (icone), quantidade.
	_ = binary.Write(&out, binary.LittleEndian, uint16(0))
	_ = binary.Write(&out, binary.LittleEndian, uint16(1))
	_ = binary.Write(&out, binary.LittleEndian, uint16(len(entradas)))

	deslocamento := cabecalhoBytes + entradaBytes*len(entradas)
	for _, e := range entradas {
		// Largura e altura 0 significam 256 no formato ICO.
		dimensao := byte(e.tamanho)
		if e.tamanho >= 256 {
			dimensao = 0
		}
		out.WriteByte(dimensao)                                           // largura
		out.WriteByte(dimensao)                                           // altura
		out.WriteByte(0)                                                  // cores da paleta
		out.WriteByte(0)                                                  // reservado
		_ = binary.Write(&out, binary.LittleEndian, uint16(1))            // planos
		_ = binary.Write(&out, binary.LittleEndian, uint16(32))           // bits por pixel
		_ = binary.Write(&out, binary.LittleEndian, uint32(len(e.dados))) // tamanho
		_ = binary.Write(&out, binary.LittleEndian, uint32(deslocamento)) // posicao
		deslocamento += len(e.dados)
	}
	for _, e := range entradas {
		out.Write(e.dados)
	}
	return out.Bytes(), nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "erro:", err)
	os.Exit(1)
}
