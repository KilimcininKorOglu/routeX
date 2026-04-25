#!/bin/sh
set -e

REPO_BASE="https://kilimcininkoroglu.github.io/routex/packages"
PKG_NAME="routex"

#
# OpenWrt
#

if [ -f /etc/openwrt_release ]; then
  echo "Platform: OpenWrt"

  if [ -f /bin/opkg ]; then
    echo "Paket yöneticisi: opkg"

    ARCH=""
    while read -r _ A _; do
      case "$A" in
        aarch64_cortex-a53|aarch64_cortex-a72|aarch64_cortex-a76|aarch64_generic|\
        arm_arm926ej-s|arm_cortex-a15_neon-vfpv4|arm_cortex-a5_vfpv4|\
        arm_cortex-a7|arm_cortex-a7_neon-vfpv4|arm_cortex-a7_vfpv4|\
        arm_cortex-a8_vfpv3|arm_cortex-a9|arm_cortex-a9_neon|arm_cortex-a9_vfpv3-d16|\
        arm_fa526|arm_xscale|\
        i386_pentium4|i386_pentium-mmx|\
        loongarch64_generic|\
        mips_24kc|mips_4kec|mips64el_mips64r2|mips64_mips64r2|mips64_octeonplus|\
        mipsel_24kc_24kf|mipsel_24kc|mipsel_74kc|mipsel_mips32|mips_mips32|\
        riscv64_generic|x86_64)
          ARCH="$A"
          break
          ;;
      esac
    done <<EOF
$(/bin/opkg print-architecture)
EOF

    if [ -z "$ARCH" ]; then
      echo "Desteklenen mimari bulunamadı" >&2
      exit 1
    fi

    echo "Mimari: ${ARCH}"

    mkdir -p /etc/opkg
    echo "src/gz ${PKG_NAME}_${ARCH} ${REPO_BASE}/openwrt-ipk/${ARCH}" > /etc/opkg/${PKG_NAME}.conf

    echo "Depo eklendi: /etc/opkg/${PKG_NAME}.conf"
    echo "Kurulum için: opkg update && opkg install ${PKG_NAME}"
    exit 0

  elif [ -f /usr/bin/apk ]; then
    echo "Paket yöneticisi: apk"

    APK_REPOS_FILE="/etc/apk/repositories.d/distfeeds.list"
    if [ ! -f "${APK_REPOS_FILE}" ]; then
      echo "Mimari belirlenemedi" >&2
      exit 1
    fi

    ARCH=""
    while IFS= read -r url; do
      case "${url}" in
        ""|\#*) continue ;;
      esac
      A=$(echo "${url}" | sed -n 's|^https://downloads\.openwrt\.org/releases/.*/packages/\([^/]*\)/packages/packages\.adb$|\1|p')
      case "$A" in
        aarch64_cortex-a53|aarch64_cortex-a72|aarch64_cortex-a76|aarch64_generic|\
        arm_arm926ej-s|arm_cortex-a15_neon-vfpv4|arm_cortex-a5_vfpv4|\
        arm_cortex-a7|arm_cortex-a7_neon-vfpv4|arm_cortex-a7_vfpv4|\
        arm_cortex-a8_vfpv3|arm_cortex-a9|arm_cortex-a9_neon|arm_cortex-a9_vfpv3-d16|\
        arm_fa526|arm_xscale|\
        i386_pentium4|i386_pentium-mmx|\
        loongarch64_generic|\
        mips_24kc|mips_4kec|mips64el_mips64r2|mips64_mips64r2|mips64_octeonplus|\
        mipsel_24kc_24kf|mipsel_24kc|mipsel_74kc|mipsel_mips32|mips_mips32|\
        riscv64_generic|x86_64)
          ARCH="$A"
          break
          ;;
      esac
    done < "${APK_REPOS_FILE}"

    if [ -z "$ARCH" ]; then
      echo "Desteklenen mimari bulunamadı" >&2
      exit 1
    fi

    echo "Mimari: ${ARCH}"

    mkdir -p /etc/apk/repositories.d
    echo "${REPO_BASE}/openwrt-apk/${ARCH}" > /etc/apk/repositories.d/${PKG_NAME}.list

    echo "Depo eklendi: /etc/apk/repositories.d/${PKG_NAME}.list"
    echo "Kurulum için: apk update && apk add --allow-untrusted ${PKG_NAME}"
    exit 0

  else
    echo "Desteklenen paket yöneticisi bulunamadı" >&2
    exit 1
  fi
fi

#
# Entware
#

if [ -f /opt/etc/entware_release ]; then
  echo "Platform: Entware"

  ARCH=""
  while read -r _ A _; do
    case "$A" in
      aarch64-3.10|aarch64-3.10_kn|\
      armv7-3.2|\
      mips-3.4|mips-3.4_kn|\
      mipsel-3.4|mipsel-3.4_kn)
        ARCH="$A"
        break
        ;;
    esac
  done <<EOF
$(/opt/bin/opkg print-architecture)
EOF

  if [ -z "$ARCH" ]; then
    echo "Desteklenen mimari bulunamadı" >&2
    exit 1
  fi

  echo "Mimari: ${ARCH}"

  mkdir -p /opt/etc/opkg
  echo "src/gz ${PKG_NAME}_${ARCH} ${REPO_BASE}/entware-ipk/${ARCH}" > /opt/etc/opkg/${PKG_NAME}.conf

  echo "Depo eklendi: /opt/etc/opkg/${PKG_NAME}.conf"
  echo "Kurulum için: opkg update && opkg install ${PKG_NAME}"
  exit 0
fi

echo "Desteklenen platform bulunamadı (OpenWrt veya Entware gerekli)" >&2
exit 1
