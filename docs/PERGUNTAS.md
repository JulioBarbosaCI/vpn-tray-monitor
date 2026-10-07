# Perguntas ao Guibson — situação

## Respondido

| Pergunta | Resposta (02/09/2026) | Como entrou |
|---|---|---|
| Qual VPN? | Nativa do Windows | `rasdial` |
| Credenciais? | Já salvas no cliente | sem usuário/senha; o `rasdial` usa as guardadas |
| Como detectar queda? | Ping em `10.254.1.172` | `-ping 10.254.1.172` |
| Tem firewall no caminho? | Não | ping é confiável aqui; não precisa de alternativa TCP |
| Nome da VPN? | Vem por parâmetro | `-vpn "Nome da conexao"` |

Uso final, sem arquivo de configuração:

```
vpnmon.exe -ping 10.254.1.172 -vpn "VPN Empresa"
```

## Ainda em aberto (não bloqueiam)

1. **Precisa funcionar sem ninguém logado?** Hoje roda no login do usuário.
   Sem sessão exige virar serviço do Windows — mudança pequena, mas melhor confirmar.
2. **Em quantas máquinas?** Muda se compensa empacotar um instalador.
3. **Precisa avisar alguém quando cair?** Hoje só grava log local e mostra balão.

## A levantar antes de implantar

**Antivírus corporativo.** Um `.exe` novo sem assinatura digital pode ser barrado.
Resolve-se assinando com certificado de code signing ou liberando o hash na
política do antivírus.

**Teste com a VPN real.** A VM de testes tem saída para a internet, então dá para
cadastrar a VPN de verdade nela e testar de ponta a ponta — basta o endereço do
servidor e o tipo de túnel (SSTP, L2TP, IKEv2, PPTP).
