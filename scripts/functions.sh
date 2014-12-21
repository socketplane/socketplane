#!/bin/sh

# Utility function to test if a command exists
command_exists() {
    hash $@ 2>/dev/null
}

# Colorized Command Output
log_info() {
    printf "\033[0;36m$@\033[0m\n"
}

log_notice() {
    printf "\033[0;32m$@\033[0m\n"
}

log_warn() {
    printf "\033[0;33m$@\033[0m\n"
}

log_error() {
    printf "\033[0;35m$@\033[0m\n"
}

log_fatal() {
    printf "\033[0;31m$@\033[0m\n"
}

log_debug() {
    printf "\033[0;37m$@\033[0m\n"
}

log_step() {
    log_info "-----> $@"
}

indent() {
    sed -u "s/^/           /"
}
