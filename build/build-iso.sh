#!/usr/bin/env bash
#
# LogiTecno Inventory — Build de ISO bootavel
#
# Requer Ubuntu 24.04 (noble) como host de build.
# Uso: sudo ./build-iso.sh
#
# O script para em qualquer erro e diz exatamente o que falhou.
# Nao gera ISO quebrada silenciosamente.

set -euo pipefail

# ---------------------------------------------------------------- config
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
WORK="$SCRIPT_DIR/work"
ROOTFS="$WORK/rootfs"
ISO_DIR="$WORK/iso"
OUTPUT_DIR="$PROJECT_ROOT/iso"
OUTPUT="$OUTPUT_DIR/logitecno-inventory.iso"

UBUNTU_SUITE="noble"
UBUNTU_MIRROR="http://br.archive.ubuntu.com/ubuntu"

AGENT_BIN="$PROJECT_ROOT/agent/collect"
WEBHOOK_URL="${LOGITECNO_WEBHOOK_URL:-}"

# Fila volatil — sem particao de dados no pendrive.
# O agente grava em /tmp e o QR Code + JSON na tela sao a saida real.
QUEUE_PATH="/tmp/logitecno/queue.jsonl"

# ---------------------------------------------------------------- helpers
log()  { printf '\n\033[1;36m[%s]\033[0m %s\n' "$1" "$2"; }
ok()   { printf '  \033[1;32mOK\033[0m %s\n' "$1"; }
warn() { printf '  \033[1;33mAVISO\033[0m %s\n' "$1"; }
die()  { printf '\n\033[1;31mERRO:\033[0m %s\n' "$1" >&2; exit 1; }

require_root() {
  [[ $EUID -eq 0 ]] || die "execute com sudo: sudo ./build-iso.sh"
}

require_cmd() {
  command -v "$1" >/dev/null 2>&1 || die "comando '$1' nao encontrado. Instale: $2"
}

# ---------------------------------------------------------------- preflight
preflight() {
  log "1/9" "Verificando pre-requisitos"

  require_root

  require_cmd debootstrap   "apt install debootstrap"
  require_cmd mksquashfs    "apt install squashfs-tools"
  require_cmd xorriso       "apt install xorriso"
  require_cmd grub-mkstandalone "apt install grub-efi-amd64-bin grub-pc-bin"
  require_cmd mcopy         "apt install mtools"

  [[ -f "$AGENT_BIN" ]] || die "agente nao compilado. Rode: cd agent && go build -o collect ."
  ok "agente encontrado: $AGENT_BIN"

  if [[ -z "$WEBHOOK_URL" ]]; then
    warn "LOGITECNO_WEBHOOK_URL nao definida — a ISO vai rodar em modo offline (QR Code)"
  else
    ok "webhook configurado"
  fi

  # isohdpfx.bin — local varia por distro
  ISOHDPFX=""
  for p in /usr/lib/ISOLINUX/isohdpfx.bin \
           /usr/lib/syslinux/mbr/isohdpfx.bin \
           /usr/share/syslinux/isohdpfx.bin; do
    [[ -f "$p" ]] && ISOHDPFX="$p" && break
  done
  [[ -n "$ISOHDPFX" ]] || die "isohdpfx.bin nao encontrado. Instale: apt install isolinux syslinux-utils"
  ok "isohdpfx: $ISOHDPFX"
}
  # Monta /proc /sys /dev para o update-initramfs funcionar
  mount -t proc  proc  "$ROOTFS/proc"
  mount -t sysfs sysfs "$ROOTFS/sys"
  mount --bind /dev   "$ROOTFS/dev"
  mount --bind /dev/pts "$ROOTFS/dev/pts"

  chroot "$ROOTFS" update-initramfs -c -k all || \
    die "falha ao gerar initrd"

  # Desmonta na ordem inversa
  umount "$ROOTFS/dev/pts"
  umount "$ROOTFS/dev"
  umount "$ROOTFS/sys"
  umount "$ROOTFS/proc"

# ---------------------------------------------------------------- rootfs
build_rootfs() {
  log "2/9" "Criando rootfs base (debootstrap)"

  rm -rf "$WORK"
  mkdir -p "$ROOTFS" "$ISO_DIR" "$OUTPUT_DIR"

  debootstrap \
    --arch=amd64 \
    --variant=minbase \
    --components=main,universe \
    --include=systemd-sysv,udev,iproute2,iputils-ping,ca-certificates,pciutils,util-linux,dmidecode,kbd,console-setup,linux-image-generic,initramfs-tools,beep,casper,lupin-casper"$UBUNTU_SUITE" "$ROOTFS" "$UBUNTU_MIRROR" \
    chroot "$ROOTFS" update-initramfs -u -k all
  # Garante que o initrd existe
  if ! ls "$ROOTFS"/boot/initrd.img-* >/dev/null 2>&1; then
    chroot "$ROOTFS" update-initramfs -c -k all || \
      die "falha ao gerar initrd"
  fi

  ok "rootfs criado: $(du -sh "$ROOTFS" | cut -f1)"
}

