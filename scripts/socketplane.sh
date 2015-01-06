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
    $0 <options> <command> [command_options] [arguments...]

OPTIONS:

    -D      Debug

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

    agent {stop|start|restart|logs}
            Start/Stop/Restart the SocketPlane container or show its logs

    info [container_id]
            Show SocketPlane info for all containers, or for a given container_id

    run [-n foo] <docker_run_args>
            Run a container and optionally specify which network to attach to

    start <container_id>
            Start a <container_id>

    stop <container_id>
            Stop the <container_id>

    rm <container_id>
            Remove the <container_id>

    attach <container_id>
            Attach to the <container_id>

    network list
            List all created networks

    network info <name>
            Display information about a given network

    network create <name> [cidr]
            Create a network

    network delete <name> [cidr]
            Delete a network

    network agent start
            Starts an existing SocketPlane image if it is not already running

    network agent stop
            Stops a running SocketPlane image. This will not delete the local image

EOF
}

basedir=$(dirname $(readlink -m $0))
. $basedir/functions.sh

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
        DOCKER_SVER=$(docker version | grep "Server version:" |  awk '{ print $3 }')
        DOCKER_CVER=$(docker version | grep "Client API version:" |  awk '{ print $4 }')
    fi

    OVS_SVER="NOT_INSTALLED"
    if command_exists ovsdb-server && command_exists ovs-vswitchd ; then
        OVS_SVER=$(ovs-appctl -V | grep "ovs-" |  awk '{ print $4 }')
    fi
}

deps() {
    echo "Socketplane  Docker Host Requirements:"
    echo ".. Open vSwitch Environment:"
    echo ".... Archicture:              amd64 or i386"
    echo "....   Current:               $ARCH"
    echo ".... Operating System:         Ubuntu, Debian and Fedora"
    echo "....   Current:               $lsb_dist"
    echo ".... Open vSwitch Version:     2.1 or higher"
    echo "....   Current:               $OVS_SVER"
    echo ".. Docker Environment:"
    echo ".... Docker Server Version:   1.4 or higher"
    echo "....   Current:               $DOCKER_SVER"
    echo ".... Docker Client Version:   1.16 or higher"
    echo "....   Current:               $DOCKER_SVER"
}

kernel_opts(){
    log_step "Setting Linux Kernel Options"
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
    sysctl -p | indent
}

pkg_update() {
    log_step "Ensuring Package Repositories are up to date"
    $pkg update > /dev/null
}

install_curl() {
    if command_exists curl; then
        log_step "Curl already installed!"
    else
        log_step "Installing Curl"
        $pkg install curl | indent
    fi
}

install_ovs() {
    if command_exists ovsdb-server && command_exists ovs-vswitchd ; then
        log_step "Open vSwitch already installed!"
    else
        if ! command getenforce  2>/dev/null || [[ $(getenforce) =~ Enforcing|Permissive ]] ; then
        log_step "Checking Open vSwitch dependencies.."
        $pkg install policycoreutils-python
        sudo semodule -d openvswitch  2>/dev/null || true
        fi
        log_step "Installing Open vSwitch.."
        $pkg install $ovs | indent
    fi

    # Make sure the processes are started
    case "$lsb_dist" in
        debian|ubuntu)
            if $(service openvswitch-switch status | grep "stop"); then
                service openvswitch-switch start
            fi
            ;;
        fedora)
            systemctl start openvswitch.service
            ;;
    esac

    sleep 1
    log_step "Setting OVSDB Listener"
    ovs-vsctl set-manager ptcp:6640
}

remove_ovs() {
    log_step "Removing existing Open vSwitch packages..."
    $pkg remove $ovs
}

install_docker() {
    if command_exists docker; then
        log_step "Docker already installed!"
    else
        log_step "Installing Docker..."
        case $lsb_dist in
            fedora)
                $pkg remove docker > /dev/null
                ;;
        esac

        if command_exists curl; then
            curl -sSL https://get.docker.com/ | sh
        elif command_exists wget; then
            wget -qO- https://get.docker.com/ | sh
        fi
    fi

    case $lsb_dist in
        debian|ubuntu)
            if [ -f /etc/init.d/docker ]; then
                if $(service docker status | grep "stop"); then
                    service docker start
                fi
            else
                if $(service docker.io status | grep "stop"); then
                    service docker.io start
                fi
            fi
            ;;
        fedora)
            systemctl start docker.service
            ;;
    esac
}

