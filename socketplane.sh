#!/bin/bash
# Temporary wrapper for OVS until native Docker integration is available upstream

set -e

command_exists() {
    hash $@ 2>/dev/null
}

puts_command() {
    echo -e "\033[0;32m$@ \033[0m"
}

puts_step() {
    echo -e "\033[1;33m-----> $@ \033[0m"
}

puts_warn() {
    echo -e "\033[0;31m ! $@ \033[0m"
}

check_supported_os() {
    puts_step "Detected Linux distribution: $OS $RELEASE $CODENAME $ARCH"
    if ! echo $OS | grep -E 'Ubuntu|Debian|Fedora|RedHat' > /dev/null; then
        puts_warn "Supported operating systems are: Ubuntu, Debian and Fedora."
        exit 1
    fi
}

get_status() {
    OS="NOT_LINUX"
    RELEASE="NOT_LINUX"
    CODENAME="NOT_LINUX"
    ARCH=$(uname -m)

    if [ "$ARCH" = "x86_64" ]; then
        ARCH="amd64";
    fi

    if [ "$ARCH" = "i686" ]; then
        ARCH="i386";
    fi

    if command_exists lsb_release; then
        OS=$(lsb_release -is)
        RELEASE=$(lsb_release -rs)
        CODENAME=$(lsb_release -cs)
    elif [ -f /etc/debian_version ]; then
        OS="Debian"
        RELEASE="UNDETECTED"
        CODENAME="UNDETECTED"
    elif [ -f /etc/redhat-release ]; then
        OS="RedHat"
        RELEASE="UNDETECTED"
        CODENAME="UNDETECTED"
    fi

    DOCKER_SVER="NOT_INSTALLED"
    DOCKER_CVER="NOT_INSTALLED"
    if command_exists docker || command_exists lxc-docker; then
        DOCKER_SVER=$(docker version | grep "Server version:" |  awk '{ print $3 }')
        DOCKER_CVER=$(docker version | grep "Client API version:" |  awk '{ print $4 }')
    fi

    OVS_SVER="NOT_INSTALLED"
    if command_exists ovs-appctl; then
        OVS_SVER=$(ovs-appctl -V | grep "ovs-" |  awk '{ print $4 }')
    fi
}

show_reqs() {
    echo "Socketplane  Docker Host Requirnments:"
    echo ".. Open vSwitch Environment:"
    echo ".... Archicture:              amd64 or i386"
    echo "....   Current:               $ARCH"
    echo ".... Operating System:         Ubuntu, Debian and Fedora"
    echo "....   Current:               $OS"
    echo ".... Open vSwitch Version:     2.1 or higher"
    echo "....   Current:               $OVS_SVER"
    echo ".. Docker Environment:"
    echo ".... Docker Server Version:   1.4 or higher"
    echo "....   Current:               $DOCKER_SVER"
    echo ".... Docker Client Version:   1.16 or higher"
    echo "....   Current:               $DOCKER_SVER"
}

verify_ovs() {
    if command_exists ovsdb-server && command_exists ovs-vswitchd ; then
        puts_step "OVS already installed"
    else
        puts_warn "OVS was not found in the current path, installing now.."
        install_ovs
    fi

    # Make sure the processes are started
    if [ $OS == "Debian" ] || [ $OS == 'Ubuntu' ]; then
        if $(service openvswitch-switch status | grep "stop"); then
            service openvswitch-switch start
        fi
    elif [ $OS == 'Fedora' ] || [ $OS == 'RedHat' ]; then
        systemctl start openvswitch.service
    fi

    sleep 1
    puts_step "Setting OVSDB Listener"
    ovs-vsctl set-manager ptcp:6640
}

install_ovs() {
    if [ "$OS" = "Ubuntu" ] || [ "$OS" = "Debian" ]; then
        apt-get -y install openvswitch-switch > /dev/null
    elif [ $OS == 'Fedora' ] || [ $OS == 'RedHat' ]; then
        yum -q -y install openvswitch
        systemctl start openvswitch.service
    fi

    sleep 3
    ovs-vsctl set-manager ptcp:6640
    sleep 1
}

remove_ovs() {
    puts_step "Removing existing Open vSwitch packages:"
    if [ "$OS" = "Ubuntu" ] || [ "$OS" = "Debian" ]; then
        apt-get -y remove openvswitch-switch > /dev/null
    elif [ $OS == 'Fedora' ] || [ $OS == 'RedHat' ]; then
        yum -q -y remove openvswitch
    fi
}

