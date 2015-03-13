Socketplane Clustering
======================

Socketplane comes default with zeroconf mechanism to discover other socketplane hosts in the network using Bonjour.
Bonjour is a mDNS based discovery protocol which requires Multicast-enabled network.

This requirement is addressed in networks that supports multicast (such as VirtualBox network, enterprise LAN).
But in many cloud deployments, Multicast is disabled by default. Hence the automatic host discovery using Bonjour will fail.

In order to address this problem, we added an experimental Static Clustering feature where the user must add 
atleast 1 peer node to particpate in the cluster.

Please note that this is experimental and a lot of work pending in making it usable (save config, restartability, etc.)

### 1. Bind to a network interface on the first node
  make sure that the network interface that is bound to has an ip-address that is reachable by the peers.
(For example : Don't bind to a VirtualBox NAT interface. Rather add a Bridged or Internal port and bind to that).

```bash
      socketplane cluster bind eth1
```
### 2. Join the Cluster on other nodes
```bash
      socketplane cluster join <Peer ip-address>
```
  For example, if the ip-address of Host1's eth1 interface is 1.1.1.1, then the join command on Host2 would look like
```bash
      socketplane cluster join 1.1.1.1
```
