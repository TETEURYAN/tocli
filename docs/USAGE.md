# Guia de uso

Atalhos de teclado e mouse, e o funcionamento de cada área do Tocli. Pressione `?` dentro do aplicativo para ver a ajuda resumida.

## Navegação geral

A tela tem três painéis focáveis: **tarefas**, **agenda** e **gráfico**. O foco muda com `Tab` e `Shift+Tab`, e o painel ativo aparece na barra de status.

| Tecla | Ação |
|-------|------|
| `Tab` / `Shift+Tab` | Próximo painel / painel anterior |
| `↑` `↓` ou `k` `j` | Mover a seleção no painel |
| `/` | Abrir a busca de eventos |
| `:` | Abrir a paleta de comandos |
| `r` | Atualizar tarefas, eventos e gráfico |
| `?` | Mostrar ou ocultar a ajuda |
| `q` / `Ctrl+C` | Sair |

Abaixo de 68 colunas os painéis são empilhados na vertical.

## Tarefas

| Tecla | Ação |
|-------|------|
| `Enter` ou `Espaço` | Concluir ou reabrir a tarefa selecionada |
| `n` | Criar tarefa |
| `e` | Editar o título e o prazo da tarefa selecionada |
| `d` | Excluir a tarefa selecionada (pede confirmação com `y`; `n` ou `Esc` cancela) |

### Criar e editar

O mesmo formulário serve para criar (`n`) e editar (`e`). Ao editar, ele abre preenchido e a lista da tarefa não pode ser alterada.

| Tecla | Ação |
|-------|------|
| `Tab` | Alternar entre os campos título e prazo |
| `[` `]` | Lista de destino anterior / seguinte (apenas ao criar) |
| `Enter` | Salvar |
| `Esc` | Cancelar |

O prazo aceita `DD-MM-AAAA` ou `DD-MM-AAAA HH:MM`, no fuso local. Ao editar, deixar o campo vazio remove o prazo. O Google Tasks guarda apenas a **data** do prazo: o horário informado é descartado na sincronização.

Depois de salvar uma edição a lista é reordenada (prioridade e prazo), e o cursor permanece na tarefa editada.

## Agenda

A agenda mostra os eventos de hoje. Com o cursor sobre um dia no gráfico, ela passa a mostrar os eventos e as tarefas concluídas daquele dia.

| Tecla | Ação |
|-------|------|
| `↑` `↓` ou `k` `j` | Selecionar evento |
| `Enter` | Abrir os detalhes do evento selecionado |

Quando há mais eventos do que linhas disponíveis, a lista acompanha a seleção e o subtítulo indica o intervalo exibido (por exemplo, `4–5 of 5`).

### Detalhes do evento

Mostra título, data, horário e duração, local, descrição e os links do evento. Os links vêm da videochamada, de endereços escritos no local ou na descrição e da página do evento no Google Calendar.

| Tecla | Ação |
|-------|------|
| `o` | Abrir o primeiro link (a videochamada, quando existe) |
| `1`-`9` | Abrir o link de mesmo número |
| `↑` `↓`, `PgUp` `PgDn` | Rolar a descrição, quando ela não cabe |
| `Esc` ou `Enter` | Fechar |

Apenas endereços `http` e `https` são abertos. No WSL o link é aberto no navegador do Windows.

## Gráfico

Dois modos, alternados com `g`: **contribution graph** (tarefas concluídas por dia) e **avaliação diária** (nota de 1 a 5).

| Tecla | Ação |
|-------|------|
| `←` `→` ou `h` `l` | Semana anterior / seguinte |
| `↑` `↓` ou `k` `j` | Dia anterior / seguinte |
| `[` `]` | Ano anterior / seguinte |
| `g` | Alternar o modo do gráfico |
| `1`-`5` | (avaliação diária) Dar nota ao dia selecionado |
| `t` | (avaliação diária) Escrever ou editar a anotação do dia |
| `e` | (avaliação diária) Exportar o mês exibido para CSV |

