//go:build windows

package vpn

import (
	"context"
	"fmt"
	"net"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Este arquivo troca o ping.exe pela API ICMP do proprio Windows.
//
// Motivo: o ping.exe devolve codigo de saida ZERO quando um roteador responde
// "Host de destino inacessivel" -- ele conta essa resposta de erro como pacote
// recebido, com 0% de perda. Confiar no codigo de saida faz o monitor concluir
// que a VPN esta no ar justamente quando ela caiu, e nunca reconectar.
//
// IcmpSendEcho devolve o status real do eco, entao a diferenca entre "o destino
// respondeu" e "alguem respondeu por ele" fica explicita, sem depender do
// idioma do Windows.

// Status devolvidos em ICMP_ECHO_REPLY.Status. Apenas ipSuccess significa que o
// proprio destino respondeu.
const (
	ipSuccess             = 0
	ipDestNetUnreachable  = 11002
	ipDestHostUnreachable = 11003
	ipDestProtUnreachable = 11004
	ipDestPortUnreachable = 11005
	ipReqTimedOut         = 11010
)

// cargaPingBytes e o tamanho do dado enviado, igual ao padrao do ping do Windows.
const cargaPingBytes = 32

// icmpEchoReply espelha ICMP_ECHO_REPLY do iphlpapi.h no layout de 64 bits.
// So o campo Status e lido; o resto existe para o tamanho da estrutura bater
// com o que a API escreve no buffer.
type icmpEchoReply struct {
	Address       uint32
	Status        uint32
	RoundTripTime uint32
	DataSize      uint16
	Reserved      uint16
	Data          uintptr
	TTL           uint8
	Tos           uint8
	Flags         uint8
	OptionsSize   uint8
	_             [4]byte // alinhamento antes do ponteiro seguinte
	OptionsData   uintptr
}

var (
	iphlpapi          = windows.NewLazySystemDLL("iphlpapi.dll")
	procIcmpCreate    = iphlpapi.NewProc("IcmpCreateFile")
	procIcmpSendEcho  = iphlpapi.NewProc("IcmpSendEcho")
	procIcmpCloseFile = iphlpapi.NewProc("IcmpCloseHandle")
)

// icmpEcho envia um eco ICMP e responde se o proprio destino respondeu.
//
// O bool devolvido diz se o host esta no ar. O erro descreve por que o teste
// nao pode ser feito -- e nao a ausencia de resposta, que e um resultado valido.
func icmpEcho(ctx context.Context, host string, timeout time.Duration) (bool, error) {
	destino, err := resolverIPv4(ctx, host)
	if err != nil {
		return false, err
	}

	handle, _, errno := procIcmpCreate.Call()
	if handle == 0 || handle == uintptr(windows.InvalidHandle) {
		return false, fmt.Errorf("abrindo o canal ICMP: %w", errno)
	}
	defer procIcmpCloseFile.Call(handle)

	carga := make([]byte, cargaPingBytes)
	for i := range carga {
		carga[i] = byte('a' + i%23)
	}

	// O buffer precisa comportar a estrutura de resposta, a carga de volta e a
	// margem que a documentacao pede para mensagens de erro ICMP.
	resposta := make([]byte, unsafe.Sizeof(icmpEchoReply{})+uintptr(len(carga))+8+64)

	milissegundos := timeout.Milliseconds()
	if milissegundos <= 0 {
		milissegundos = 1000
	}

	n, _, _ := procIcmpSendEcho.Call(
		handle,
		uintptr(destino),
		uintptr(unsafe.Pointer(&carga[0])),
		uintptr(len(carga)),
		0, // sem opcoes de IP
		uintptr(unsafe.Pointer(&resposta[0])),
		uintptr(len(resposta)),
		uintptr(milissegundos),
	)
	if n == 0 {
		// Nenhuma resposta chegou. Isso e o retrato normal de VPN fora do ar,
		// e nao uma falha do monitor.
		return false, nil
	}

	reply := (*icmpEchoReply)(unsafe.Pointer(&resposta[0]))
	return reply.Status == ipSuccess, nil
}

// resolverIPv4 transforma o host configurado no inteiro que a API espera.
// A API so trabalha com IPv4; um nome que so resolva para IPv6 e recusado com
// mensagem clara, em vez de falhar de forma obscura mais adiante.
func resolverIPv4(ctx context.Context, host string) (uint32, error) {
	if ip := net.ParseIP(host); ip != nil {
		return ipv4ParaUint32(ip)
	}

	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return 0, fmt.Errorf("resolvendo %q: %w", host, err)
	}
	if len(ips) == 0 {
		return 0, fmt.Errorf("o nome %q nao resolveu para nenhum endereco IPv4", host)
	}
	return ipv4ParaUint32(ips[0])
}

// ipv4ParaUint32 monta o IPAddr na ordem de bytes que a API do Windows espera.
func ipv4ParaUint32(ip net.IP) (uint32, error) {
	v4 := ip.To4()
	if v4 == nil {
		return 0, fmt.Errorf("%s nao e um endereco IPv4", ip)
	}
	return uint32(v4[0]) | uint32(v4[1])<<8 | uint32(v4[2])<<16 | uint32(v4[3])<<24, nil
}
