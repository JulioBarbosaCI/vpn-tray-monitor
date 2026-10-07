package trayui

import (
	"fyne.io/systray"

	"github.com/guibsu/vpn-tray-monitor/internal/state"
	"github.com/guibsu/vpn-tray-monitor/internal/vpn"
)

// escolhaVPN e uma entrada do submenu de conexoes.
type escolhaVPN struct {
	nome string
	item *systray.MenuItem
}

// montarMenuVPN cria o submenu que lista as conexoes VPN cadastradas no
// Windows e deixa o usuario escolher qual sera reconectada.
//
// A lista vem do proprio Windows, e nao de um arquivo de configuracao: assim o
// usuario escolhe pelo nome que ja conhece, sem risco de digitar errado.
func (t *tray) montarMenuVPN() {
	t.itemVPN = systray.AddMenuItem("Conexão VPN", "Escolha qual VPN o programa deve reconectar")
	t.preencherVPNs()
}

// preencherVPNs coloca no submenu as conexoes encontradas no Windows.
func (t *tray) preencherVPNs() {
	nomes := vpn.ListWindowsVPNs()

	if len(nomes) == 0 {
		vazio := t.itemVPN.AddSubMenuItem("Nenhuma VPN cadastrada no Windows", "")
		vazio.Disable()
		t.atualizarRotuloVPN("")
		return
	}

	for _, nome := range nomes {
		item := t.itemVPN.AddSubMenuItemCheckbox(nome, "Usar esta conexão", nome == t.perfilAtual)
		escolha := escolhaVPN{nome: nome, item: item}
		t.escolhas = append(t.escolhas, escolha)
		go t.aguardarEscolha(escolha)
	}
	t.atualizarRotuloVPN(t.perfilAtual)
}

// aguardarEscolha reage ao clique em uma das conexoes do submenu.
func (t *tray) aguardarEscolha(e escolhaVPN) {
	for range e.item.ClickedCh {
		t.selecionarVPN(e.nome)
	}
}

// selecionarVPN troca a conexao em uso, marca o item no menu e grava a escolha
// para que ela sobreviva ao proximo inicio do programa.
func (t *tray) selecionarVPN(nome string) {
	t.perfilAtual = nome

	for _, e := range t.escolhas {
		if e.nome == nome {
			e.item.Check()
			continue
		}
		e.item.Uncheck()
	}

	t.deps.Supervisor.SetConnector(vpn.NewRasdialFor(nome))
	t.atualizarRotuloVPN(nome)

	if t.deps.StatePath == "" {
		return
	}
	if err := state.Save(t.deps.StatePath, state.State{VPNProfile: nome}); err != nil {
		// Nao poder gravar a escolha nao impede de usa-la agora; so nao
		// sobrevive ao proximo inicio.
		t.deps.Log.Errorf("nao foi possivel guardar a VPN escolhida: %v", err)
	}
}

// atualizarRotuloVPN mostra no menu principal qual conexao esta em uso, para
// que o usuario veja a escolha sem precisar abrir o submenu.
func (t *tray) atualizarRotuloVPN(nome string) {
	if nome == "" {
		t.itemVPN.SetTitle("Conexão VPN: (nenhuma escolhida)")
		return
	}
	t.itemVPN.SetTitle("Conexão VPN: " + nome)
}
