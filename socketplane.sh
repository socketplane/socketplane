#!/bin/sh
# Temporary wrapper for OVS until native Docker integration is available upstream
set -e

usage() {
    cat << EOF
NAME:
    socketplane - Install, Configure and Run SocketPlane

VERSION:
    0.1

USAGE:
    $0 <command> [command_options] [arguments...]

COMMANDS:
    help
            Help and usage

    install [unattended]
            Install SocketPlane (installs docker and openvswitch)

    uninstall
            Remove Socketplane installation

    clean
            Remove Socketplane installation and dependencies (docker and openvswitch)

    deps
            Show SocketPlane dependencies

    logs
            Show SocketPlane container logs

    info [container_id]
            Show SocketPlane info for all containers, or for a given container_id

    run [--network foo] <docker_run_args>
            Run a container and optionally specify which network to attach to

    start <container_id>
            Start a <container_id>

    stop <container_id>
            Stop the <container_id>

    rm <container_id>
            Remove the <container_id>

    network list
            List all created networks

    network info <name>
            Display information about a given network

    network create <name> [cidr]
            Create a network

    network delete <name> [cidr]
            Delete a network

EOF
}

# Utility function to test if a command exists
command_exists() {
    hash $@ 2>/dev/null
}

# Colorized Command Output
puts_command() {
    printf "\033[0;32m$@ \033[0m\n"
}

puts_step() {
    printf "\033[1;33m-----> $@ \033[0m\n"
}

puts_warn() {
    printf "\033[0;31m ! $@ \033[0m\n"
}

# Operating System Check Utility
check_supported_os() {
    puts_step "Detected Linux distribution: $OS $ARCH"
    if ! echo $OS | grep -E 'Ubuntu|Debian|Fedora|RedHat' > /dev/null; then
        puts_warn "Supported operating systems are: Ubuntu, Debian and Fedora."
        exit 1
    fi
}

# Utility function to attach a port to a network namespace
attach()    # OVS Port ID
            # IP Address
            # Subnet
            # MAC Address
            # Gateway IP
            # Namespace PID
{
    # see: https://docs.docker.com/articles/networking/

    [ ! -d /var/run/netns ] && mkdir -p /var/run/netns
    [ -f /var/run/netns/$6 ] && rm -f /var/run/netns/$6
    ln -s /proc/$6/ns/net /var/run/netns/$6

    ip link set dev $1 netns $6
    ip netns exec $6 ip link set dev $1 address $4
    ip netns exec $6 ip link set dev $1 up
    ip netns exec $6 ip addr add $2$3 dev $1
    ip netns exec $6 ip route add default via $5

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
        DOCKER_SVER=$(sudo docker version | grep "Server version:" |  awk '{ print $3 }')
        DOCKER_CVER=$(sudo docker version | grep "Client API version:" |  awk '{ print $4 }')
    fi

    OVS_SVER="NOT_INSTALLED"
    if command_exists ovs-appctl; then
        OVS_SVER=$(ovs-appctl -V | grep "ovs-" |  awk '{ print $4 }')
    fi
}

deps() {
    echo "Socketplane  Docker Host Requirements:"
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

kernel_opts(){
    if [ $(cat /etc/sysctl.conf | grep icmp_echo_ignore_broadcasts) ]; then
        sed -i 's/^#\?net\.ipv4\.icmp_echo_ignore_broadcasts.*$/net\.ipv4\.icmp_echo_ignore_broadcasts=0/g' /etc/sysctl.conf
    else
        echo 'net.ipv4.icmp_echo_ignore_broadcasts=0' >> /etc/sysctl.conf
    fi

    if [ $(cat /etc/sysctl.conf | grep ip_forward ) ]; then
        sed -i 's/^#\?net\.ipv4\.ip_forward.*$/net\.ipv4\.ip_forward=1/g' /etc/sysctl.conf
    else
        echo 'net.ipv4.ip_forward=1' >> /etc/sysctl.conf
    fi
    sysctl -p
}

verify_ovs() {
    if command_exists ovsdb-server && command_exists ovs-vswitchd ; then
        puts_step "OVS already installed"
    else
        puts_warn "OVS was not found in the current path, installing now.."
        install_ovs
    fi

    # Make sure the processes are started
    case "$OS" in
        Debian|Ubuntu)
            if $(service openvswitch-switch status | grep "stop"); then
                service openvswitch-switch start
            fi
            ;;
        Fedora|RedHat)
            systemctl start openvswitch.service
            ;;
    esac

    sleep 1
    puts_step "Setting OVSDB Listener"
    ovs-vsctl set-manager ptcp:6640
}

