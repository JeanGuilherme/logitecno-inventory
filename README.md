# LogiTecno Inventory

Agente de coleta de inventário de hardware para MSCM.

## Objetivo

Coletar dados de hardware (serial, modelo, CPU, RAM, GPU, rede) de máquinas
usadas, gerar JSON versionado, enviar por webhook e manter fila local quando
não há rede.

## Estrutura

- `agent/` — binário Go de coleta
- `build/` — script de build da ISO bootável
- `scripts/` — utilitários de desenvolvimento
- `test/` — validação em campo

## Fluxo de trabalho

1. Editar código no Windows (VS Code)
2. `git push`
3. No Ubuntu: `git pull && cd agent && go build -o collect .`
4. Testar: `sudo ./collect`

## Status

- [x] Coleta de hardware (DMI, CPU, RAM, GPU, rede)
- [x] Schema JSON versionado
- [x] Detecção de rede com timeout
- [x] Envio por webhook com retry
- [x] Fila local (JSONL append-only)
- [ ] QR Code (aguarda definição de payload com MSCM)
- [ ] Build da ISO (aguarda assets visuais)
- [ ] Importador Windows (aguarda schema validado)

## Pendências de decisão

- Schema JSON precisa ser validado com a MSCM
- Endpoint do webhook precisa ser definido
- Formato do payload do QR precisa ser acordado