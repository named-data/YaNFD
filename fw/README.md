# YaNFD - Yet another Named Data Networking Forwarding Daemon

YaNFD is a forwarding daemon for the [Named Data Networking](https://named-data.net) platform written in Go.
It is compatible with existing NDN applications and utilizes the management tools and protocols developed for the [NFD](https://github.com/named-data/NFD) forwarder.

## Prerequisites

YaNFD requires [Go 1.23](https://go.dev/doc/install) or later.

YaNFD has been developed and tested on Linux.
However, we have designed it with support for Windows, macOS, and BSD in mind.
We have received reports that YaNFD operates properly on Windows and macOS; this has not been evaluated by the developers.

## Usage

Install the YaNFD binary with the following command:

```bash
go install github.com/pulsejet/ndnd/fw/cmd/yanfd@latest
```

To run YaNFD, run the `yanfd` (or `yanfd.exe`) executable.
To view a list of available options, specify the `-help` argument.

## Configuration

YaNFD's configuration is split into two components: startup configuration and runtime configuration.
Startup configuration sets default ports, queue sizes, logging levels, and other important options.
Meanwhile, runtime configuration is used to create NDN faces, set routes and strategies, and other related tasks.

Startup configuration for YaNFD is performed via a YAML file, by default read from `/usr/local/etc/ndn/yanfd.yml` on Unix-like systems. You can specify the configuration file location as the first CLI argument when starting YaNFD.

```bash
yanfd /path/to/yanfd.yml
```

Runtime configuration is performed via the [NFD Management Protocol](https://redmine.named-data.net/projects/nfd/wiki/Management).
At the moment, this requires the installation of the [NFD](https://github.com/named-data/NFD) package to obtain the `nfdc` configuration utility.

## Building from source

### Linux, macOS, BSD

To build and install YaNFD on Unix-like platforms, run:

```bash
make
sudo make install
```

### Windows

To build and install YaNFD on Windows, run the `go build` command in the `Makefile` manually:

```powershell
go build github.com/pulsejet/ndnd/fw/cmd/yanfd
```

## Publications

- Eric Newberry, Xinyu Ma, and Lixia Zhang. 2021. [YaNFD: Yet another named data networking forwarding daemon](https://dl.acm.org/doi/10.1145/3460417.3482969). In Proceedings of the 8th ACM Conference on Information-Centric Networking (ICN '21).
