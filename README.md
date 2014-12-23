# SocketPlane

Developers don't want to care about VLANs, VXLANs, Tunnels or TEPs. People responsible for managing the infra expect it to be performant and reliable. SocketPlane provides a networking abstraction at the socket-layer in order to solve the problems of the network in a manageable fashion.

## SocketPlane Technology Preview

This early release is just a peek at some of the things we are working on and releasing to the community as open source. In order to manipulate Docker we needed to use a temporary wrapper until the upstream Docker work for drivers and plugins are complete. There is a `socketplane` command that is used as a front-end to the `docker` CLI commands. This is what enables us send hooks to the SocketPlane Daemon. 

In this release we support the following features:

- ZeroConf multi-host networking for Docker
- Elastic growth of a Docker/SocketPlane cluster
- Support for multiple networks
- Distributed IP Address Management (IPAM)

Our 'ZeroConf' technology is based on [multicast DNS](). This allows us to discover other SocketPlane cluster members on the same segment and to start peering with them. This allows us to elastically grow the cluster on demand by simply deploying another host - mDNS handles the rest. Since multicast availability is hit and miss in most networks, it is aimed at making it easy to deploy Docker and SocketPlane to start getting familiar with the exciting marriage of advanced, yet sane networking scenario with the exciting Docker use cases.

Once we've discovered our neighbors, we're able to join an embedded [Consul] instance, giving us access to an eventually consistent key/value store for network state.

We support mutiple networks, to allow you to divide your containers in to subnets to ease the burden of enforcing firewall policy in the network.

Finally, we've implemented a distributed IP address management solution that enables non conflicting address assignment throught a cluster.


> Note: As we previously mentioned, it's not an *ideal* approach, but it allows people to start kicking the tyres as soon as possible. All of the functionality in `socketplane.sh` will move in to our Golang core over time.

## Deploy
While Golang, Docker and OVS can run on many operating systems, we are currently running tests and QA against [Ubuntu](http://www.ubuntu.com/download) and [Fedora](https://getfedora.org/). 

    curl -sSL https://get.socketplane.io/ | sh

or

    sudo wget -qO- https://get.socketplane.io/ | sh

Next start an image, for example a bash shell:

    sudo ./socketplane.sh run -i -t ubuntu /bin/bash

## Vagrant Installation
A Default Vagrant file has been provided to setup a three VM demo system.  By default three Ubuntu 14.04 VM hosts will be installed each with an installed version of Socketplane.  

To start the demo systems:

    git clone https://github.com/socketplane/socketplane
    cd socketplane
    vagrant up  

This will create three hosts (socketplane-1, socketplane-2 and socketplane-3).  Once the VM's are started you can ssh in as follows:

    vagrant ssh socketplane-1 
    vagrant ssh socketplane-2 
    vagrant ssh socketplane-3 

You can start Docker instances in each of the VM's and they will all be in a default network.

    sudo ocketplane run -itd ubuntu

You can also see the status of instances on a specific host VM by typing:

    sudo socketplane info 

If you want to create multiple networks you can do the following:

    sudo socketplane network create web 10.2.0.0/16

    sudo socketplane run -n web -itd ubuntu

You can list all the created networks with the following command:

    sudo socketplane network list 

For more options use the HELP command

    sudo socketplane help

## Hacking
Clone the repository:

    git clone https://github.com/socketplane/socketplane

or
    
    git clone git@github.com:<username>/socketplane.git

Once cloned:

    sudo cd socketplane
    sudo ./scripts socketplane install
    sudo ./scripts socketplane help

## Contact us

For bugs please file an [issue](https://github.com/socketplane/socketplane/issues). For any assistance, questions or just to say hi, please visit us on IRC, `#socketplane` at `irc.freenode.net`

Stay tuned for some exciting features coming soon from the SocketPlane team.

## Copyright and license

Code and content copyright 2014 SocketPlane, inc. Code is released under the Apache 2.0 license. 
