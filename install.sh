#!/bin/bash
set -e

REPO="cr0hn/outbound-lb"
INSTALL_DIR="/usr/local/bin"
BINARY="outbound-lb"

# Detectar OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux) OS="linux" ;;
  darwin) OS="darwin" ;;
  *) echo "OS no soportado: $OS"; exit 1 ;;
esac

# Detectar arquitectura
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  armv7*|armv6*) ARCH="armv7" ;;
  *) echo "Arquitectura no soportada: $ARCH"; exit 1 ;;
esac

# Obtener última versión
VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
VERSION_NUM=${VERSION#v}

# Descargar
FILENAME="${BINARY}_${VERSION_NUM}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/${REPO}/releases/download/${VERSION}/${FILENAME}"

echo "Descargando ${BINARY} ${VERSION} para ${OS}/${ARCH}..."
curl -sL "$URL" -o "/tmp/${FILENAME}"

# Extraer e instalar
tar -xzf "/tmp/${FILENAME}" -C /tmp
sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/"
sudo chmod +x "${INSTALL_DIR}/${BINARY}"

# Limpiar
rm -f "/tmp/${FILENAME}"

# Verificar
echo "Instalado: $(${BINARY} --version 2>/dev/null || echo ${VERSION})"
echo "Ubicación: ${INSTALL_DIR}/${BINARY}"
