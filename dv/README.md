# Named Data Networking Distance Vector Router

ndn-dv is a router based on the distance vector algorithm for [Named Data Networking](https://named-data.net) written in Go.
It is compatible with existing NDN applications and protocols developed for the [NFD](https://github.com/named-data/NFD) forwarder.

The specification of the ndn-dv protocol can be found in [SPEC.md](./SPEC.md)

## Usage

A sample configuration file is provided in [dv.sample.yml](./dv.sample.yml)

```bash
ndn-dv /etc/ndn/dv.yml
```

## Building from source

ndn-dv requires [Go 1.23](https://go.dev/doc/install) or later.

```bash
CGO_ENABLED=0 go build -o ndn-dv cmd/dv/main.go
```

## Publications

- Varun Patil, Sirapop Theeranantachai, Beichuan Zhang, Lixia Zhang. 2024. [Poster: Distance Vector Routing for Named Data Networking](https://dl.acm.org/doi/abs/10.1145/3680121.3699885).
  In Proceedings of the 20th International Conference on emerging Networking EXperiments and Technologies