verify_docker() {
    if ! command_exists docker; then
        puts_warn "Docker is not installed. Installing now..."
        if [ $OS == 'Fedora' ] || [ $OS == 'RedHat' ]; then
            yum -q -y remove docker > /dev/null
        fi
        wget -qO- https://get.docker.com/ | sh
    fi

    if [ $OS == "Debian" ] || [ $OS == 'Ubuntu' ]; then
        if $(service docker status | grep "stop"); then
            service docker start
        fi
    elif [ $OS == 'Fedora' ] || [ $OS == 'RedHat' ]; then
        systemctl start docker.service
    fi
}

start_socketplane() {
    puts_step "Starting the SocketPlane container"

    if [ ! -z $(docker ps | grep socketplane/socketplane | awk '{ print $1; }') ]; then
        puts_warn "SocketPlane container is already running"
        return 1
    fi

    # The following will prompt for:
    #------------------------------#
    # userid
    # password
    # email
    docker login
    mkdir -p /var/run/socketplane
    cid=$(docker run -itd --privileged=true --net=host socketplane/socketplane)
    echo $cid > /var/run/socketplane/cid
}

stop_socketplane() {
    puts_step "Stopping the SocketPlane container"
    if ! command_exists docker; then
        puts_warn "Docker is not installed"
        exit 1
    fi

    for IMAGE_ID in $(docker ps | grep socketplane/socketplane | awk '{ print $1; }'); do
        echo "Removing socketplane image: $IMAGE_ID"
        docker stop $IMAGE_ID > /dev/null
        sleep 1
        docker rm $IMAGE_ID > /dev/null
    done
    rm /var/run/socketplane/cid
}

remove_socketplane() {
    puts_step "Removing the SocketPlane container image"
    if ! command_exists docker; then
        puts_warn "Docker is not installed"
        exit 1
    fi

    for IMAGE_ID in $(docker images | grep socketplane/socketplane | awk '{ print $1; }'); do
        echo "Removing socketplane image: $IMAGE_ID"
        docker rmi $IMAGE_ID > /dev/null
    done
}

logs() {
    if [ ! -f /var/run/socketplane/cid ] || [ -z $(cat /var/run/socketplane/cid) ]; then
        puts_warn "SocketPlane container is not running"
    fi
    docker logs $(cat /var/run/socketplane/cid)
}

run() {
    args=$(echo $@ | sed 's/^run\s//g')
    $cid=$(docker run $args)
    $cPid=$(docker inspect --format='{{ .State.Pid }}' $cid)
    $cName=$(docker inspect --format='{{ .Name }}' $cid)

    sudo curl -S -v -X POST http://localhost:6675/v0.1/connections -d '{ "container_id": "$cid", "container_name": "$cName", "container_pid": "$cPid"}'

    # ToDo: grep return JSON to get OVS_Port_Id

    [ ! -d /var/run/netns ] && mkdir -p /var/run/netns
    [ -f /var/run/netns/$NSPID ] && rm -f /var/run/netns/$NSPID
    ln -s /proc/$NSPID/ns/net /var/run/netns/$NSPID

    # ToDo: add ovs interface to container
}

usage() {
    cat << EOF
usage: $0 <command>

Install and run SocketPlane. This will install various packages
from the distributions default repositories if not already installed,
including open vswitch, docker and the socketplane control image from
dockerhub.

INSTALLATION COMMANDS:
    socketplane help                Help and usage
    socketplane install             Install SocketPlane (installs docker and openvswitch)
    socketplane uninstall           Remove Socketplane installation
    socketplane clean               Remove Socketplane installation and dependencies (docker and openvswitch)
    socketplane show_reqs           List all socketplane installation requirements

RUNTIME COMMANDS:
    socketplane logs                Show SocketPlane container logs
    socketplane run <args>          Run a container where <args> are the same as "docker run"

EOF
}

if [ "$EUID" -ne 0 ]
  then echo "Please run as root"
  exit
fi

case "$1" in
    install)
        puts_command "Installing SocketPlane..."
        get_status
        check_supported_os
        verify_ovs
        verify_docker
        start_socketplane
        puts_command "Done!!!"
        ;;
    uninstall)
        puts_command "Uninstalling SocketPlane..."
        get_status
        check_supported_os
        stop_socketplane
        puts_command "Done!!!"
        ;;
    clean)
        puts_command "Removing SocketPlane and all it's dependencies..."
        get_status
        check_supported_os
        remove_ovs
        stop_socketplane
        remove_socketplane
        puts_command "Done!!!"
        ;;
    logs)
        logs
        ;;
    run)
        run $@
        ;;
    show_reqs)
        get_status
        check_supported_os
        show_reqs
        ;;
    *)
        usage
        ;;
esac
exit