start_socketplane() {
    log_step "Starting the SocketPlane container"

    if [ -n "$(docker ps | grep socketplane/socketplane | awk '{ print $1 }')" ]; then
        log_fatal "A SocketPlane container is already running"
        return 1
    fi

    flags="--iface=auto"

    if [ "$1" = "unattended" ]; then
        [ -z $BOOTSTRAP ] && log_fatal "BOOTSTRAP not set" && exit 1

        if [ "$BOOTSTRAP" = "true" ] ; then
            flags="$flags --bootstrap=true"
        fi
    else
        while true; do
            read -p "Is this the first node in the cluster? (y/n)" yn
            case $yn in
                [Yy]* )
                    flags="$flags --bootstrap=true"
                    break
                    ;;
                [Nn]* )
                    break
                    ;;
                * )
                    echo "Please answer yes or no."
                    ;;
            esac
        done
    fi

    if [ "$DEBUG" = "true" ]; then
        flags="$flags --debug=true"
    fi

    cid=$(docker run -itd --privileged=true --net=host socketplane/socketplane socketplane $flags)

    if [ -n "$cid" ]; then
        log_info "A SocketPlane container was started" | indent
    else
        log_fatal "Error starting the SocketPlane container"
        exit 1
    fi

    mkdir -p /var/run/socketplane
    echo $cid > /var/run/socketplane/cid
}

start_socketplane_image() {
    log_step "Starting SocketPlane Agent"
    if ! command_exists docker; then
        log_fatal "Docker is not installed, please run \"./socketplane install\""
        exit 1
    fi

    for IMAGE_ID in $(docker ps -a | grep socketplane/socketplane | awk '{ print $1; }'); do
        log_info "Starting container $IMAGE_ID" | indent
        docker start ${IMAGE_ID} > /dev/null
    done

    if [ -n "$(docker ps | grep socketplane/socketplane | awk '{ print $1 }')" ]; then
        log_info  "All Socketplane agent containers are started."
    else
        log_info  "No Socketplane agent containers are started."
    fi
}

stop_socketplane_image() {
    log_step "Stopping SocketPlane Agent"
    if ! command_exists docker; then
        log_fatal "Docker is not installed, please run \"./socketplane install\""
        exit 1
    fi

    for IMAGE_ID in $(docker ps | grep socketplane/socketplane | awk '{ print $1; }'); do
        log_info "Stopping the SocketPlane container $IMAGE_ID" | indent
        docker stop ${IMAGE_ID} > /dev/null
    done

    if [ -z $(docker ps | grep socketplane/socketplane | awk '{ print $1 }') ]; then
        log_info "All Socketplane agent containers are stopped. Please run \"socketplane agent start\" to start them again"
    fi
}

stop_socketplane() {
    log_step "Stopping SocketPlane Agent"
    if ! command_exists docker; then
        log_fatal "Docker is not installed"
        exit 1
    fi

    for IMAGE_ID in $(docker ps -a | grep socketplane/socketplane | awk '{ print $1; }'); do
        log_info "Stopping socketplane container: $IMAGE_ID" | indent
        docker stop $IMAGE_ID > /dev/null
        sleep 1
        log_info "Removing socketplane container: $IMAGE_ID" | indent
        docker rm $IMAGE_ID > /dev/null
    done
    log_info "SocketPlane container deleted" | indent
}

remove_socketplane() {
    log_step "Removing the SocketPlane container image"
    if ! command_exists docker; then
        log_fatal "Docker is not installed"
        exit 1
    fi

    for IMAGE_ID in $(docker images | grep socketplane/socketplane | awk '{ print $1; }'); do
        log_info "Removing socketplane image: $IMAGE_ID" | indent
        docker rmi $IMAGE_ID > /dev/null
    done
}

logs() {
    if [ ! -f /var/run/socketplane/cid ] || [ -z $(cat /var/run/socketplane/cid) ]; then
        log_fatal "SocketPlane container is not running"
        exit 1
    fi
    docker logs $@ $(cat /var/run/socketplane/cid)
}

info() {
    if [ -z "$1" ]; then
        curl -s -X GET http://localhost:6675/v0.1/connections | python -m json.tool
    else
        containerId=$(docker ps -a --no-trunc=true | grep $1 | awk {' print $1'})
        if [ -z "$containerId" ]; then
            log_fatal "Could not find a Container with Id : $1"
        else
            curl -s -X GET http://localhost:6675/v0.1/connections/$containerId | python -m json.tool
        fi
    fi
}

