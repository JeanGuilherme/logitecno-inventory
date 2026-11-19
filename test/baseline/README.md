# Baseline de campo

Um arquivo `.txt` por máquina testada, nomeado como
`<fabricante>-<modelo>-<data>.txt`.

## O que registrar em cada máquina

- Serial retornou valor real, vazio, ou "To be filled by O.E.M."?
- Modelo e fabricante retornaram?
- Bootou sem desativar Secure Boot?
- Alguma tela de erro?

## Como coletar

No Alpine ou Ubuntu, com o pendrive bootado:

    sudo ./scripts/dev-collect.sh > baseline/lenovo-thinkcentre-2026-11-19.txt

## Resumo até agora

| Máquina | Bootou | Serial OK | Secure Boot |
|---|---|---|---|
| Lenovo | sim | sim | desativado |
| HP | não | — | bloqueou (0x1A) |