# Monitor de VPN (bandeja do Windows)

Programa residente que fica na bandeja do Windows, verifica de tempos em tempos
se a VPN está no ar e a reconecta sozinho quando ela cai.

Um único `.exe` de 3,6 MB, sem instalador, sem .NET, sem Python — é só copiar a
pasta para a máquina e executar.

## Estado atual

Configurado para o cenário real: VPN nativa do Windows, credenciais já salvas no
cliente, detecção por ping em `10.254.1.172`.

Testado numa VM Windows 11 com uma conexão VPN nativa de verdade: detecta a
queda, descobre sozinho o nome do perfil na agenda do Windows, chama o `rasdial`
e registra tudo em log.

O que falta é apontar para o servidor de VPN real — na VM de teste o servidor é
fictício, então o `rasdial` termina em `0x80072af9` (host desconhecido), que é o
resultado correto para um endereço que não existe.

### Descoberta do perfil

`connect.profile` pode ficar vazio. O programa lê a agenda de conexões do Windows
(`rasphone.pbk`, nas pastas do usuário e da máquina) e usa a VPN cadastrada. Com
mais de uma cadastrada ele recusa adivinhar e pede que se informe qual — errar a
VPN seria pior que uma mensagem clara.

## Como usar

1. Copie a pasta com `vpnmon.exe` e `config.json` para a máquina.
2. Execute o `vpnmon.exe`. O ícone aparece na bandeja.
3. **Clique com o botão direito no ícone**, abra **Conexão VPN** e escolha a
   conexão que ele deve reconectar.

Pronto. A escolha fica guardada e vale nos próximos inícios.

### O menu do ícone

```
Desconectada
Última verificação: 16:41:52  |  reconexões: 3
─────────────────────────────
Verificar agora
Reconectar agora
Pausar monitoramento
─────────────────────────────
Conexão VPN: VPN Empresa      ▸   VPN Empresa  ✓
─────────────────────────────     VPN Filial
Iniciar com o Windows         ✓
Abrir log
Abrir configuração
─────────────────────────────
Sair
```

A lista de VPNs vem do próprio Windows — são as mesmas que aparecem em
**Configurações › Rede › VPN**. Trocar de conexão vale na hora, sem reiniciar
o programa.

Se houver só uma VPN cadastrada, ele a usa sozinho e nem é preciso escolher.

### Deixar o ícone sempre visível

O Windows 11 esconde ícones novos atrás da setinha da barra. Para fixar o do
monitor, rode uma vez:

```
powershell -ep bypass -f fixar-icone.ps1
```

Ou arraste o ícone da setinha para a barra, que dá no mesmo.

### Parâmetros (opcionais)

| Parâmetro | O que faz |
|---|---|
| `-ping ENDERECO` | endereço que responde quando a VPN está no ar |
| `-vpn NOME` | conexão a reconectar, se preferir fixar por linha de comando |
| `-intervalo N` | segundos entre verificações (padrão 30) |
| `-check` | verifica uma vez, imprime o resultado e sai |
| `-config CAMINHO` | usa outro arquivo de configuração |

Precedência: o que vem por parâmetro vence a escolha do menu, que vence o
`config.json`.

### Cores do ícone

| Cor | Significado |
|---|---|
| Verde | VPN conectada |
| Vermelho | VPN caiu; o monitor vai reconectar |
| Âmbar | reconexão em andamento |
| Cinza | pausado, ou a verificação não pôde ser feita |

## Configuração

Os parâmetros de linha de comando bastam para o caso normal. O `config.json`
existe para quem quiser ajustar o resto — e o que vier por parâmetro sempre vence
o que estiver no arquivo.

Ele tem duas partes: **como saber** que a VPN caiu (`check`) e **como
reconectar** (`connect`).

### `check` — como detectar

| `kind` | O que faz | Quando usar |
|---|---|---|
| `tcp` | conecta em `host:port` interno | **Preferido.** Prova que a rota existe e o outro lado responde |
| `ping` | ping num host interno | quando o host não expõe porta TCP conhecida |
| `interface` | procura adaptador cujo nome contenha `interfaceMatch` | detecta o túnel mesmo sem tráfego |
| `command` | roda um comando; saída 0 = conectada | quando já existe um script que sabe responder |

### `connect` — como reconectar

| `kind` | O que faz |
|---|---|
| `rasdial` | VPN nativa do Windows |
| `openvpn` | OpenVPN GUI ou CLI |
| `service` | sobe um serviço do Windows (WireGuard, clientes corporativos) |
| `task` | dispara uma tarefa agendada (útil quando exige privilégio elevado) |
| `command` | qualquer executável ou script |

Há um modelo pronto por tipo em `exemplos/`.

## Senha da VPN

A senha **nunca** fica no `config.json`. Ela é gravada cifrada pela DPAPI do
Windows, amarrada ao usuário que a gravou: o arquivo copiado para outra máquina
ou aberto por outro usuário não pode ser decifrado.

```
vpnmonctl.exe -set-password vpn
```

No `config.json`, aponte `connect.secretName` para esse nome. A senha também é
removida das mensagens de log antes de serem gravadas.

## Decisões de projeto

**Duas falhas antes de reconectar.** Uma verificação que falha sozinha costuma
ser oscilação momentânea; derrubar e reerguer a VPN por causa dela causaria mais
interrupção do que resolveria. O limite é configurável em `failuresBeforeReconnect`.

**Erro de verificação não dispara reconexão.** Se o próprio teste não pôde ser
executado (binário ausente, permissão negada), o programa não conclui que a VPN
caiu — reconectar às cegas nesse caso só piora. O ícone fica cinza e o motivo vai
para o log.

**Backoff progressivo.** Tentativas seguidas de reconexão espaçam-se até o teto
de `maxBackoffSeconds`, para não martelar um concentrador que já está fora.
O intervalo volta ao normal assim que a conexão se restabelece.

**Carência após conectar.** Depois de mandar conectar, o programa espera
`graceAfterConnectSeconds` antes de verificar de novo, porque o túnel demora
alguns segundos para subir e uma verificação imediata daria falso negativo.

## Desenvolvimento

```
make test     # testes com cobertura
make lint     # gofmt + go vet (Linux e Windows)
make build    # gera os dois .exe em build/
```

Compila de Linux para Windows sem CGO: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0`.

### Organização

```
cmd/vpnmon/          ponto de entrada e modos de linha de comando
internal/config/     leitura e validação do config.json
internal/vpn/        verificadores (check) e reconectores (connect)
internal/supervisor/ o ciclo verificar → decidir → reconectar
internal/trayui/     ícone, menu e tooltip
internal/winsys/     DPAPI, registro de inicialização, console
internal/logging/    log rotativo
tools/geniconos/     desenha os .ico embutidos no executável
```
