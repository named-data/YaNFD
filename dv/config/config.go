package config

import (
	"errors"
	"time"

	enc "github.com/zjkmxy/go-ndn/pkg/encoding"
)

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

	// Parsed Global Prefix
	GlobalPfxN enc.Name
	// Parsed Router Prefix
	RouterPfxN enc.Name
	// Advertisement Sync Prefix
	AdvSyncPfxN enc.Name
	// Advertisement Data Prefix
	AdvDataPfxN enc.Name
	// Prefix Table Sync Prefix
	PfxSyncPfxN enc.Name
	// Prefix Table Data Prefix
	PfxDataPfxN enc.Name
}

func (c *Config) Parse() (err error) {
	// Validate prefixes not empty
	if c.GlobalPrefix == "" || c.RouterPrefix == "" {
		return errors.New("GlobalPrefix and RouterPrefix must be set")
	}

	// Parse prefixes
	c.GlobalPfxN, err = enc.NameFromStr(c.GlobalPrefix)
	if err != nil {
		return err
	}

	c.RouterPfxN, err = enc.NameFromStr(c.RouterPrefix)
	if err != nil {
		return err
	}

	// Validate intervals are not too short
	if c.AdvertisementSyncInterval < 1*time.Second {
		return errors.New("AdvertisementSyncInterval must be at least 1 second")
	}

	// Dead interval at least 2 sync intervals
	if c.RouterDeadInterval < 2*c.AdvertisementSyncInterval {
		return errors.New("RouterDeadInterval must be at least 2*AdvertisementSyncInterval")
	}

	// Create name table
	c.AdvSyncPfxN = append(c.GlobalPfxN,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADS"),
	)
	c.AdvDataPfxN = append(c.RouterPfxN,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "ADV"),
	)
	c.PfxSyncPfxN = append(c.GlobalPfxN,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "PFS"),
	)
	c.PfxDataPfxN = append(c.RouterPfxN,
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "DV"),
		enc.NewStringComponent(enc.TypeKeywordNameComponent, "PFX"),
	)

	return nil
}
