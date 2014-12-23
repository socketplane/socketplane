#!/bin/sh

# Utility to check if a command exists
command_exists() {
    hash $@ 2>/dev/null
}

# Run as root only
if [ "$(id -u)" != "0" ]; then
    log_fatal "Please run as root"
    exit 1
fi

if command_exists socketplane; then
    echo >&2 'Warning: "socketplane" command appears to already exist.'
    echo >&2 'Please ensure that you do not already have socketplane installed.'
    exit 1
fi

curl=''
if command_exists curl; then
    curl='curl'
elif command_exists wget; then
    curl='wget'
fi

mkdir -p /opt/socketplane

$curl -o /opt/socketplane/socketplane https://raw.githubusercontent.com/socketplane/socketplane/master/scripts/socketplane.sh
$curl -o /opt/socketplane/functions.sh https://raw.githubusercontent.com/socketplane/socketplane/master/scripts/functions.sh

chmod +x /opt/socketplane/socketplane
ln -s /opt/socketplane/socketplane /usr/bin/socketplane

socketplane install
