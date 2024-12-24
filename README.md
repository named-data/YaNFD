<div align="center">
  <a href="https://named-data.net/">
    <img alt height="65" src="logo.svg"/>
  </a>

  <h1> Named Data Networking Daemon </h1>
</div>

[![build](https://github.com/named-data/ndnd/actions/workflows/build.yml/badge.svg)](https://github.com/named-data/ndnd/actions/workflows/build.yml)
[![test](https://github.com/named-data/ndnd/actions/workflows/test.yml/badge.svg)](https://github.com/named-data/ndnd/actions/workflows/test.yml)
[![MIT license](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)

NDNd is a Golang implementation of the Named Data Networking (NDN) [protocol](https://named-data.net) stack.

See the project [overview](https://named-data.net/project/), architecture [details](https://named-data.net/project/archoverview/) and the [tutorial](https://101.named-data.net/) for more info on NDN.

## 🏗️ Building the source

NDNd is written in pure Go and requires [Go 1.23](https://go.dev/doc/install) or later.

Once Go is installed, run `make` to build the `ndnd` executable, followed by `make install` to install it globally.

# 🌟 Usage

NDNd provides several independent modules that can be used separately or together.

You can use the `ndnd` CLI to list available modules and get more info on their usage.

```text
root@0037b98ec2ac:~# ndnd
  _   _ ____  _   _     _
 | \ | |  _ \| \ | | __| |
 |  \| | | | |  \| |/ _  |
 | |\  | |_| | |\  | (_| |
 |_| \_|____/|_| \_|\____|

Named Data Networking Daemon (ndnd)

Usage: ndnd [command]
  fw              NDN Forwarding Daemon
  dv              NDN Distance Vector Routing Daemon

  ping            Send Interests to an NDN ping server
  pingserver      Start an NDN ping server under a prefix
  cat             Retrieve data under a prefix
  put             Publish data under a prefix
```

## 🔀 Network Forwarder

The `ndnd/fw` package implements YaNFD, a packet forwarder for the NDN platform.
It is compatible with the management tools and protocols developed for the [NFD](https://github.com/named-data/NFD) forwarder.

To start the forwarder locally, run the following:

```bash
ndnd fw start yanfd.config.yml
```

A full configuration example can be found in [fw/yanfd.sample.yml](fw/yanfd.sample.yml).
Note that the default configuration may require root privileges to bind to multicast interfaces.

## 📡 Distance Vector Router

The `ndnd/dv` package implements `ndn-dv`, an NDN Distance Vector routing daemon.

To start the routing daemon bound to the local forwarder, run the following:

```bash
ndnd dv start dv.config.yml
```

A full configuration example can be found in [dv/dv.sample.yml](dv/dv.sample.yml).
Make sure the network and router name are correctly configured and the forwarder is running.

## 📚 Standard Library

The `ndnd/std` package implements `go-ndn`, a standard library for NDN applications.

You can use this package to build your own NDN applications.
Several examples are provided in the [std/examples](std/examples) directory.

## 🛠️ Tools

The `ndnd/tools` package provides basic utilities for NDN networks.
These can be used directly using the `ndnd` CLI.

- `ping`/`pingserver`: test reachability between two NDN nodes
- `cat`/`put`: segmented file transfer between a consumer and a producer

# Contributing & License

Contributions to NDNd are greatly appreciated and can be made through GitHub pull requests and issues.

NDNd is free software distributed under the permissive [MIT license](LICENSE).
