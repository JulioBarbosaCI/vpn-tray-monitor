//go:build windows

package vpn

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Este arquivo disca a VPN pela API RAS, e nao pelo utilitario rasdial.exe.
//
// Motivo, verificado em uma maquina de teste: `rasdial "Nome"` sem argumentos
// NAO usa as credenciais guardadas na conexao. Ele envia o usuario logado do
// Windows, e o servidor recusa -- o log do concentrador mostrava
// "No CHAP secret found for authenticating dev", sendo `dev` o usuario da
// maquina, e o rasdial devolvia erro 628. Mesmo com a senha salva, o rasphone
// ainda exige que alguem clique em "Conectar".
//
// RasDial, ao contrario, aceita os parametros que o proprio Windows guardou
// (RasGetEntryDialParams), que e exatamente o que a janela do Windows faz.

// Limites de tamanho das cadeias em RASDIALPARAMSW, do ras.h.
const (
	rasMaxEntryName      = 256
	rasMaxPhoneNumber    = 128
	rasMaxCallbackNumber = 128
	rasMaxUserName       = 256
	rasMaxPassword       = 256
	rasMaxDomain         = 15
)

// errorInvalidSize e devolvido quando dwSize nao corresponde ao que esta
// versao do Windows espera.
const errorInvalidSize = 632

// rasDialParams espelha RASDIALPARAMSW na variante com dwIfIndex (Windows 7+).
type rasDialParams struct {
	Size           uint32
	EntryName      [rasMaxEntryName + 1]uint16
	PhoneNumber    [rasMaxPhoneNumber + 1]uint16
	CallbackNumber [rasMaxCallbackNumber + 1]uint16
	UserName       [rasMaxUserName + 1]uint16
	Password       [rasMaxPassword + 1]uint16
	Domain         [rasMaxDomain + 1]uint16
	SubEntry       uint32
	CallbackID     uintptr
	IfIndex        uint32
}

var (
	rasapi32                 = windows.NewLazySystemDLL("rasapi32.dll")
	procRasDial              = rasapi32.NewProc("RasDialW")
	procRasHangUp            = rasapi32.NewProc("RasHangUpW")
	procRasGetEntryDialParam = rasapi32.NewProc("RasGetEntryDialParamsW")
	procRasGetErrorString    = rasapi32.NewProc("RasGetErrorStringW")
)

// tamanhosRasDialParams lista os dwSize aceitos, do mais novo para o mais
// antigo. Versoes de Windows diferentes esperam estruturas diferentes, e
// errar o tamanho devolve 632 em vez de conectar.
func tamanhosRasDialParams() []uint32 {
	completo := uint32(unsafe.Sizeof(rasDialParams{}))
	semIfIndex := completo - uint32(unsafe.Sizeof(uint32(0)))
	return []uint32{completo, semIfIndex}
}

// copiarParaUTF16 grava o texto no vetor de tamanho fixo, truncando com
// seguranca e sempre deixando o terminador nulo.
func copiarParaUTF16(destino []uint16, texto string) {
	for i := range destino {
		destino[i] = 0
	}
	if texto == "" {
		return
	}
	codificado, err := windows.UTF16FromString(texto)
	if err != nil {
		return
	}
	if len(codificado) > len(destino) {
		codificado = codificado[:len(destino)]
		codificado[len(codificado)-1] = 0
	}
	copy(destino, codificado)
}

// carregarParamsSalvos pede ao Windows os parametros guardados da conexao,
// incluindo usuario e senha quando o usuario marcou "lembrar credenciais".
// Devolve tambem o dwSize que a API aceitou, para reusar na discagem.
func carregarParamsSalvos(entrada string) (rasDialParams, uint32, error) {
	var ultimoErro uintptr

	for _, tamanho := range tamanhosRasDialParams() {
		var params rasDialParams
		params.Size = tamanho
		copiarParaUTF16(params.EntryName[:], entrada)

		var temSenha uint32
		ret, _, _ := procRasGetEntryDialParam.Call(
			0, // phonebook padrao
			uintptr(unsafe.Pointer(&params)),
			uintptr(unsafe.Pointer(&temSenha)),
		)
		if ret == 0 {
			return params, tamanho, nil
		}
		ultimoErro = ret
		if ret != errorInvalidSize {
			break
		}
	}

	return rasDialParams{}, 0, fmt.Errorf("lendo as credenciais salvas: %s", descreverErroRas(ultimoErro))
}

// dialWindowsVPN estabelece a conexao pela API do Windows.
//
// Quando usuario e senha vem vazios, valem os que o Windows ja tem guardados.
// Informa-los sobrescreve os salvos, que e o caminho para quem prefere manter
// a senha no cofre do proprio programa.
func dialWindowsVPN(entrada, usuario, senha string) error {
	params, tamanho, err := carregarParamsSalvos(entrada)
	if err != nil {
		return err
	}

	// O dwSize aceito por RasGetEntryDialParams e o mesmo que RasDial espera.
	params.Size = tamanho
	copiarParaUTF16(params.EntryName[:], entrada)
	if usuario != "" {
		copiarParaUTF16(params.UserName[:], usuario)
	}
	if senha != "" {
		copiarParaUTF16(params.Password[:], senha)
	}

	var conexao windows.Handle
	ret, _, _ := procRasDial.Call(
		0, // sem extensoes
		0, // phonebook padrao
		uintptr(unsafe.Pointer(&params)),
		0, // sem notificacoes: a chamada bloqueia ate concluir
		0,
		uintptr(unsafe.Pointer(&conexao)),
	)
	if ret != 0 {
		// Uma tentativa malsucedida ainda pode deixar a conexao meio aberta;
		// desligar evita acumular sessoes penduradas no concentrador.
		if conexao != 0 {
			procRasHangUp.Call(uintptr(conexao))
		}
		return fmt.Errorf("%s", descreverErroRas(ret))
	}
	return nil
}

// descreverErroRas traduz o codigo numerico para a mensagem do proprio Windows,
// que sai no idioma do sistema e e o que o usuario consegue pesquisar.
func descreverErroRas(codigo uintptr) string {
	if codigo == 0 {
		return "sem erro"
	}

	buffer := make([]uint16, 512)
	ret, _, _ := procRasGetErrorString.Call(
		codigo,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)
	if ret != 0 {
		return fmt.Sprintf("erro %d do Acesso Remoto", codigo)
	}
	return fmt.Sprintf("erro %d: %s", codigo, windows.UTF16ToString(buffer))
}
