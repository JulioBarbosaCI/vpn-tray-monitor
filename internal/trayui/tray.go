package trayui

import (
	"context"
	"fmt"

	"fyne.io/systray"

	"github.com/guibsu/vpn-tray-monitor/internal/supervisor"
	"github.com/guibsu/vpn-tray-monitor/internal/winsys"
)

// detalheMaxChars limita o texto do menu para nao estourar a largura util.
const detalheMaxChars = 60

// Deps reune o que a bandeja precisa para funcionar, injetado pelo main.
type Deps struct {
	AppName        string
	Supervisor     *supervisor.Supervisor
	LogPath        string
	ConfigPath     string
	NotifyOnChange bool
	Log            supervisor.Logger
	// StatePath e onde a VPN escolhida no menu fica guardada entre execucoes.
	StatePath string
	// CurrentProfile e a conexao em uso quando o programa sobe.
	CurrentProfile string
	// Quit encerra o programa inteiro, nao apenas a bandeja.
	Quit context.CancelFunc
}

// tray guarda os itens do menu para atualiza-los conforme o estado muda.
type tray struct {
	deps Deps

	itemStatus     *systray.MenuItem
	itemDetail     *systray.MenuItem
	itemCheckNow   *systray.MenuItem
	itemReconnect  *systray.MenuItem
	itemPause      *systray.MenuItem
	itemAutoStart  *systray.MenuItem
	itemOpenLog    *systray.MenuItem
	itemOpenConfig *systray.MenuItem
	itemQuit       *systray.MenuItem

	// Submenu de escolha da VPN.
	itemVPN     *systray.MenuItem
	escolhas    []escolhaVPN
	perfilAtual string
}

// Run assume a thread principal e so retorna quando o usuario sai do programa.
func Run(deps Deps) {
	t := &tray{deps: deps, perfilAtual: deps.CurrentProfile}
	systray.Run(t.onReady, t.onExit)
}

func (t *tray) onReady() {
	systray.SetIcon(iconInativa)
	systray.SetTitle(t.deps.AppName)
	systray.SetTooltip(t.deps.AppName + ": verificando...")

	t.itemStatus = systray.AddMenuItem("Verificando...", "Estado atual da VPN")
	t.itemStatus.Disable()
	t.itemDetail = systray.AddMenuItem("", "Detalhe da ultima verificacao")
	t.itemDetail.Disable()
	t.itemDetail.Hide()

	systray.AddSeparator()
	t.itemCheckNow = systray.AddMenuItem("Verificar agora", "Forca uma verificacao imediata")
	t.itemReconnect = systray.AddMenuItem("Reconectar agora", "Executa o comando de conexao imediatamente")
	t.itemPause = systray.AddMenuItemCheckbox("Pausar monitoramento", "Suspende as verificacoes", false)

	systray.AddSeparator()
	t.montarMenuVPN()

	systray.AddSeparator()
	t.itemAutoStart = systray.AddMenuItemCheckbox("Iniciar com o Windows", "Registra o programa no login", winsys.IsAutoStartEnabled())
	t.itemOpenLog = systray.AddMenuItem("Abrir log", "Abre o arquivo de log")
	t.itemOpenConfig = systray.AddMenuItem("Abrir configuracao", "Abre o config.json")

	systray.AddSeparator()
	t.itemQuit = systray.AddMenuItem("Sair", "Encerra o monitor")

	t.deps.Supervisor.Subscribe(t.onStateChange)
	t.render(t.deps.Supervisor.Snapshot())

	go t.handleClicks()
}

// handleClicks trata os cliques do menu em um unico laco, mantendo a
// interacao com a bandeja em um so lugar.
func (t *tray) handleClicks() {
	for {
		select {
		case <-t.itemCheckNow.ClickedCh:
			t.deps.Supervisor.CheckNow()

		case <-t.itemReconnect.ClickedCh:
			t.deps.Supervisor.ForceReconnect()

		case <-t.itemPause.ClickedCh:
			t.togglePause()

		case <-t.itemAutoStart.ClickedCh:
			t.toggleAutoStart()

		case <-t.itemOpenLog.ClickedCh:
			t.openInShell(t.deps.LogPath)

		case <-t.itemOpenConfig.ClickedCh:
			t.openInShell(t.deps.ConfigPath)

		case <-t.itemQuit.ClickedCh:
			t.deps.Quit()
			systray.Quit()
			return
		}
	}
}

// togglePause inverte a pausa e reflete o novo estado na marca do menu.
func (t *tray) togglePause() {
	paused := !t.itemPause.Checked()
	if paused {
		t.itemPause.Check()
	} else {
		t.itemPause.Uncheck()
	}
	t.deps.Supervisor.SetPaused(paused)
}

// toggleAutoStart inverte o registro de inicializacao e reflete o resultado
// real no menu: se a gravacao falhar, a marca nao muda.
func (t *tray) toggleAutoStart() {
	desired := !t.itemAutoStart.Checked()
	if err := winsys.SetAutoStart(desired); err != nil {
		t.deps.Log.Errorf("nao foi possivel alterar a inicializacao automatica: %v", err)
		return
	}
	if desired {
		t.itemAutoStart.Check()
		t.deps.Log.Infof("inicializacao automatica ativada")
		return
	}
	t.itemAutoStart.Uncheck()
	t.deps.Log.Infof("inicializacao automatica desativada")
}

// onStateChange e chamado pelo supervisor a cada transicao de estado.
func (t *tray) onStateChange(snap supervisor.Snapshot) {
	t.render(snap)
	if t.deps.NotifyOnChange {
		t.notify(snap)
	}
}

// render atualiza icone, tooltip e os itens informativos do menu.
func (t *tray) render(snap supervisor.Snapshot) {
	systray.SetIcon(iconFor(snap.State))
	systray.SetTooltip(fmt.Sprintf("%s: %s", t.deps.AppName, snap.State))
	t.itemStatus.SetTitle(snap.State.String())

	detalhe := detailFor(snap)
	if detalhe == "" {
		t.itemDetail.Hide()
		return
	}
	t.itemDetail.SetTitle(detalhe)
	t.itemDetail.Show()
}

// notify mostra o balao do Windows apenas nas transicoes que o usuario
// precisa saber. Avisar a cada verificacao seria ruido.
func (t *tray) notify(snap supervisor.Snapshot) {
	switch snap.State {
	case supervisor.StateConnected:
		winsys.Notify(t.deps.AppName, "VPN conectada.")
	case supervisor.StateDisconnected:
		winsys.Notify(t.deps.AppName, "VPN caiu. Tentando reconectar...")
	}
}

// openInShell abre o arquivo no programa padrao do sistema.
func (t *tray) openInShell(path string) {
	if path == "" {
		return
	}
	if err := openPath(path); err != nil {
		t.deps.Log.Errorf("nao foi possivel abrir %s: %v", path, err)
	}
}

func (t *tray) onExit() {}
