module github.com/named-data/YaNFD

go 1.18

require (
	github.com/Link512/stealthpool v0.2.0
	github.com/apex/log v1.9.0
	github.com/cespare/xxhash v1.1.0
	github.com/cornelk/hashmap v1.0.1
	github.com/google/gopacket v1.1.19
	github.com/gorilla/websocket v1.5.0
	github.com/pelletier/go-toml v1.9.4
	github.com/stretchr/testify v1.7.1
	golang.org/x/exp v0.0.0-20220414153411-bcd21879b8fd
	golang.org/x/sys v0.0.0-20220406163625-3f8b81556e12
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dchest/siphash v1.2.3 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)

replace github.com/Link512/stealthpool => github.com/zjkmxy/stealthpool v0.2.1