**Anos.** O gráfico exibe o ano pedido imediatamente, com "loading", e busca os dados quando você para de apertar as teclas. O cursor mantém o mesmo mês e dia. O intervalo vai de 2000 a cinco anos à frente do atual.

**Avaliação diária.** A nota vai de 1 (vermelho) a 5 (verde), é manual e fica apenas no seu computador, em `~/.config/tocli/ratings.json` (caminho configurável com `TOC_RATINGS_PATH`). Dias sem nota aparecem como um ponto discreto. A anotação do dia (`t`) abre um editor de várias linhas: `Ctrl+S` salva e `Esc` cancela.

**Exportação.** `e` grava `tocli-ratings-AAAA-MM.csv` no diretório atual, com uma linha por dia (`data,nota,texto`).

## Busca de eventos

`/` (ou um clique na caixa acima da agenda) abre a busca: a barra vai da posição original até o centro da tela e os resultados aparecem logo abaixo, enquanto você digita.

- Cobre os eventos do calendário de 12 meses atrás a 12 meses à frente.
- Procura no título, no local e na descrição, ignorando maiúsculas e acentos (`reuniao` encontra "Reunião").
- Todos os termos digitados precisam aparecer, em qualquer ordem. Título pesa mais que local, e local mais que descrição. Em empate, vem primeiro o evento mais próximo de hoje.

| Tecla | Ação |
|-------|------|
| `↑` `↓` ou `Ctrl+P` `Ctrl+N` | Selecionar resultado (`PgUp` `PgDn` pulam uma página) |
| `Enter` | Ir ao dia do evento: o gráfico muda para o ano e a data, e a agenda mostra o dia |
| `Tab` | Abrir os detalhes do evento |
| `Esc` | Fechar |

Em terminais com menos de 12 linhas de corpo a caixa de busca é omitida, mas o atalho `/` continua funcionando.

## Paleta de comandos

`:` abre a paleta direto no modo de comandos. Também é possível digitar `>` no início da busca. Os comandos são filtrados pelo que você digita (sem diferenciar maiúsculas e acentos), e `Enter` executa o selecionado. Comandos que não se aplicam ao momento ficam ocultos.

| Comando | Equivalente |
|---------|-------------|
| New task | `n` |
| Edit selected task | `e` |
| Complete or reopen selected task | `Espaço` |
| Delete selected task | `d` (pede confirmação) |
| Show details of the selected event | `Enter` na agenda |
| Go to today | |
| Graph: previous year / next year | `[` / `]` |
| Graph: switch between contribution and daily rating | `g` |
| Write note for the selected day | `t` (avaliação diária) |
| Export this month's ratings to CSV | `e` (avaliação diária) |
| Focus tasks / agenda / graph | |
| Theme: dark / light / follow the terminal | |
| Refresh data | `r` |
| Show keyboard shortcuts | `?` |
| Quit | `q` |

## Mouse

| Ação | Efeito |
|------|--------|
| Clique em um painel | Dá foco ao painel |
| Clique em uma tarefa | Seleciona a tarefa |
| Clique em um dia do gráfico | Seleciona o dia e mostra seus eventos na agenda |
| Clique em um evento da agenda | Abre os detalhes do evento |
| Clique na caixa de busca | Abre a busca |
| Clique em um resultado da busca | Abre o resultado (vai ao dia, ou executa o comando) |
| Clique fora da busca | Fecha a busca |
| Roda do mouse | Rola o painel sob o ponteiro (no gráfico, semana a semana) |

Com um diálogo aberto (ajuda, formulários, detalhes), os cliques são ignorados.

## Temas

`-theme auto` (padrão) acompanha o fundo do terminal e troca sozinho se ele mudar. `dark` e `light` fixam a paleta. A paleta também pode ser trocada dentro do aplicativo pelos comandos `Theme: ...`.
