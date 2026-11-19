#!/usr/bin/env bash
# Build da ISO bootável LogiTecno Inventory
# Requer: debootstrap, squashfs-tools, xorriso, mtools, grub-efi-amd64-bin
# Uso: sudo ./build-iso.sh

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
WORK="$SCRIPT_DIR/work"
ROOTFS="$WORK/rootfs"
ISO_DIR="$WORK/iso"
OUTPUT="$PROJECT_ROOT/iso/logitecno-inventory.iso"

UBUNTU_SUITE="noble"
UBUNTU_MIRROR="http://br.archive.ubuntu.com/ubuntu"

echo "=== LogiTecno ISO Builder ==="
echo "Projeto: $PROJECT_ROOT"
echo "Work:    $WORK"

# Limpa build anterior
rm -rf "$WORK"
mkdir -p "$ROOTFS" "$ISO_DIR" "$(dirname "$OUTPUT")"

# 1. Criar rootfs base
echo "[1/6] debootstrap..."
debootstrap --arch=amd64 --variant=minbase \
  --include=systemd-sysv,udev,iproute2,iputils-ping,ca-certificates,pciutils \
  "$UBUNTU_SUITE" "$ROOTFS" "$UBUNTU_MIRROR"

# 2. Copiar o agente
echo "[2/6] copiando agente..."
if [[ ! -f "$PROJECT_ROOT/agent/collect" ]]; then
  echo "ERRO: agent/collect nao encontrado. Rode 'go build -o collect .' em agent/ primeiro."
  exit 1
fi
mkdir -p "$ROOTFS/usr/local/bin"
cp "$PROJECT_ROOT/agent/collect" "$ROOTFS/usr/local/bin/logitecno-collect"
chmod +x "$ROOTFS/usr/local/bin/logitecno-collect"

# 3. Configurar init
echo "[3/6] configurando init..."
cat > "$ROOTFS/etc/systemd/system/logitecno.service" <<'EOF'
[Unit]
Description=LogiTecno Inventory Collector
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/logitecno-collect
StandardOutput=tty
StandardError=tty
TTYPath=/dev/tty1
RemainAfterExit=yes

[Install]
WantedBy=multi-user.target
EOF

ln -sf /etc/systemd/system/logitecno.service \
  "$ROOTFS/etc/systemd/system/multi-user.target.wants/logitecno.service"

# 4. Empacotar squashfs
echo "[4/6] mksquashfs..."
mkdir -p "$ISO_DIR/casper"
mksquashfs "$ROOTFS" "$ISO_DIR/casper/filesystem.squashfs" \
  -comp zstd -noappend

# 5. Montar estrutura da ISO
echo "[5/6] montando estrutura..."
cp "$ROOTFS/boot/vmlinuz"* "$ISO_DIR/casper/vmlinuz"
cp "$ROOTFS/boot/initrd.img"* "$ISO_DIR/casper/initrd"

mkdir -p "$ISO_DIR/boot/grub"
cat > "$ISO_DIR/boot/grub/grub.cfg" <<'EOF'
set timeout=3
set default=0

menuentry "LogiTecno Inventory" {
    linux /casper/vmlinuz boot=casper quiet splash ---
    initrd /casper/initrd
}
EOF

# 6. Gerar ISO
echo "[6/6] xorriso..."
grub-mkstandalone \
  --format=x86_64-efi \
  --output="$WORK/bootx64.efi" \
  --locales="" \
  --fonts="" \
  "boot/grub/grub.cfg=$ISO_DIR/boot/grub/grub.cfg"

mkdir -p "$ISO_DIR/EFI/BOOT"
cp "$WORK/bootx64.efi" "$ISO_DIR/EFI/BOOT/BOOTX64.EFI"

xorriso -as mkisofs \
  -o "$OUTPUT" \
  -isohybrid-mbr /usr/lib/ISOLINUX/isohdpfx.bin \
  -c boot/boot.cat \
  -b boot/grub/i386-pc/eltorito.img \
  -no-emul-boot -boot-load-size 4 -boot-info-table \
  -eltorito-alt-boot \
  -e EFI/BOOT/BOOTX64.EFI \
  -no-emul-boot \
  -isohybrid-gpt-basdat \
  "$ISO_DIR"

echo ""
echo "=== ISO gerada: $OUTPUT ==="
ls -lh "$OUTPUT"