package table

// FIBEntry Represents a FIB Entry
type FIBEntry struct {
	Prefix   string
	Nexthops []uint32
}

// FIB Holds the FIB
type FIB struct {
	Entries []FIBEntry
}
