#!/bin/bash

set -u

RIEMANN_VERSION=$1
RIEMANN_URL=https://github.com/riemann/riemann/releases/download/${RIEMANN_VERSION}/riemann-${RIEMANN_VERSION}.tar.bz2

BIN=riemann-${RIEMANN_VERSION}/bin/riemann

if [ ! -f "${BIN}" ]; then
    echo "Download Riemann"
    wget "${RIEMANN_URL}"
    echo "Untar Riemann"
    tar xjf "riemann-${RIEMANN_VERSION}.tar.bz2"
fi

echo "Launch Riemann"
${BIN} &
RIEMANN_PID=$!
sleep 10
echo "Launch tests"
echo
echo
go test -v -race ./... -tags=integration
echo
echo
echo "Stop Riemann"
kill -9 ${RIEMANN_PID}
