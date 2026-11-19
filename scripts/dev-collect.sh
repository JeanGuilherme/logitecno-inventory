#!/usr/bin/env bash
# Coleta manual de hardware para validação em campo.
# Roda no Alpine ou Ubuntu, sem precisar compilar nada.
# Uso: sudo ./dev-collect.sh

set -uo pipefail

echo "=== LogiTecno — Coleta Manual ==="
echo "Data: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Hostname: $(hostname)"
echo ""

echo "--- DMI ---"
for f in sys_vendor product_name product_serial board_serial; do
  val=$(cat "/sys/class/dmi/id/$f" 2>/dev/null || echo "N/A")
  echo "$f: $val"
done
echo ""

echo "--- CPU ---"
grep "model name" /proc/cpuinfo 2>/dev/null | head -1 || echo "N/A"
echo ""

echo "--- RAM ---"
free -h 2>/dev/null || echo "N/A"
echo ""

echo "--- GPU ---"
lspci 2>/dev/null | grep -i vga || echo "N/A"
echo ""

echo "--- REDE ---"
ip link show 2>/dev/null || echo "N/A"
echo ""

echo "--- DISCO ---"
lsblk 2>/dev/null || echo "N/A"