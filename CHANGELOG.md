# Changelog

Todas as mudanças notáveis deste projeto são documentadas neste arquivo.

O formato segue [Keep a Changelog](https://keepachangelog.com/pt-BR/1.0.0/), e este projeto adere ao [Semantic Versioning](https://semver.org/lang/pt-BR/). Veja [docs/VERSIONING.md](docs/VERSIONING.md) para o guia completo de versionamento adotado pelo Tocli.

## [Unreleased]

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