install_ovs() {
    puts_step  "Installing Open vSwitch.."
    case $OS in
        Debian|Ubuntu)
            apt-get -y install openvswitch-switch > /dev/null
            ;;
        Fedora|RedHat)
            yum -q -y install openvswitch
            systemctl start openvswitch.service
            ;;
    esac

    sleep 1
    ovs-vsctl set-manager ptcp:6640
    sleep 1
}

remove_ovs() {
    puts_step "Removing existing Open vSwitch packages:"
    case $OS in
        Debian|Ubuntu)
            apt-get -y remove openvswitch-switch > /dev/null
            ;;
        Fedora|RedHat)
            yum -q -y remove openvswitch
            ;;
    esac
}

verify_docker() {
    if ! command_exists docker; then
        puts_warn "Docker is not installed. Installing now..."
        case $OS in
            Fedora|RedHat)
                yum -q -y remove docker > /dev/null
                ;;
        esac

        if test -x "$(which curl 2>/dev/null)" ; then
            curl -sSL https://get.docker.com/ | sh
        elif test -x "$(which wget 2>/dev/null)" ; then
            wget -qO- https://get.docker.com/ | sh
        fi
    fi

    case $OS in
        Debian|Ubuntu)
            if $(service docker.io status | grep "stop"); then
                service docker.io start
            fi
            ;;
        Fedora|RedHat)
            systemctl start docker.service
            ;;
    esac
}

start_socketplane() {
    puts_step "Starting the SocketPlane container"

    if [ ! -n $(sudo docker ps | grep socketplane/socketplane | awk '{ print $1; }') ]; then
        puts_warn "A SocketPlane container is already running"
        return 1
    fi

    if [ "$1" = "unattended" ]; then
        [ -z $DOCKERHUB_USER ] && puts_warn "DOCKERHUB_USER not set" && exit 1
        [ -z $DOCKERHUB_PASS ] && puts_warn "DOCKERHUB_PASS not set" && exit 1
        [ -z $DOCKERHUB_MAIL ] && puts_warn "DOCKERHUB_MAIL not set" && exit 1
        [ -z $BOOTSTRAP ] && puts_warn "BOOTSTRAP not set" && exit 1

        sudo docker login -e $DOCKERHUB_MAIL -p $DOCKERHUB_PASS -u $DOCKERHUB_USER

        if [ "$BOOTSTRAP" == "true" ] ; then
            cid=$(sudo docker run -itd --privileged=true --net=host socketplane/socketplane socketplane --bootstrap=true --iface=eth1)
            puts_step "A SocketPlane container was started"
        else
            cid=$(sudo docker run -itd --privileged=true --net=host socketplane/socketplane socketplane --iface=eth1)
            puts_step "A SocketPlane container was started"
        fi
    else

        # The following will prompt for:
        #------------------------------#
        # userid
        # password
        # email
        sudo docker login
        while true; do
            read -p "Is this the first node in the cluster? (y/n)" yn
            case $yn in
                [Yy]* )
                    cid=$(sudo docker run -itd --privileged=true --net=host socketplane/socketplane socketplane --bootstrap=true --iface=eth1)
                    puts_step "A SocketPlane container was started"
                    break
                    ;;
                [Nn]* )
                    cid=$(sudo docker run -itd --privileged=true --net=host socketplane/socketplane socketplane --iface=eth1)
                    puts_step "A SocketPlane container was started"
                    break
                    ;;
                * )
                    echo "Please answer yes or no."
                    ;;
            esac
        done
    fi
    mkdir -p /var/run/socketplane
    echo $cid > /var/run/socketplane/cid
}

start_socketplane_image() {
    if ! command_exists docker; then
        puts_warn "Docker is not installed, please run \"./socketplane install\""
        exit 1
    fi

    for IMAGE_ID in $(sudo docker ps -a | grep socketplane/socketplane | awk '{ print $1; }'); do
            puts_step "Starting all SocketPlane containers $IMAGE_ID"
        sudo docker start ${IMAGE_ID} > /dev/null
    done

    if [[ ! -n $(sudo docker ps | grep socketplane/socketplane | awk '{ print $1; }') ]]; then
        puts_step  "All Socketplane agent containers are started."
    fi
}

