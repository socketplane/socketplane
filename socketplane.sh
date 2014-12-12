#!/bin/sh
# Temporary wrapper for OVS until native Docker integration is available upstream

command_exists() {
    command -v "$@" > /dev/null 2>&1
}

verify_ovs() {
    OS=NOT_LINUX
    RELEASE=NOT_LINUX
    CODENAME=NOT_LINUX
    ARCH=$(uname -m)
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi
    if [ "$ARCH" = "i686" ]; then ARCH="i386"; fi
    if which lsb_release &> /dev/null; then
        OS=$(lsb_release -is)
        RELEASE=$(lsb_release -rs)
        CODENAME=$(lsb_release -cs)
    fi
    echo "Detected Linux distribution: $OS $RELEASE $CODENAME $ARCH"
    if ! echo $OS | egrep 'Ubuntu|Debian|Fedora'; then
        echo "Supported operating systems are: Ubuntu, Debian and Fedora."
        exit 1
    fi
    if ! `which ovsdb-server &> /dev/null && which ovs-vswitchd &> /dev/null`; then
	    echo "ovsdb-server and ovs-vswitchd were found, checking the processes next.."
	else
	    echo "OVS was not found in the current path, installing now.."
	    install_ovs
	fi
    SWPID=$(ps aux | grep ovs-vswitchd | grep -v grep | awk '{ print $2 }')
    DBPID=$(ps aux | grep ovsdb-server | grep -v grep | awk '{ print $2 }')
    if [ -z "$SWPID" ] && [ -z "$DBPID" ]; then
        echo "OVS is installed but not running, attempting to start the service.."
	    if echo $OS | egrep 'Ubuntu'; then
            sudo /etc/init.d/openvswitch-switch start
	    elif echo $OS | egrep 'Debian' &> /dev/null; then
            sudo /etc/init.d/openvswitch start
        else echo $OS | egrep 'Fedora' &> /dev/null;
            sudo sudo systemctl start openvswitch.service
        fi
        sleep 1
    fi
	echo "OVS is installed and running, setting the OVSDB listener.."
	sudo ovs-vsctl set-manager ptcp:6640
}

install_ovs() {
    OS=NOT_LINUX
    RELEASE=NOT_LINUX
    CODENAME=NOT_LINUX
    ARCH=$(uname -m)
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi
    if [ "$ARCH" = "i686" ]; then ARCH="i386"; fi
    if which lsb_release &> /dev/null; then
        OS=$(lsb_release -is)
        RELEASE=$(lsb_release -rs)
        CODENAME=$(lsb_release -cs)
    fi
    echo "Detected Linux distribution: $OS $RELEASE $CODENAME $ARCH"
    if ! echo $OS | egrep 'Ubuntu|Debian|Fedora'; then
        echo "Supported operating systems are: Ubuntu, Debian and Fedora."
        exit 1
    fi
    test -e /etc/debian_version && OS="Debian"
    grep Ubuntu /etc/lsb-release &> /dev/null && OS="Ubuntu"
    if [ "$OS" = "Ubuntu" ] || [ "$OS" = "Debian" ]; then
        install='sudo apt-get -y install '
        $install openvswitch-switch  openvswitch-datapath-dkms
        sleep 3
        ovs-vsctl set-manager ptcp:6640
        if ! which lsb_release &> /dev/null; then
            $install lsb-release
        fi
    fi
    test -e /etc/fedora-release && OS="Fedora"
        if [ "$OS" = "Fedora" ]; then
        install='sudo yum -y install '
        $install openvswitch
        sleep 3
        ovs-vsctl set-manager ptcp:6640
        if ! which lsb_release &> /dev/null; then
            $install redhat-lsb-core
        fi
    fi
    sleep 1
}

remove_ovs() {
    OS=NOT_LINUX
    RELEASE=NOT_LINUX
    CODENAME=NOT_LINUX
    ARCH=$(uname -m)
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi
    if [ "$ARCH" = "i686" ]; then ARCH="i386"; fi
    if which lsb_release &> /dev/null; then
        OS=$(lsb_release -is)
        RELEASE=$(lsb_release -rs)
        CODENAME=$(lsb_release -cs)
    fi
    echo "Detected Linux distribution: $OS $RELEASE $CODENAME $ARCH"
    if ! echo $OS | egrep 'Ubuntu|Debian|Fedora'; then
        echo "Supported operating systems are: Ubuntu, Debian and Fedora."
        exit 1
    fi
    echo "Removing existing Open vSwitch packages:"
    sudo apt-get remove -y openvswitch-switch openvswitch-common
}

stop_all_images() {
    for IMAGE_ID in $(docker ps | grep socketplane/socketplane | awk '{ print $1; }')
        do
        echo "Removing socketplane image: $IMAGE_ID"
        docker stop $IMAGE_ID
    done
}

verify_docker_sh() {
    if command_exists if command_exists sudo ps -ef | grep docker |awk '{print $2}' && [ -e /var/run/docker.sock ]; then
        (set -x $dk '"Docker has been installed"') || true
        echo "Docker appears to already be installed and running.."
        else
            echo "Docker is not installed, downloading and installing now"
            wget -qO- https://get.docker.com/ | sh
    fi
}

container_run() {
    echo "Downloading and starting the SocketPlane container"
    # The following will prompt for:
    #------------------------------#
    # userid
    # password
    # email
    sudo docker login
    sudo docker run -itd --net=host socketplane/socketplane
}

usage() {
cat << EOF
usage: $0 options
EOF
}

case "$1" in
	install)
	    echo "Installing StackPlane Software.."
	    verify_ovs
        verify_docker_sh
	    container_run
		 echo "Done."
		;;
	uninstall)
	    echo "Removing StackPlane Software.."
		remove_ovs
        stop_all_images
		;;
	*)
	cat << EOF
usage: $0 options

Install and run SocketPlane. This will install various packages
from the distributions default repositories if not already installed,
including open vswitch, docker and the socketplane control image from
dockerhub.

OPTIONS:
    socketplane help              Help and usage
    socketplane install           Install SocketPlane
    socketplane uninstall         Remove Socketplane installation and dependencies

EOF
esac
exit