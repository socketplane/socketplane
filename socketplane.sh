#!/bin/sh
# Temporary wrapper for OVS until native
# Docker integration is available

installation() {
    DIST=Unknown
    RELEASE=Unknown
    CODENAME=Unknown
    ARCH=`uname -m`
    if [ "$ARCH" = "x86_64" ]; then ARCH="amd64"; fi
    if [ "$ARCH" = "i686" ]; then ARCH="i386"; fi

    test -e /etc/debian_version && DIST="Debian"
    grep Ubuntu /etc/lsb-release &> /dev/null && DIST="Ubuntu"
    if [ "$DIST" = "Ubuntu" ] || [ "$DIST" = "Debian" ]; then
        install='sudo apt-get -y install'
        remove='sudo apt-get -y remove'
        pkginst='sudo dpkg -i'
        if ! which lsb_release &> /dev/null; then
            $install lsb-release
        fi
    fi
    test -e /etc/fedora-release && DIST="Fedora"
    if [ "$DIST" = "Fedora" ]; then
        install='sudo yum -y install'
        remove='sudo yum -y erase'
        pkginst='sudo rpm -ivh'
        # Prereqs for this script
        if ! which lsb_release &> /dev/null; then
            $install redhat-lsb-core
        fi
    fi
    if which lsb_release &> /dev/null; then
        DIST=`lsb_release -is`
        RELEASE=`lsb_release -rs`
        CODENAME=`lsb_release -cs`
    fi
    echo "Detected Linux distribution: $DIST $RELEASE $CODENAME $ARCH"
    # Kernel distribution check
    KERNEL_NAME=`uname -r`
    KERNEL_HEADERS=kernel-headers-${KERNEL_NAME}
    if ! echo $DIST | egrep 'Ubuntu|Debian|Fedora'; then
        echo "Install.sh currently only supports Ubuntu, Debian and Fedora."
        exit 1
    fi
    test -e /etc/debian_version && DIST="Debian"
    grep Ubuntu /etc/lsb-release &> /dev/null && DIST="Ubuntu"
    if [ "$DIST" = "Ubuntu" ] || [ "$DIST" = "Debian" ]; then
        install='sudo apt-get -y install'
        remove='sudo apt-get -y remove'
        pkginst='sudo dpkg -i'
        if ! which lsb_release &> /dev/null; then
            $install lsb-release
        fi
    fi
    test -e /etc/fedora-release && DIST="Fedora"
        if [ "$DIST" = "Fedora" ]; then
        install='sudo yum -y install'
        remove='sudo yum -y erase'
        pkginst='sudo rpm -ivh'
        if ! which lsb_release &> /dev/null; then
            $install redhat-lsb-core
        fi
    fi
    echo "Installing Open vSwitch"
    if [ "$DIST" == "Fedora" ]; then
        $install openvswitch
        return
    fi
    if ! dpkg --get-selections | grep openvswitch-datapath; then
        $install openvswitch-datapath-dkms
    fi
    $install openvswitch-switch
    if sudo service openvswitch-controller stop; then
        echo "Stopped running controller"
    fi
    if [ -e /etc/init.d/openvswitch-controller ]; then
        sudo update-rc.d openvswitch-controller disable
    fi
}

remove_ovs() {
    pkgs=`dpkg --get-selections | grep openvswitch | awk '{ print $1;}'`
    echo "Removing existing Open vSwitch packages:"
    echo $pkgs
    if ! $remove $pkgs; then
        echo "Not all packages removed correctly"
    fi
    # For some reason this doesn't happen
    if scripts=`ls /etc/init.d/*openvswitch* 2>/dev/null`; then
        echo $scripts
        for s in $scripts; do
            s=$(basename $s)
            echo SCRIPT $s
            sudo service $s stop
            sudo rm -f /etc/init.d/$s
            sudo update-rc.d -f $s remove
        done
    fi
    echo "Done removing OVS"
}

usage()
{
cat << EOF
usage: $0 options

Install and run SocketPlane

OPTIONS:
    -h      (H)elp and usage
    -b      (B)oom just go already
    -i      (I)nstall SocketPlane
    -b      (D)elete Socketplane installation and dependencies
EOF
}

# Parse CLI arguments
if [ $# -eq 0 ]
then
    usage
else
while getopts ":bhid" opt; do
	case ${opt} in
      b)    installation;;
      i)    installation;;
      d)    remove_ovs;;
      h)    usage;;
      ?)    usage;;
      esac
    done
    shift $(($OPTIND - 1))
fi

