# Prioridade e categoria das tarefas

O Google Tasks não tem campo de prioridade. O Tocli infere três níveis a partir do título da tarefa e do nome da lista, e usa esse nível para ordenar a lista e destacar as tarefas.

## Prioridade

| Nível | Como definir |
|-------|--------------|
| Urgente | Título começando com `[U]` (ou `🔴`), ou lista com nome contendo `urgente`, `urgent`, `asap`, `critical`, `crítico`, `alta prioridade`, `high priority` ou `firefight`. |
| Importante | Título começando com `[I]` (ou `⭐`, `★`), ou lista com nome contendo `important`, `importante`, `star`, `favorit`, `priority`, `prioridade`, `focus` ou `foco`. |
| Normal | Todo o restante. |

O prefixo do título tem precedência sobre o nome da lista. As tarefas abertas são ordenadas por prioridade e, dentro de cada nível, por prazo.

## Categoria

Para tarefas de prioridade normal, um marcador de uma letra e uma cor indicam a categoria da lista. A categoria é independente da prioridade e usa palavras-chave próprias.

| Marcador | Categoria | Nomes de lista reconhecidos (exemplos) |
|----------|-----------|----------------------------------------|
| `W` | Trabalho | `work`, `trabalho`, `office` |
| `J` | Emprego | `job`, `jobs`, `emprego` |
| `P` | Pessoal | `personal`, `pessoal`, `home`, `casa` |
| `L` | Aprendizado | `learning`, `study`, `estudos`, `education`, `curso` |
| `·` | Padrão | qualquer outro nome |

Tarefas urgentes aparecem com o marcador `U` e as importantes com `I`, em vermelho e amarelo.