# ---------------------------------------------------------------- agent
install_agent() {
  log "3/9" "Instalando agente e configuracao"

  install -Dm755 "$AGENT_BIN" "$ROOTFS/usr/local/bin/logitecno-collect"

  # Diretorio da fila volatil
  mkdir -p "$ROOTFS$(dirname "$QUEUE_PATH")"

  # Variaveis de ambiente do agente
  {
    echo "# LogiTecno Inventory — configuracao do agente"
    [[ -n "$WEBHOOK_URL" ]] && echo "LOGITECNO_WEBHOOK_URL=$WEBHOOK_URL"
    echo "LOGITECNO_QUEUE_PATH=$QUEUE_PATH"
  } > "$ROOTFS/etc/environment"

  ok "agente instalado"
}

# ---------------------------------------------------------------- systemd
configure_systemd() {
  log "4/9" "Configurando servico e console"

  # Servico do agente — roda no tty1, espera rede, desliga ao final.
  # SEM dependencia de particao de dados: o agente usa /tmp e QR Code.
  cat > "$ROOTFS/etc/systemd/system/logitecno.service" <<'EOF'
[Unit]
Description=LogiTecno Inventory Collector
After=network-online.target
Wants=network-online.target
Conflicts=getty@tty1.service
Before=getty@tty1.service

[Service]
Type=oneshot
ExecStart=/usr/local/bin/logitecno-collect
StandardInput=tty
StandardOutput=tty
StandardError=tty
TTYPath=/dev/tty1
TTYReset=yes
TTYVHangup=yes
TTYVTDisallocate=yes
RemainAfterExit=yes
TimeoutStartSec=300

[Install]
WantedBy=multi-user.target
EOF

  mkdir -p "$ROOTFS/etc/systemd/system/multi-user.target.wants"
  ln -sf /etc/systemd/system/logitecno.service \
        "$ROOTFS/etc/systemd/system/multi-user.target.wants/logitecno.service"

  # Desabilita getty no tty1 (o agente usa)
  rm -f "$ROOTFS/etc/systemd/system/getty.target.wants/getty@tty1.service" 2>/dev/null || true

  # Hostname padrao
  echo "logitecno-inventory" > "$ROOTFS/etc/hostname"

  # fstab minimo
  cat > "$ROOTFS/etc/fstab" <<'EOF'
proc  /proc  proc  defaults  0 0
sysfs /sys   sysfs defaults  0 0
tmpfs /tmp   tmpfs defaults  0 0
EOF

  ok "systemd configurado (sem particao de dados)"
}

# ---------------------------------------------------------------- live boot
configure_live() {
  log "5/9" "Configurando ambiente live (casper)"

  # casper.conf precisa ser ARQUIVO, nao diretorio
  cat > "$ROOTFS/etc/casper.conf" <<'EOF'
export USERNAME="logitecno"
export USERFULLNAME="LogiTecno Inventory"
export HOST="logitecno-inventory"
export BUILD_SYSTEM="Ubuntu"
EOF

  # Desabilita servicos que nao fazem sentido em live
  for svc in apt-daily.timer apt-daily-upgrade.timer \
             systemd-timesyncd.service e2scrub_all.timer; do
    rm -f "$ROOTFS/etc/systemd/system/"*".wants/$svc" 2>/dev/null || true
  done

  ok "casper configurado"
}

# ---------------------------------------------------------------- squashfs
pack_squashfs() {
  log "6/9" "Empacotando squashfs"

  mkdir -p "$ISO_DIR/casper"
  mksquashfs "$ROOTFS" "$ISO_DIR/casper/filesystem.squashfs" \
    -comp zstd -Xcompression-level 19 -noappend -no-progress

  # Tamanho do filesystem (casper usa para barra de progresso)
  du -sx --block-size=1 "$ROOTFS" | cut -f1 \
    > "$ISO_DIR/casper/filesystem.size"

  ok "squashfs: $(du -sh "$ISO_DIR/casper/filesystem.squashfs" | cut -f1)"
}

