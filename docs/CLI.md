# Linha de comando

## Flags

| Flag | Descrição |
|------|-----------|
| `-offline` | Usa repositórios de exemplo em memória e não chama as APIs do Google. |
| `-sync` | Valida o acesso ao Google Tasks e ao Google Calendar e encerra, sem abrir a interface. Não pode ser combinada com `-offline`. |
| `-theme auto\|dark\|light` | Tema de cores. O padrão `auto` segue o fundo do terminal. |
| `-version` | Exibe a versão e o commit de origem do binário, e se existe uma release mais nova. |
| `-update` | Compila a release mais recente e substitui o executável em uso. |

Na inicialização o Tocli verifica a conexão com o Google e faz a autenticação. Se isso falhar, ele continua com dados de exemplo e avisa. Com `-sync`, a falha encerra o programa com erro.

## Variáveis de ambiente

| Variável | Efeito |
|----------|--------|
| `TOC_GOOGLE_CREDENTIALS` | Caminho do arquivo de credenciais OAuth, usado quando elas não foram embutidas no binário. Padrão: `~/.config/tocli/credentials.json`. |
| `TOC_RATINGS_PATH` | Caminho do arquivo das avaliações diárias. Padrão: `~/.config/tocli/ratings.json`. |
| `XDG_CONFIG_HOME` | Substitui `~/.config` como base dos arquivos acima e do token (`token.json`). |

## Versão

```text
$ tocli -version
tocli v2.0.1 (commit 1a2b3c4)
You are on the latest version.
```

O commit permite conferir de que código o binário foi compilado. O sufixo `+dirty` indica que a compilação foi feita com alterações não commitadas. Uma versão local mais nova que a última release (build de desenvolvimento) é identificada como tal, em vez de aparecer como "atualização disponível".

## Atualização

`tocli -update` instala a release mais recente publicada no GitHub. Requer `git` e `go` no `PATH`, porque o binário é compilado a partir do código-fonte.

O processo:

1. Consulta a última release e encerra se a versão atual já é igual ou mais nova.
2. Clona **somente a tag da release** em um diretório temporário. Sua cópia local do código, a branch atual e arquivos não commitados não são tocados, e o comando funciona de qualquer diretório.
3. Compila ali, reaproveitando as opções de link do executável atual. Assim as credenciais do Google embutidas com `-ldflags` continuam no binário novo.
4. **Verifica o binário novo** lendo as informações embutidas nele: o commit precisa ser o da tag, a árvore de origem precisa estar limpa e a versão precisa estar gravada.
5. Só então substitui o executável em uso, mantendo as permissões. Se qualquer etapa falhar, o executável atual permanece como estava.

Não há alternativa para a branch `main`: se a tag não puder ser obtida, a atualização falha com uma mensagem, em vez de compilar outro código com o número da versão nova. Se o executável estiver em um diretório protegido, o erro sugere executar com `sudo` ou mover o binário. Executáveis temporários do `go run` são recusados, porque não faria sentido substituí-los.
