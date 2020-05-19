# Zos Network initialization

For a node to boot, it needs connectivity to perform it's second-stage boot (i.e. download latest versions of the mamnagement daemons and continue booting)

## Internet

For that `internet` is a one-time boot process that finishes by registering a lease refreshing dhcp client in the zinit process manager (PID1)

```pseudo

interfacelist = do in parallel (
    for interface in interfaces:
        create namespace 
        send interface in namespace
        run dhcp client in namespace
        if interface received ip[46]:
            if ip has default gateway:
                add interface to list
)
if no default gateway on any interface:
    restart 
if interfacelist > 1
    search for first default gw with RFC1918 address and/or ULA prefix
        add interface to newly created zos bridge
        run dhcp client on zos and register it in zinit
    if no rfc1918  addr
        use first public IPv4 address

exit
```

the IPv6 part of above will mostly configure itself

For this to work, zos needs to live an a network with a default gateway, be it IPv4 or IPv6, with al least one interface.

The idea is that we want to zos to continue booting when any type of connectivity is established, be it directly connected public IPv4 or IPv6, or even RFC1918 or ULA addresses with NAT.

## Networkd

Once the node has booted and registered itself to the grid, `networkd` continues the network setup.

what it does:

  - it sets up an `ndmz` routing/nat virtual router that will be responsible to give public unidirectional connectivity, meaning that future network resources of user networks can communicate with the internet.  
  That means that workloads that need to initiate a connection to the Internet will reach the Internet through NAT, be it for IPv4 as for IPv6.

  - `networkd` will loop while trying to receive an ip address in `ndmz`, as long as it receives at least one IPv6 address, but will also run a dhcp client for an IPv4 address, although it will not error if it doesn't receive any. (that still means that IPv6 __is__ required for `ndmz` to finish it's setup)

  - once `ndmz` has an IP address , default NAT rules are put in place for future Network resources to reach the Internet.

## 0-db 

Apart from some pitfalls with the actual network features (ipv6 required, hidden node woes), also 0-db is only reachable over IPv6 for consumer workloads, and as such, can only be used with ipv6.

there are other pitfalls in 0-db, like the fact that security is here also just an afterthought, no ssl, always public for consumer workloads, and most of all, compaction of 0-db for fast-changing data is still a hairy and unsolved problem. 

# Zos network environment setup

There are a few ways to set up your network such that it can host nodes to bear user workloads.

A lot of times, farmers will have a difficult time to go through the administrative and technical hurdles to get a network properly set up for a node or group of nodes.
This as well on the physical setup (switches,routers, network cards to use,...) as on the level of IP transit.
Also, with the scarceness of IPv4, requests for an IPv4 subnet to be routed to the farm will be cumbersome, well documented and  well argumented to as well the ISP of the farmer and/or the RIR (Regional Internet Registrar, like RIPE, ARIN, APNIC, AFRINIC...)
Point is that a network is not ubiquitous, until it is, because it's set up. But all the steps to get there can be cumbersome and need a lot of attention/actions.

For IPv6, getting an allocation that is routed and handeled with an ISP, things tend to get less and less difficult, but the Internet is going to need a big nudge from the users to force their ISP to get that IPv6 thing going.

Anyways it is what it is, but we need to get as many ISPs on board for IPv6. We had already a victory with NTT Astria that implemented IPv6 because we insisted. 

### Small networks / Home networks

  - node is connected in a private setting, wit only an IPv4 nat router downstream
  - node is connected in a private IPv4 setting, but received an IPv6 address from an RA
    - that IPv6 address is fully reachable
    - that IPv6 address is firewalled (router allows only outgoing traffic) 

### Farmers in DCs

Mostly multiple nodes: ISP provides for transit 

In DCs, in Europe and in the US/CA, most ISPs will be able to provide for IPv6, if pressured enough. We don't know what's going on in Sweden, but on average, there seems no really big friction any more to request and obtain an ipv6 allocation.
Mostly a farmer can ask for a `/48` without too much hassle.

For IPv4, things get a lot hairier, and receiving a subnet of 30-62 IPv4 addresses will be met with a lot of pushback, and will require a lot of justification to the ISP for obtaining the needed IPv4 addresses (RIPE, for instance, has no consecutive blocks of 1024 addresses free any more, and in a 4 billion pool that says a lot).

Either way, for a farmer to be able to host a proper cluster, some adminstration will be required to get some decent farm in terms of IP transit, that being with the ISP in the DC, or even directly with RIPE, or het local RIR/LIR (google for `internet registrar`).

Layouts: 

  - nodes with single connection (one NIC)
    - nodes are directly connected on a segment that contains public addreses  
      - IPV4 only
      - IPV6 only
      - Dual-stacked (geek for both ;-) )
    - nodes are behind an IPv4 NAT router, but have routed IPV6
    - nodes are behind an IPv4 NAT router, but without IPv6
  - nodes with multiple NICS, where the network is separated in OOB and workload traffic
  - nodes with multiple NICs, where the workload traffic is made highly available.
