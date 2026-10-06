# Changelog

Todas as mudanças notáveis deste projeto são documentadas neste arquivo.

O formato segue [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/), e este projeto adere ao [Semantic Versioning](https://semver.org/lang/pt-BR/). Veja [docs/VERSIONING.md](docs/VERSIONING.md) para o guia completo de versionamento adotado pelo Tocli.

## [Unreleased]

## [1.2.0] - 2026-10-06

Adiciona busca de eventos, paleta de comandos, edição de tarefas, navegação entre anos e detalhes do evento. A interface foi migrada para o Charm v2, com layout em cartões, tema claro e suporte a mouse. O `-update` foi corrigido: ele atualizava apenas o número da versão, sem trazer o código novo.

### Added

- **Busca de eventos** (`/` ou clique na caixa acima da agenda): a barra vai até o centro da tela e os resultados aparecem abaixo. Cobre de 12 meses atrás a 12 meses à frente, ignora maiúsculas e acentos, exige todos os termos e procura em título, local e descrição. `Enter` leva ao dia do evento (mudando o ano do gráfico, se preciso) e `Tab` abre os detalhes.
- **Paleta de comandos** (`:` ou `>` na busca): executa ações pelo nome (nova tarefa, editar, concluir, excluir, ir para hoje, trocar de ano, alternar o gráfico, foco dos painéis, tema, atualizar, ajuda, sair). Comandos que não se aplicam ao momento ficam ocultos.
- **Edição de tarefas** (`e` no painel de tarefas): altera título e prazo; prazo vazio remove o prazo. A lista é reordenada e o cursor permanece na tarefa editada.
- **Navegação entre anos** no gráfico (`[` e `]`): o ano aparece na hora e os dados são buscados quando a navegação para, descartando respostas atrasadas.
- **Detalhes do evento** (`Enter` na agenda, clique no evento, `Tab` na busca ou pela paleta): data, horário, duração, local, descrição com rolagem e links numerados. `o` e `1`-`9` abrem os links no navegador, incluindo a videochamada do Google Meet.
- **Mouse**: clique em painéis, tarefas, dias do gráfico e eventos; roda para rolar o painel sob o ponteiro.
- **Tema claro** e a flag `-theme auto|dark|light`. O modo `auto` acompanha o fundo do terminal, e o tema também pode ser trocado em execução.
- `-version` agora mostra o commit de origem do binário.
- Abertura de links com suporte ao WSL (`wslview` e `explorer.exe`), usada nos detalhes do evento e no login do Google.
- Testes automatizados para a interface, os casos de uso, o domínio, o adapter do Google, a abertura de links e o `-update`.
- Documentação em `docs/`: `USAGE.md`, `CLI.md`, `PRIORITY.md` e `ARCHITECTURE.md`.

### Changed

- Interface migrada para Bubble Tea, Lip Gloss e Bubbles v2. **Requer Go 1.24.2 ou superior.**
- Layout calculado por uma função pura e testada: agenda, gráfico e progresso são cartões independentes, e os diálogos (formulários, ajuda, detalhes) são desenhados sobre o dashboard em vez de ocupar a tela inteira.
- Estado de interação centralizado em um único modo, no lugar de vários booleanos. Com a ajuda aberta, apenas `?`, `Esc` e `q` agem.
- Barra de atalhos baseada em `bubbles/help`, que reduz os atalhos por prioridade conforme a largura e mantém sempre ajuda e sair.
- Dias sem atividade no gráfico aparecem como um ponto discreto, e o progresso do ano usa uma barra com gradiente.
- A agenda rola acompanhando a seleção e indica quantos eventos estão visíveis.
- `domain.TaskRepository` ganhou o método `UpdateTask`; implementações próprias do repositório precisam adicioná-lo.
- O modo `-offline` ganhou eventos de exemplo passados e futuros, com links e descrição.
- README reescrito de forma mais objetiva, com o conteúdo detalhado movido para `docs/` e a captura de tela em `docs/PREVIEW.md`.

### Fixed

- **`-update` mudava apenas o número da versão.** O binário era carimbado com a tag mais recente mesmo quando o `git` não tinha trazido o código dela (por exemplo, após cair no fallback para `main`), e o resultado era gravado no diretório atual em vez de substituir o executável em uso. Agora a release é clonada em um diretório temporário, compilada, verificada pelo commit embutido e só então instalada, sem tocar na sua cópia do código. As credenciais do Google embutidas no binário são preservadas.
- Prazos do Google Tasks apareciam como "21:00 do dia anterior" e a criação com horário podia mover a data. O Google guarda apenas a data; ela agora é lida e gravada sem conversão de fuso.
- O login do Google não abria o navegador no WSL.
- Alturas de terminal entre 11 e 15 linhas no layout empilhado reservavam mais linhas do que existiam.
- Avisos do `go vet` em `internal/adapter/google/auth.go`. `go vet ./...` agora passa.
- `go.sum` passou a ser versionado, o que é necessário para compilar a partir de um clone limpo (e para o `-update`).

**Pull Requests**
- TUI-05: Add search, command palette, task editing, year navigation and event details by @TETEURYAN in https://github.com/TETEURYAN/tocli/pull/7

**Full Changelog**: https://github.com/TETEURYAN/tocli/compare/v1.1.0...v1.2.0

## [1.1.0] - 2026-07-09

Adiciona um segundo modo ao painel de gráfico: avaliação diária de humor/produtividade, com nota de texto livre e exportação em CSV.

### Added

- **Daily Rating Graph**: segundo modo do painel de gráfico (tecla `g` alterna entre *contribution graph* e *daily rating*).
- Nota de 1 a 5 por dia (teclas `1`-`5`), com cor interpolada de vermelho a verde.
- Texto livre por dia (tecla `t`) descrevendo como foi a jornada, editável em uma tela dedicada (`ctrl+s` salva, `esc` cancela).
- Exportação mensal em CSV (tecla `e`), com uma linha por dia (`data,nota,texto`).
- Persistência local das notas e textos em `~/.config/tocli/ratings.json`, independente da conta Google.

### Changed

- Cores do gradiente do Daily Rating tornadas mais saturadas (`#ef4444` → `#22c55e`) para melhor leitura, no lugar do vermelho/verde pastel reaproveitado do tema.

**Pull Requests**
- TUI-04: Add Daily Rating Graph mode to Graph Pane, CSV export and day notes by @TETEURYAN in https://github.com/TETEURYAN/tocli/pull/6

**Full Changelog**: https://github.com/TETEURYAN/tocli/compare/v1.0.0...v1.1.0

## [1.0.0] - 2026-04-06

🚀 Primeira versão pública do **Tocli**, um painel de produtividade no terminal que integra tarefas, agenda e métricas em uma interface moderna e totalmente orientada a teclado.

### Added

- **Lista de tarefas**: tarefas abertas e concluídas de hoje; criar, concluir, reabrir e excluir; sincroniza com Google Tasks.
- **Agenda do dia**: eventos de hoje com horário, título e local; destaque para o evento em andamento e esmaecimento dos passados.
- **Detalhe por dia**: navegar pelo contribution graph mostra os eventos e tarefas concluídas daquele dia na agenda.
- **Contribution graph**: grade anual de tarefas concluídas por dia, com intensidade de cor proporcional ao volume — estilo GitHub.
- **Progresso do ano**: percentual do ano decorrido, dia atual e dias restantes.
- **Sistema de prioridade**: três níveis (Urgente / Importante / Normal), inferido automaticamente pelo nome da lista ou por prefixo no título da tarefa.

**Pull Requests**
- TUI-02: Adds feature to delete and create task with date by @TETEURYAN in https://github.com/TETEURYAN/tocli/pull/3
- TUI-03: Add update feature by @TETEURYAN in https://github.com/TETEURYAN/tocli/pull/4
- TUI-02: Order tasks by @TETEURYAN in https://github.com/TETEURYAN/tocli/pull/5

**Full Changelog**: https://github.com/TETEURYAN/tocli/compare/v0.2.0-alpha...v1.0.0
