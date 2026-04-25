#!/bin/sh
set -e

DIR="$1"
if [ -z "$DIR" ] || [ ! -d "$DIR" ]; then
  echo "Kullanım: $0 <ipk-dizini>" >&2
  exit 1
fi

cd "$DIR"

for ipk in *.ipk; do
  [ -f "$ipk" ] || continue

  TMPDIR=$(mktemp -d)
  tar xzf "$ipk" -C "$TMPDIR" ./control.tar.gz 2>/dev/null || tar xzf "$ipk" -C "$TMPDIR" control.tar.gz 2>/dev/null
  tar xzf "$TMPDIR/control.tar.gz" -C "$TMPDIR" ./control 2>/dev/null || tar xzf "$TMPDIR/control.tar.gz" -C "$TMPDIR" control 2>/dev/null

  CONTROL=""
  if [ -f "$TMPDIR/control" ]; then
    CONTROL="$TMPDIR/control"
  elif [ -f "$TMPDIR/./control" ]; then
    CONTROL="$TMPDIR/./control"
  fi

  if [ -z "$CONTROL" ]; then
    rm -rf "$TMPDIR"
    continue
  fi

  cat "$CONTROL"
  echo "Filename: $ipk"
  echo "Size: $(wc -c < "$ipk" | tr -d ' ')"
  echo "SHA256sum: $(sha256sum "$ipk" | cut -d' ' -f1)"
  echo ""

  rm -rf "$TMPDIR"
done > Packages

gzip -k -f Packages
echo "Packages ve Packages.gz oluşturuldu: $DIR ($(ls *.ipk 2>/dev/null | wc -l | tr -d ' ') paket)"
