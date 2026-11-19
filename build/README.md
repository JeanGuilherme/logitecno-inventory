# Build da ISO

## Pré-requisitos

Instalados no Ubuntu:

    sudo apt install -y debootstrap squashfs-tools xorriso mtools \
      grub-pc-bin grub-efi-amd64-bin grub-common isolinux syslinux-utils

## Ordem de execução

1. Compilar o agente:

       cd ../agent
       go build -o collect .

2. Gerar a ISO:

       cd ../build
       sudo ./build-iso.sh

3. Testar em QEMU (sem precisar de pendrive):

       sudo qemu-system-x86_64 -m 2048 -enable-kvm \
         -cdrom ../iso/logitecno-inventory.iso

## Pendências

- Tema GRUB com identidade LogiTecno (aguarda assets visuais)
- Configurar `LOGITECNO_WEBHOOK_URL` no `/etc/environment` do rootfs
- Testar boot UEFI e Legacy em hardware real
- Validar Secure Boot (provavelmente vai bloquear — precisa de shim)