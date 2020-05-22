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


#### nodes with a single connection

- where the node receives an IPv4 address that is globally routed

That node will boot normally, but you'll also need to consider that the ndmz interface wil __also__ need an IPv4 address. In such an environment a node will need 2 IPv4 addresses. There is a reason for that: as we build wireguard interfaces we need to be sure that the destination port will never be occupied by some NAT conntrack, and as that is in a constant flux, it would be difficult to be sure the port  is unused.

**ERGO**: 2 IPv4 addresses per node, can be dynamic or static, but try to make the leases as static as possible

For IPv6, if your router provides for a Router Advertisment for an IPv6 prefix, you're all set, the node will pick it up and set up it's ndmz with another. For IPv6 no worries about the number of addresses in your prefix allocation, a `/64` (smallest entity in IPv6) can have 2^64 addresses... go ahead, grab your calculator and have a look, I'll wait.

If you run a segment with only IPv6, hop! you're all set, but with the important distinction that your node will only be reachable over IPv6 directly. So to expose workloads for IPv4 clients, you'll need to have access to tcp proxies that bridge the deep schism between IPv4 and IPv6.

- where a node is dual-stacked, has only private-space (rfc1918) IPv4 and globally routed IPv6

Seen the scarcity of IPv4, one might opt to do solely IPv6, but still have a node in the rack some private IPv4 for management; basically the same applies as an IPv6-only segment, with the difference that you ... meh, there is no real difference...


- multi-homed nodes 

Ok, in DC's farmers tend to listen to enterprise guys and their penchant for High-Availability. While we're convinced that these kinds of setup create more problems than they solve, bonds are supported, so if you have 2 switches with MLAG, you can request a bond for the workloads you define. (tbd)



## IPv6 integration for nodes that have only IPv4

There are a few use cases:

- user networks
- single nodes in a home setting
- multiple nodes in a farm on the same segment

We __should__ really be convinced that NAT ins an IPv6 space is definitely not done. Hence, the existing solution for assigning an ula prefix from the farmer_id and Network resources prefixes from the node_id, and then dual-nat the IPv6 __still__ needs IPv6 in the `ndmz` namespace.

So: what if we start to manage IPv6 prefixes __again__ in the Explorer, where a farmer has an IP-transit agreement with a farmer that __has__ IPv6.

That way, we can use all the code we already have for WG meshes, give __real__ ipv6 to a farmer with fully hidden nodes.

There are some implementation details though:

- single node, fully hidden in an IPv4 RFC1918 network:
nothing needs to be done, once it has a peer, the transit farmer has merely to route the packets, and on the single node, we don't have to run an RA, as it's only the node itself.  
The ULA prefix of the NR will be able to connect local 0-DBs, and in case the 0-DB is remote, the ULA address of the NR will be NATed over the Global addr on the WG interface to the transit farmer

- multiple nodes, fully hidden in an IPv4 RFC1918 network:
can be two(three)fold :
  - handle them each as a single node
  - make one node a router and RA and route a /64 prefix that is requested from the Explorer.  

  - Alternatively, we can __always__ request a /64 from the explorer in the transit farmer's allocation, and have it handy in case another node gets added.
  For that, workloads can then have a standard route from their ULA to that Global IPv6, where the 0-DBs won't need an ULA address any more.
  Given the ease for receiving allocations, a transit framer can easily request a `/48 /44 /40` by merely asking for them. Which enven with a `/48` , the transit farmer can hand out 16K prefixes, with a `/40` that number would be `2^24 = 16M` (million)

- user networks, IPv4 and IPv6
  Right now, the IPv6 ULA address that gets assigned to a container is only needed to reach the local 0-DB in case of hidden nodes, or, in case of publicly reachable 0-DB to reach all of them.  
  This is both unconvenient and a limitation, as a workload on a hidden node can only use local 0-DBs.  
  We could re-introduce the same principle of exitpoints in User Networks, and give each NR a valid prefix (`/64`), or even a subnet of that prefix. Keeping IPv4 as it is, while having routable IPv6 in the NRs.
  With that setup, an exitpoint is merely a diode firewall, unless in a later phase we would want to add filters to allow certain traffic into that User Network.

A **BIG** word of warning with the actual IPv4 setup, though, is that a node in a home network (or a private DC) has virtually full access to that network. A User NR, by means of the double NAT that is in place, can snoop (or at least nmap) the whole network in which it lives. We'll have to add a proper rule in NDMZ that there is only forwarding to the default gateway possible.

## Nodes that have only IPv4

Ok, let's talk about Zerotier, the panacea, 42, deus ex machina, what it is, what it doesn't and why we think it's not a great fit.

- the zerotier infrastructure is an overlay **switch**. There, I said it. L2. While that seems the very best idea, L2 talks. __A lot__. Certainly Zerotier, as it's penchant to try to discover peers, it crafts it's own packets, and sends them out on any interface it can find. whether or not that interface has a default route, or even if that crafted packet is sent out from the same ip on that interface. To alleviate that a little, you need to install a whole bunch of drop routes, enable martian filtering and even then... these packets get still generated and are effectively put on the nic. A part from poisioning neighbour tables and arp caches, these things might be ok in single-nic nodes, but with multiple nics, it's really a whole load of cruft on __all__ your networks.
all the to do some nat hole punching which is notoriously unreliable and a kludge.
- there are 6 places that are intermediate hosts, they have moves 3 times already (hosted -> gcp -> back to hosted). These sites are the ultimate packet forwarders when peers are not directly reachable (60% of the cases) and are __sslllooowwww__ at best. This is logical because these nodes carry virtually __all__ networks and need to do encryption for all of them. So yeah, running our own moons might help, but then we need to force all zt clients to orbit these moons too. The more, we never know a farmers network, how it's connected and when there is some dual nat, the zt won't work. Like in : will not.
- Zerotier is not TF. Any changes to policy, suddenly making their service payable per network or per client and we're hosed. Although they say we could run our own, they want $$ for it.
- 2.0 is still way off.
- the magical ad-hoc networks are accidents waiting to happen. Also
