# Named Data Networking Daemon

NDNd is a Golang implementation of the Named Data Networking (NDN) [protocol](https://named-data.net) stack.

See the project [overview](https://named-data.net/project/), architecture [details](https://named-data.net/project/archoverview/) and the [tutorial](https://101.named-data.net/) for more info on NDN.

## Building the source

NDNd is written in pure Go and requires [Go 1.23](https://go.dev/doc/install) or later.

Once Go is installed, run `make` to build the `ndnd` executable.

## Modules

NDNd provides several independent modules that can be used separately or together.

### `ndnd/fw`

The `fw` package implements YaNFD, a packet forwarder for the NDN platform.
It is compatible with the management tools and protocols developed for the [NFD](https://github.com/named-data/NFD) forwarder.

To start the forwarder locally, run the following:

```bash
ndnd fw start yanfd.config.yml
```

A full configuration example can be found in [fw/yanfd.sample.yml](fw/yanfd.sample.yml).
Note that the default configuration may require root privileges to bind to multicast interfaces.

### `ndnd/dv`

The `dv` package implements `ndn-dv`, an NDN Distance Vector routing daemon.

To start the routing daemon bound to the local forwarder, run the following:

```bash
ndnd dv start dv.config.yml
```

A full configuration example can be found in [dv/dv.sample.yml](dv/dv.sample.yml).
Make sure the network and router name are correctly configured and the forwarder is running.

### `ndnd/std`

The `std` package implements `go-ndn`, a standard library for NDN applications.

You can use this package to build your own NDN applications.
Several examples are provided in the [std/examples](std/examples) directory.

### `ndnd/tools`

The `tools` package provides basic utilities for NDN networks.
These can be used directly using the `ndnd` CLI.

- `ping`/`pingserver`: test reachability between two NDN nodes
- `cat`/`put`: segmented file transfer between a consumer and a producer

## Contributing & License

Contributions to NDNd are greatly appreciated and can be made through GitHub pull requests and issues.

NDNd is free software distributed under the permissive [MIT license](LICENSE).
