package config

import "time"

const CostInfinity = uint64(16)
const MulticastStrategy = "/localhost/nfd/strategy/multicast"

type Config struct {
	// GlobalPrefix should be the same for all routers in the network.
	GlobalPrefix string
	// RouterPrefix should be unique for each router in the network.
	RouterPrefix string
	// Period of sending Advertisement Sync Interests.
	AdvertisementSyncInterval time.Duration
	// Time after which a neighbor is considered dead.
	RouterDeadInterval time.Duration
}