stop_socketplane_image() {
    if ! command_exists docker; then
        puts_warn "Docker is not installed, please run \"./socketplane install\""
        exit 1
    fi

    for IMAGE_ID in $(sudo docker ps | grep socketplane/socketplane | awk '{ print $1; }'); do
            puts_step "Stopping the SocketPlane container $IMAGE_ID"
        sudo docker stop ${IMAGE_ID} > /dev/null
    done

    if [[ ! -n $(sudo docker ps | grep socketplane/socketplane | awk '{ print $1; }') ]]; then
        puts_step  "All Socketplane agent containers are stopped. Please run \"./socketplane.sh start\" to start them again"
    fi
}

stop_socketplane() {
    if ! command_exists docker; then
        puts_warn "Docker is not installed"
        exit 1
    fi

    for IMAGE_ID in $(sudo docker ps | grep socketplane/socketplane | awk '{ print $1; }'); do
        echo "Removing socketplane image: $IMAGE_ID"
        sudo docker stop $IMAGE_ID > /dev/null
        sleep 1
        sudo docker rm $IMAGE_ID > /dev/null
    done
    puts_warn "SocketPlane container images was deleted. Run \"./socketplane.sh install\" to download a new one."
}

remove_socketplane() {
    puts_step "Removing the SocketPlane container image"
    if ! command_exists docker; then
        puts_warn "Docker is not installed"
        exit 1
    fi

    for IMAGE_ID in $(sudo docker images | grep socketplane/socketplane | awk '{ print $1; }'); do
        echo "Removing socketplane image: $IMAGE_ID"
       sudo docker rmi $IMAGE_ID > /dev/null
    done
}

logs() {
    if [ ! -f /var/run/socketplane/cid ] || [ -z $(cat /var/run/socketplane/cid) ]; then
        puts_warn "SocketPlane container is not running"
        exit 1
    fi
    sudo docker logs $(cat /var/run/socketplane/cid)
}

info() {
    containerId=$1
    if [ -z "$containerId" ]; then
        curl -s -X GET http://localhost:6675/v0.1/connections | python -m json.tool
    else
        curl -s -X GET http://localhost:6675/v0.1/connections/$containerId | python -m json.tool
    fi
}

run() {
    cid=$(sudo docker run --net=none $@)
    cPid=$(sudo docker inspect --format='{{ .State.Pid }}' $cid)
    cName=$(sudo docker inspect --format='{{ .Name }}' $cid)

    json=$(curl -s -X POST http://localhost:6675/v0.1/connections -d "{ \"container_id\": \"$cid\", \"container_name\": \"$cName\", \"container_pid\": \"$cPid\" }")
    result=$(echo $json | sed 's/[,{}]/\n/g' | sed 's/^".*":"\(.*\)"/\1/g' | awk -v RS="" '{ print $6, $7, $8, $9, $10 }')

    attach $result $cPid

    echo $cid
}

delete() {
    sudo docker rm $@
    curl -s -X DELETE http://localhost:6675/v0.1/connections/$@
    sleep 1
    # clean up dangling symlinks
    find -L /var/run/netns -type l -delete
}

network_list() {
    exit
}

network_info() {
    exit
}

network_create() {
    exit
}

network_delete() {
    exit
}

# Run as root only
if [ "$(id -u)" != "0" ]; then
    puts_warn "Please run as root"
    exit 1
fi

case "$1" in
    help)
        usage
        ;;
    install)
        shift
        puts_command "Installing SocketPlane..."
        get_status
        check_supported_os
        kernel_opts
        verify_ovs
        verify_docker
        start_socketplane $@
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
        puts_command "Removing SocketPlane and all its dependencies..."
        get_status
        check_supported_os
        remove_ovs
        stop_socketplane
        remove_socketplane
        puts_command "Done!!!"
        ;;
    deps)
        get_status
        check_supported_os
        deps
        ;;
    logs)
        logs
        ;;
    info)
        shift
        info $@
        ;;
    run)
        shift
        run $@
        ;;
    stop)
        shift
        docker stop $@
        ;;
    start)
        shift
        docker start $@
        ;;
    rm)
        shift
        delete $@
        ;;
    network)
        shift
        case "$1" in
            list)
                network_list
                ;;
            info)
                network_info
                ;;
            create)
                network_create
                ;;
            delete)
                network_delete
                ;;
            *)
                puts_warn "Unknown Command"
                usage
                exit 1
        esac
        ;;
    agent)
        shift 1
        if [ "$@" == "start" ]; then
            start_socketplane_image
        elif [ "$@" == "stop" ]; then
            stop_socketplane_image
        else puts_warn "\"socketplane agent\" options are {stop|start}"
        fi
        ;;
    *)
        usage
        exit 1
        ;;
esac
