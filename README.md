# YaNFD - Yet another Named Data Networking Forwarding Daemon

YaNFD is a forwarding daemon for the [Named Data Networking](https://named-data.net) platform written in Go.
It is compatible with existing NDN applications and utilizes the management tools and protocols developed for the [NFD](https://github.com/named-data/NFD) forwarder.

# Prerequisites

YaNFD requires [Go 1.20+](https://go.dev/doc/install), although it may be possible to use older versions of Go.
Besides that, you will need `libpcap` and `g++` on Linux and MacOS. On Ubuntu, these libraries can be installed by:

```bash
sudo apt install build-essential pkg-config libpcap-dev
```

You may refer to [this](https://github.com/zjkmxy/YaNFD-docker) if you want to build a Docker image.

YaNFD has been developed and tested on Linux (namely, Ubuntu).
However, we have designed it with support for Windows, macOS, and BSD in mind.
We have received reports that YaNFD operates properly on Windows 10 (with minor changes -- see below) and macOS, but this has not been evaluated by the developers.

# Installation

## Install the YaNFD binary

```bash
go install github.com/named-data/YaNFD/cmd/yanfd@latest
```

## Install YaNFD from Windows Store

Get it from: https://www.microsoft.com/store/apps/9NBK3ZJT4CL8

## Install the configuration file
### On MacOS/Linux
```bash
curl -o ./yanfd.toml https://raw.githubusercontent.com/named-data/YaNFD/master/yanfd.toml.sample
mkdir -p /usr/local/etc/ndn
install -m 644 ./yanfd.toml /usr/local/etc/ndn
rm ./yanfd.toml
```

On MacOS, one also needs to change `socket_path` to `/var/run/nfd/nfd.sock` in the copied configuration file.

### On Windows 10/11
```text
curl -o yanfd.toml https://raw.githubusercontent.com/named-data/YaNFD/master/yanfd.toml.sample
mkdir %APPDATA%\ndn
move yanfd.toml %APPDATA%\ndn\
```

One needs to change `socket_path` to `${TEMP}\\nfd\\nfd.sock` in the copied configuration file.
Also, to execute YaNFD on Windows 10, one needs to explicitly specify the configuration path:
```text
yanfd.exe --config=%APPDATA%\ndn\yanfd.toml
```

# Building from source

## Linux, macOS, BSD

To build and install YaNFD on Unix-like platforms, run:

    make
    sudo make install

## Windows 10

To build and install YaNFD on Windows, please run the `go build` command in the `Makefile` manually:
```text
go build github.com/named-data/YaNFD/cmd/yanfd
```

At the moment, you will need to manually install the executable (`yanfd.exe`) and the configuration file (`yanfd.toml.sample`) to a location of your choice.

# Configuration

YaNFD's configuration is split into two components: startup configuration and runtime configuration.
Startup configuration sets default ports, queue sizes, logging levels, and other important options.
Meanwhile, runtime configuration is used to create NDN faces, set routes and strategies, and other related tasks.

## Startup configuration

Startup configuration for YaNFD is performed via a TOML file, by default read from `/usr/local/etc/ndn/yanfd.toml` on Unix-like systems.
Note that you will need to copy the sample config file installed to `/usr/local/etc/ndn/yanfd.toml.sample` to this location before running YaNFD for the first time.
The configuration options are documented via comments in the sample file.

On Windows, at this time, you will need to specify the location of the configuration file manually when starting YaNFD via the `--config` argument.

## Runtime configuration

Runtime configuration is performed via the [NFD Management Protocol](https://redmine.named-data.net/projects/nfd/wiki/Management).
At the moment, this requires the installation of the [NFD](https://github.com/named-data/NFD) package to obtain the `nfdc` configuration utility.
YaNFD supports the majority of this management protocol, but some features are currently unsupported, such as ContentStore management.

# Running

To run YaNFD, run the `yanfd` (or `yanfd.exe`) executable.
To view a list of available options, specify the `--help` argument. 

After starting YaNFD, you can treat it like NFD from an application and configuration perspective.

# Publications

- Eric Newberry, Xinyu Ma, and Lixia Zhang. 2021. [YaNFD: Yet another named data networking forwarding daemon](https://dl.acm.org/doi/10.1145/3460417.3482969). In Proceedings of the 8th ACM Conference on Information-Centric Networking (ICN '21).