container_run() {
    network=""
    if [ $1 = '-n' ]; then
        network=$2
        shift 2
    fi

    attach="false"
    if [ -z "$(echo "$@" | grep -e '-[a-zA-Z]*d[a-zA-Z]*\s')" ]; then
        attach="true"
    fi

    if [ "$attach" = "false" ]; then
         cid=$(docker run --net=none $@)
    else
         cid=$(docker run --net=none -d $@)
    fi

    cPid=$(docker inspect --format='{{ .State.Pid }}' $cid)
    cName=$(docker inspect --format='{{ .Name }}' $cid)

    json=$(curl -s -X POST http://localhost:6675/v0.1/connections -d "{ \"container_id\": \"$cid\", \"container_name\": \"$cName\", \"container_pid\": \"$cPid\", \"network\": \"$network\" }")
    result=$(echo $json | sed 's/[,{}]/\n/g' | sed 's/^".*":"\(.*\)"/\1/g' | awk -v RS="" '{ print $7, $8, $9, $10, $11 }')

    attach $result $cPid

    if [ "$attach" = "false" ]; then
        echo $cid
    else
        docker attach $cid
    fi
}

container_stop() {
    docker stop $@
    find -L /var/run/netns -type l -delete
}

container_start() {
    docker start $1 > /dev/null
    cid=$(docker ps --no-trunc=true | grep $1 | awk {' print $1'})
    cPid=$(docker inspect --format='{{ .State.Pid }}' $cid)
    cName=$(docker inspect --format='{{ .Name }}' $cid)

    json=$(curl -s -X GET http://localhost:6675/v0.1/connections/$cid)
    result=$(echo $json | sed 's/[,{}]/\n/g' | sed 's/^".*":"\(.*\)"/\1/g' | awk -v RS="" '{ print $7, $8, $9, $10, $11 }')

    attach $result $cPid

    # ToDo: PUT new container info to the api

    echo $cid

}

container_delete() {
    cid=$(docker ps -a --no-trunc=true | grep $1 | awk {' print $1'})
    docker rm $@
    curl -s -X DELETE http://localhost:6675/v0.1/connections/$cid
    sleep 1
    # clean up dangling symlinks
    find -L /var/run/netns -type l -delete
}

network_list() {
    curl -s -X GET http://localhost:6675/v0.1/networks | python -m json.tool
}

network_info() {
    curl -s -X GET http://localhost:6675/v0.1/networks/$1| python -m json.tool
}

network_create() #name
                 #cidr
{
    #ToDo: Check CIDR is valid
    curl -s -X POST http://localhost:6675/v0.1/networks -d "{ \"id\": \"$1\", \"subnet\": \"$2\" }" | python -m json.tool

}

network_delete() {
    curl -s -X DELETE http://localhost:6675/v0.1/networks/$@
}

# Run as root only
if [ "$(id -u)" != "0" ]; then
    log_fatal "Please run as root"
    exit 1
fi

# perform some very rudimentary platform detection
lsb_dist=''
if command_exists lsb_release; then
    lsb_dist="$(lsb_release -si)"
fi
if [ -z "$lsb_dist" ] && [ -r /etc/lsb-release ]; then
    lsb_dist="$(. /etc/lsb-release && echo "$DISTRIB_ID")"
fi
if [ -z "$lsb_dist" ] && [ -r /etc/debian_version ]; then
    lsb_dist='debian'
fi
if [ -z "$lsb_dist" ] && [ -r /etc/fedora-release ]; then
    lsb_dist='fedora'
fi
if [ -z "$lsb_dist" ] && [ -r /etc/os-release ]; then
    lsb_dist="$(. /etc/os-release && echo "$ID")"
fi

lsb_dist="$(echo "$lsb_dist" | tr '[:upper:]' '[:lower:]')"

if [ -z lsb_dist ]; then
    log_fatal "Operating System could not be detected"
    exit 1
fi

if [ -z "$(echo "$lsb_dist" | grep -E 'ubuntu|debian|fedora')" ]; then
    log_fatal "Operating System $lsb_dist is not yet supported. Please contact support@socketplane.io"
    exit 1
fi

case "$lsb_dist" in
    debian|ubuntu)
        pkg="apt-get -q -y"
        ovs="openvswitch-switch"
        ;;
    fedora)
        pkg="yum -q -y"
        ovs="openvswitch"
        ;;
esac

while getopts ":D" opt; do
  case $opt in
    D)
      DEBUG="true"
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      ;;
  esac
done
shift $((OPTIND-1))

if [ "$DEBUG" = "true" ]; then
    set -x
fi

case "$1" in
    help)
        usage
        ;;
    install)
        shift
        log_notice "Installing SocketPlane..."
        kernel_opts
        #pkg_update
        install_curl
        install_ovs
        install_docker
        start_socketplane $@
        log_notice "Done!!!"
        ;;
    uninstall)
        log_notice "Uninstalling SocketPlane..."
        stop_socketplane
        remove_socketplane
        log_notice "Done!!!"
        ;;
    clean)
        log_notice "Removing SocketPlane and all its dependencies..."
        get_status
        remove_ovs
        stop_socketplane
        remove_socketplane
        log_notice "Done!!!"
        ;;
    deps)
        get_status
        deps
        ;;
    info)
        shift
        info $@
        ;;
    run)
        shift
        container_run $@
        ;;
    stop)
        shift
        container_stop $@
        ;;
    start)
        shift
        container_start $@
        ;;
    rm)
        shift
        container_delete $@
        ;;
    attach)
        shift
        docker attach $@
        ;;
    network)
        shift
        case "$1" in
            list)
                network_list
                ;;
            info)
                shift
                network_info $@
                ;;
            create)
                shift
                network_create $@
                ;;
            delete)
                shift
                network_delete $@
                ;;
            *)
                log_fatal "Unknown Command"
                usage
                exit 1
        esac
        ;;
    agent)
        shift 1
        case "$1" in
            start)
                start_socketplane_image
                ;;
            stop)
                stop_socketplane_image
                ;;
            restart)
                stop_socketplane_image
                start_socketplane_image
                ;;
            logs)
                shift 1
                logs $@
                ;;
            *)
                log_fatal "\"socketplane agent\" {stop|start|restart|logs}"
                exit 1
                ;;
        esac
        ;;
    *)
        usage
        exit 1
        ;;
esac
