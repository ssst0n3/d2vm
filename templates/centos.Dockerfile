FROM {{ .Image }}

USER root

{{ $version := atoi .Release.VersionID }}

{{ if le $version 8 }}
RUN sed -i 's/mirrorlist/#mirrorlist/g' /etc/yum.repos.d/CentOS-* && \
    sed -i 's|#baseurl=http://mirror.centos.org|baseurl=http://vault.centos.org|g' /etc/yum.repos.d/CentOS-*
{{ end }}

# RUN yum update -y

# See https://bugzilla.redhat.com/show_bug.cgi?id=1917213
RUN yum install -y \
{{ if le $version 8 }}
    grubby centos-linux-release \
{{ end }}
{{- if .Kernel }}
    kernel \
{{- end }}
    systemd \
    NetworkManager \
{{- if .GrubBIOS }}
    grub2 \
{{- end }}
{{- if .GrubEFI }}
    grub2 grub2-efi-x64 grub2-efi-x64-modules \
{{- end }}
    e2fsprogs \
    sudo && \
    systemctl enable NetworkManager && \
    systemctl unmask systemd-remount-fs.service && \
    systemctl unmask getty.target && \
    find /boot -type l -exec rm {} \;

{{ if .Luks }}
RUN yum install -y cryptsetup && \
    dracut --no-hostonly --regenerate-all --force --install="/usr/sbin/cryptsetup"
{{ else }}
RUN dracut --no-hostonly --regenerate-all --force
{{ end }}

{{ if .Password }}RUN echo "root:{{ .Password }}" | chpasswd {{ end }}

{{- if not .Grub }}
# Select the kernel image and initramfs in a way that tolerates the
# CentOS/RHEL layout where /boot/vmlinuz-* may be missing or a symlink
# (removed above) and the real image lives under /usr/lib/modules/*/vmlinuz.
RUN set -e; \
    if [ -n "$(ls -t /boot/vmlinuz-* 2>/dev/null | head -n1)" ]; then \
      ksrc="$(ls -t /boot/vmlinuz-* | head -n1)"; \
    elif [ -n "$(ls -t /usr/lib/modules/*/vmlinuz 2>/dev/null | head -n1)" ]; then \
      ksrc="$(ls -t /usr/lib/modules/*/vmlinuz | head -n1)"; \
    else \
      echo "d2vm: kernel image not found in /boot/vmlinuz-* or /usr/lib/modules/*/vmlinuz" >&2; \
      exit 1; \
    fi; \
    case "$ksrc" in \
      /boot/vmlinuz-*) kver="${ksrc#/boot/vmlinuz-}" ;; \
      */vmlinuz)        kver="$(basename "$(dirname "$ksrc")")" ;; \
      *)                kver="" ;; \
    esac; \
    cp -f "$ksrc" /boot/vmlinuz; \
    if [ -n "$(ls -t /boot/initramfs-*.img 2>/dev/null | head -n1)" ]; then \
      isrc="$(ls -t /boot/initramfs-*.img | head -n1)"; \
    elif command -v dracut >/dev/null 2>&1 && [ -n "$kver" ]; then \
      isrc="/boot/initramfs-${kver}.img"; \
      dracut --no-hostonly --force "$isrc" "$kver"; \
    else \
      echo "d2vm: initramfs not found in /boot/initramfs-*.img and dracut unavailable" >&2; \
      exit 1; \
    fi; \
    cp -f "$isrc" /boot/initrd.img
{{- end }}

RUN yum clean all && \
    rm -rf /var/cache/yum
