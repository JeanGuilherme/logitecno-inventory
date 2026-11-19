#!/usr/bin/env bash
#
# Prepara o pendrive com 2 particoes:
#   1. ISO (read-only, bootavel)
#   2. DADOS (FAT32, gravavel — onde a fila e salva)
#
# Uso: sudo ./prepare-usb.sh /dev/sdX
#
# ATENCAO: APAGA TUDO no dispositivo informado.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

DEVICE="${1:-}"
ISO="${2:-$PROJECT_ROOT/iso/logitecno-inventory.iso}"

if [[ -z "$DEVICE" ]]; then
  echo "Uso: sudo $0 /dev/sdX [caminho/da/iso]"
  echo ""
  echo "Discos disponiveis:"
  lsblk -d -o NAME,SIZE,MODEL,TRAN | grep -v loop
  exit 1
fi

[[ -b "$DEVICE" ]] || { echo "ERRO: $DEVICE nao e um dispositivo de bloco"; exit 1; }
[[ -f "$ISO" ]] || { echo "ERRO: ISO nao encontrada: $ISO"; exit 1; }

echo ""
echo "============================================================"
echo "  PREPARAR PENDRIVE"
echo "============================================================"
echo "  Dispositivo: $DEVICE"
echo "  ISO:         $ISO"
echo ""
lsblk "$DEVICE"
echo ""
echo "  ATENCAO: TODOS OS DADOS EM $DEVICE SERAO APAGADOS."
echo ""
read -p "  Digite SIM (maiusculo) para continuar: " CONFIRM
[[ "$CONFIRM" == "SIM" ]] || { echo "Cancelado."; exit 1; }

# Desmonta tudo
for p in "${DEVICE}"?*; do
  umount "$p" 2>/dev/null || true
done

echo ""
echo "[1/5] Limpando tabela de particoes..."
wipefs -a "$DEVICE"

echo "[2/5] Criando tabela GPT com 2 particoes..."
# Particao 1: 2GB para a ISO (isohybrid)
# Particao 2: resto para dados
parted -s "$DEVICE" mklabel gpt
parted -s "$DEVICE" mkpart ISO fat32 1MiB 2GiB
parted -s "$DEVICE" set 1 boot on
parted -s "$DEVICE" mkpart DADOS fat32 2GiB 100%

sleep 1
partprobe "$DEVICE"
sleep 1

PART_ISO="${DEVICE}1"
PART_DADOS="${DEVICE}2"

echo "[3/5] Gravando ISO na particao 1..."
dd if="$ISO" of="$PART_ISO" bs=4M status=progress oflag=sync conv=fsync

echo "[4/5] Formatando particao de dados (FAT32)..."
mkfs.vfat -F 32 -n LOGITECNO "$PART_DADOS"

echo "[5/5] Criando estrutura na particao de dados..."
TMP=$(mktemp -d)
mount "$PART_DADOS" "$TMP"
mkdir -p "$TMP/coletas"
cat > "$TMP/LEIA-ME.txt" <<'EOF'
LogiTecno Inventory — Particao de Dados

Os arquivos de coleta ficam em:
  coletas/queue.jsonl

Cada linha do arquivo e uma maquina coletada (formato JSON).

Para processar no Windows:
  1. Copie queue.jsonl para o computador
  2. Abra no Excel, Notepad++ ou importe no sistema da MSCM

Se o arquivo nao existir, nenhuma maquina foi coletada ainda
ou todas as coletas foram enviadas com sucesso pela rede.
EOF
umount "$TMP"
rmdir "$TMP"

sync

echo ""
echo "============================================================"
echo "  PENDRIVE PRONTO"
echo "============================================================"
echo ""
lsblk "$DEVICE"
echo ""
echo "  Boot:   $PART_ISO (ISO)"
echo "  Dados:  $PART_DADOS (FAT32, label LOGITECNO)"
echo ""
echo "  No Windows, a particao de dados aparece como unidade."
echo "  Os arquivos ficam em: coletas/queue.jsonl"
echo ""