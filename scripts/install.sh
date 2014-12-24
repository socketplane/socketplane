#!/bin/sh

set -e

# Utility to check if a command exists
command_exists() {
    hash $@ 2>/dev/null
}

# Run as root only
if [ "$(id -u)" != "0" ]; then
    echo >&2 "Please run as root"
    exit 1
fi

if command_exists socketplane; then
    echo >&2 'Warning: "socketplane" command appears to already exist.'
    echo >&2 'Please ensure that you do not already have socketplane installed.'
    exit 1
fi

curl=''
if command_exists curl; then
    curl='curl -sSL -o'
elif command_exists wget; then
    curl='wget -q -o'
fi

if [ ! -d /opt/socketplane ]; then
    mkdir -p /opt/socketplane
fi

if [ ! -f /opt/socketplane/socketplane]; then
    $curl /opt/socketplane/socketplane https://raw.githubusercontent.com/socketplane/socketplane/master/scripts/socketplane.sh
fi

if [ ! -f /opt/socketplane/functions.sh ]; then
    $curl /opt/socketplane/functions.sh https://raw.githubusercontent.com/socketplane/socketplane/master/scripts/functions.sh
fi

chmod +x /opt/socketplane/socketplane

if [ ! -f /usr/bin/socketplane ]; then
    ln -s /opt/socketplane/socketplane /usr/bin/socketplane
fi

socketplane install