# ---------------------------------------------------------------- kernel
copy_kernel() {
  log "7/9" "Copiando kernel e initrd"

  local vmlinuz initrd
  vmlinuz=$(ls "$ROOTFS"/boot/vmlinuz-* 2>/dev/null | head -1) \
    || die "vmlinuz nao encontrado em $ROOTFS/boot"
  initrd=$(ls "$ROOTFS"/boot/initrd.img-* 2>/dev/null | head -1) \
    || die "initrd nao encontrado em $ROOTFS/boot"

  cp "$vmlinuz" "$ISO_DIR/casper/vmlinuz"
  cp "$initrd"  "$ISO_DIR/casper/initrd"

  ok "kernel: $(basename "$vmlinuz")"
  ok "initrd: $(basename "$initrd")"
}

# ---------------------------------------------------------------- grub
build_grub() {
  log "8/9" "Gerando bootloader GRUB (UEFI + Legacy)"

  mkdir -p "$ISO_DIR/boot/grub"

  # Menu do GRUB — sem quiet/splash para o operador ver o que acontece
  cat > "$ISO_DIR/boot/grub/grub.cfg" <<'EOF'
set timeout=3
set default=0

insmod all_video
insmod gfxterm
terminal_output gfxterm

menuentry "LogiTecno Inventory" {
    linux /casper/vmlinuz boot=casper console=tty1 loglevel=4 ---
    initrd /casper/initrd
}

menuentry "LogiTecno Inventory (diagnostico)" {
    linux /casper/vmlinuz boot=casper verbose console=tty1 ---
    initrd /casper/initrd
}
EOF

  # EFI standalone
  grub-mkstandalone \
    --format=x86_64-efi \
    --output="$WORK/bootx64.efi" \
    --locales="" \
    --fonts="" \
    "boot/grub/grub.cfg=$ISO_DIR/boot/grub/grub.cfg"

  mkdir -p "$ISO_DIR/EFI/BOOT"
  cp "$WORK/bootx64.efi" "$ISO_DIR/EFI/BOOT/BOOTX64.EFI"

  # El Torito para Legacy BIOS
  mkdir -p "$ISO_DIR/boot/grub/i386-pc"
  grub-mkimage \
    -O i386-pc-eltorito \
    -o "$ISO_DIR/boot/grub/i386-pc/eltorito.img" \
    -p /boot/grub \
    biosdisk iso9660 part_msdos part_gpt normal linux configfile search

  ok "GRUB gerado (UEFI + Legacy)"
}

# ---------------------------------------------------------------- iso
make_iso() {
  log "9/9" "Gerando ISO"

  xorriso -as mkisofs \
    -o "$OUTPUT" \
    -V "LOGITECNO_INV" \
    -isohybrid-mbr "$ISOHDPFX" \
    -c boot/boot.cat \
    -b boot/grub/i386-pc/eltorito.img \
    -no-emul-boot -boot-load-size 4 -boot-info-table \
    -eltorito-alt-boot \
    -e EFI/BOOT/BOOTX64.EFI \
    -no-emul-boot \
    -isohybrid-gpt-basdat \
    "$ISO_DIR"

  [[ -f "$OUTPUT" ]] || die "ISO nao foi gerada"

  ok "ISO: $OUTPUT ($(du -h "$OUTPUT" | cut -f1))"
}

# ---------------------------------------------------------------- summary
summary() {
  cat <<EOF

============================================================
  BUILD CONCLUIDO
============================================================

  ISO:      $OUTPUT
  Tamanho:  $(du -h "$OUTPUT" | cut -f1)

  Testar em QEMU (sem pendrive):
    sudo qemu-system-x86_64 -m 2048 -enable-kvm -cdrom "$OUTPUT"

  Gravar no pendrive (SEM particao de dados):
    sudo umount /dev/sdX?* 2>/dev/null
    sudo wipefs -a /dev/sdX
    sudo dd if="$OUTPUT" of=/dev/sdX bs=4M status=progress oflag=sync conv=fsync
    sudo partprobe /dev/sdX
    lsblk /dev/sdX -o NAME,SIZE,FSTYPE,LABEL

  ATENCAO: substitua /dev/sdX pelo pendrive correto.
  Use 'lsblk' para identificar. Errar o dispositivo apaga seu disco.

  Sem rede: o agente mostra QR Code + JSON na tela.
  Com rede: envia para o webhook.

============================================================
EOF
}

# ---------------------------------------------------------------- main
main() {
  printf '\n\033[1m=== LogiTecno ISO Builder ===\033[0m\n'
  printf 'Projeto: %s\n' "$PROJECT_ROOT"
  printf 'Work:    %s\n' "$WORK"

  preflight
  build_rootfs
  install_agent
  configure_systemd
  configure_live
  pack_squashfs
  copy_kernel
  build_grub
  make_iso
  summary
}

main "$@"