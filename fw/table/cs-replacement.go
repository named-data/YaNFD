/* YaNFD - Yet another NDN Forwarding Daemon
 *
 * Copyright (C) 2020-2021 Eric Newberry.
 *
 * This file is licensed under the terms of the MIT License, as found in LICENSE.md.
 */

package table

import "github.com/named-data/YaNFD/ndn_defn"

// CsReplacementPolicy represents a cache replacement policy for the Content Store.
type CsReplacementPolicy interface {
	// AfterInsert is called after a new entry is inserted into the Content Store.
	AfterInsert(index uint64, data *ndn_defn.PendingPacket)

	// AfterRefresh is called after a new data packet refreshes an existing entry in the Content Store.
	AfterRefresh(index uint64, data *ndn_defn.PendingPacket)

	// BeforeErase is called before an entry is erased from the Content Store through management.
	BeforeErase(index uint64, data *ndn_defn.PendingPacket)

	// BeforeUse is called before an entry in the Content Store is used to satisfy a pending Interest.
	BeforeUse(index uint64, data *ndn_defn.PendingPacket)

	// EvictEntries is called to instruct the policy to evict enough entries to reduce
	// the Content Store size below its size limit.
	EvictEntries()
}
