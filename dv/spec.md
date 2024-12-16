# ndn-dv specification

This page describes the protocol specification of NDN Distance Vector Routing (ndn-dv).

## 1. Basic Protocol Design

1. All routers must have a unique *name* in the network for identification,
   and routers should be able to mutually authenticate each other.

1. Every router maintains a Routing Information Base (RIB) and
   computes a single *Advertisement* every time the RIB changes.
   The Advertisement is synchronized with all *neighbors* using a
   router-specific State Vector Sync group (*Advertisement Sync* group).

1. All routers join a global *Prefix Sync* SVS group to synchronize the
   global prefix table, which contains the mapping of prefixes to
   routers that can reach them.

## 2. Format and Naming

- **Advertisement Sync group**: `/localhop/<network>/32=DV/32=ADS`
- **Advertisement Data**: `/localhop/<router>/32=DV/32=ADV/58=<seq>`
- **Prefix Sync group**: `/<network>/32=DV/32=PFS`
- **Prefix Data**: `/<router>/32=DV/32=PFX/58=<seq>`
- **Prefix Data snapshot**: `/<router>/32=DV/32=PFX/32=SNAP/58=<seq>`

`<router>` is the router's unique name in the network.\
`<network>` is the globally unique network prefix.

## 3. TLV Specification

```abnf
Advertisement = ADVERTISEMENT-TYPE TLV-LENGTH
                *AdvEntry

Interface = INTERFACE-TYPE TLV-LENGTH NonNegativeInteger
Neighbor = NEIGHBOR-TYPE TLV-LENGTH Name

AdvEntry = ADV-ENTRY-TYPE TLV-LENGTH
           Destination
           NextHop
           Cost
           OtherCost

Destination = DESTINATION-TYPE TLV-LENGTH Name
NextHop = NEXT-HOP-TYPE TLV-LENGTH Name
Cost = COST-TYPE TLV-LENGTH NonNegativeInteger
OtherCost = OTHER-COST-TYPE TLV-LENGTH NonNegativeInteger

ADVERTISEMENT-TYPE = 201
ADV-ENTRY-TYPE = 202
DESTINATION-TYPE = 204
NEXT-HOP-TYPE = 206
COST-TYPE = 208
OTHER-COST-TYPE = 210
```

```abnf
PrefixOpList = PREFIX-OP-LIST-TYPE TLV-LENGTH
               ExitRouter
               [*PrefixOpReset]
               [*PrefixOpAdd]
               [*PrefixOpRemove]

ExitRouter = DESTINATION-TYPE TLV-LENGTH Name
PrefixOpReset = PREFIX-OP-RESET-TYPE TLV-LENGTH
PrefixOpAdd = PREFIX-OP-ADD-TYPE TLV-LENGTH
              Name
              Cost
PrefixOpRemove = PREFIX-OP-REMOVE-TYPE TLV-LENGTH
                 Name

PREFIX-OP-LIST-TYPE = 301
PREFIX-OP-RESET-TYPE = 302
PREFIX-OP-ADD-TYPE = 304
PREFIX-OP-REMOVE-TYPE = 306
```

## 4. Protocol Operation

### A. RIB State

Each router maintains a list of RIB entries as the RIB state. Each RIB entry
contains the following fields:

1. `Destination`: name of the destination router.
1. `Cost (Interface)`: cost to reach destination through this interface (one for each interface).

### B. Advertisement Computation

A new advertisement is computed by the router whenever the RIB changes.

1. `Links` in the advertisement are populated with the router's interfaces.
1. `AdvEntries` are added to the advertisement based on the RIB state.

One `AdvEntry` is generated for each RIB entry and contains the following fields:

1. `Destination`: name of the destination router.
1. `Interface`: Interface identifier for reaching the destination with lowest cost.
1. `Cost`: Cost associated with the next-hop interface.
1. `OtherCost`: Cost associated with the *second-best* next-hop interface.

Notes:

1. If multiple next hops have the same cost, the router chooses the next hop with the lexicographically lowest name.
1. If the advertisement changes, the router increments the sequence number for the *Advertisement Sync* group.
1. (TODO) The sequence number is incremented periodically every 10 seconds.
1. (TODO) Neighbor is considered dead if no update is received for 3 periods.

### C. Update Processing

On receiving a new advertisement from a neighbor, the router processes the advertisement as follows:

```python
for n in neighbors:
  if n.advertisement is None:
    continue

  for entry in n.advertisement:
    cost = entry.cost + 1

    if entry.nexthop is self:
      if entry.other < INFINITY:
        cost = entry.other + 1
      else:
        cost = INFINITY

    if cost >= INFINITY:
      continue

    rib[entry.destination][n.interface] = cost
```

`INFINITY` is the maximum cost value, set to `16` by default.

### D. Prefix Sync

Each router maintains a global prefix table that maps prefixes to routers that can reach them.

1. When any router makes a change to their local prefix list, it increments the
   sequence number for the *Prefix Sync* group, and publishes a `PrefixOpList`
   message. The contents of the `PrefixOpList` must be processed strictly in order.

1. When a router starts, it sends a `PREFIX-OP-RESET` operation.
   This clears all prefix entries for the sender at all routers.

1. When a router adds a new prefix, it sends a `PREFIX-OP-ADD` operation.
   If the cost is updated, the router sends a `PREFIX-OP-ADD` with the new cost.

1. When a router removes a prefix, it sends a `PREFIX-OP-REMOVE` operation.

### E. FIB Computation

The FIB is configured based on the RIB state and the global prefix table.

1. For each prefix in the global prefix table, the router selects the lowest-cost
   next-hop interface from the RIB state and installs a FIB entry.

1. If the prefix is not reachable, any existing FIB entry is removed.

1. If the prefix is reachable through multiple interfaces, the router installs
   multiple FIB entries, one for each interface.

1. When a prefix destination has multiple exit routers, the router chooses the exit
   router that it can reach with the lowest cost.
