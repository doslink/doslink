Doslink
====

[![Build Status](https://travis-ci.org/Doslink/doslink.svg)](https://travis-ci.org/Doslink/doslink) [![AGPL v3](https://img.shields.io/badge/license-AGPL%20v3-brightgreen.svg)](./LICENSE)

**Official golang implementation of the Doslink protocol.**

Automated builds are available for stable releases and the unstable master branch. Binary archives are published at https://github.com/doslink/doslink/releases.

## What is Doslink?

Doslink is software designed to operate and connect to highly scalable blockchain networks confirming to the Doslink Blockchain Protocol, which allows partipicants to define, issue and transfer digitial assets on a multi-asset shared ledger. Please refer to the [White Paper](https://github.com/doslink/wiki/blob/master/White-Paper/%E6%AF%94%E5%8E%9F%E9%93%BE%E6%8A%80%E6%9C%AF%E7%99%BD%E7%9A%AE%E4%B9%A6-%E8%8B%B1%E6%96%87%E7%89%88.md) for more details.

In the current state `doslink` is able to:

- Manage key, account as well as asset
- Send transactions, i.e., issue, spend and retire asset


## Building from source

### Requirements

- [Go](https://golang.org/doc/install) version 1.8 or higher, with `$GOPATH` set to your preferred directory

### Installation

Ensure Go with the supported version is installed properly:

```bash
$ go version
$ go env GOROOT GOPATH
```

- Get the source code

``` bash
$ git clone https://github.com/doslink/doslink.git $GOPATH/src/github.com/doslink/doslink
```

- Build source code

``` bash
$ cd $GOPATH/src/github.com/doslink/doslink
$ make server    # build server
$ make client  # build client
```

When successfully building the project, the `server` and `client` binary should be present in `cmd/server` and `cmd/client` directory, respectively.

### Executables

The Doslink project comes with several executables found in the `cmd` directory.

| Command      | Description                                                  |
| ------------ | ------------------------------------------------------------ |
| **server**   | server command can help to initialize and launch doslink domain by custom parameters. `server --help` for command line options. |
| **client**   | Our main Doslink CLI client. It is the entry point into the Doslink network (main-, test- or private net), capable of running as a full node archive node (retaining all historical state). It can be used by other processes as a gateway into the Doslink network via JSON RPC endpoints exposed on top of HTTP, WebSocket and/or IPC transports. `client --help` and the [client Wiki page](https://github.com/doslink/doslink/wiki/Command-Line-Options) for command line options. |

## Running doslink

Currently, doslink is still in active development and a ton of work needs to be done, but we also provide the following content for these eager to do something with `doslink`. This section won't cover all the commands of `server` and `client` at length, for more information, please the help of every command, e.g., `client help`.

### Initialize

First of all, initialize the node:

```bash
$ cd ./cmd/server
$ ./server init --chain_id mainnet
```

There are three options for the flag `--chain_id`:

- `mainnet`: connect to the mainnet.
- `testnet`: connect to the testnet.
- `solonet`: standalone mode.

After that, you'll see `config.toml` generated, then launch the node.

### launch

``` bash
$ ./server node
```

available flags for `server node`:

```
      --auth.disable                Disable rpc access authenticate
      --chain_id string             Select network type
  -h, --help                        help for node
      --mining                      Enable mining
      --p2p.dial_timeout int        Set dial timeout (default 3)
      --p2p.handshake_timeout int   Set handshake timeout (default 30)
      --p2p.laddr string            Node listen address.
      --p2p.max_num_peers int       Set max num peers (default 50)
      --p2p.pex                     Enable Peer-Exchange  (default true)
      --p2p.seeds string            Comma delimited host:port seed nodes
      --p2p.skip_upnp               Skip UPNP configuration
      --prof_laddr string           Use http to profile server programs
      --vault_mode                  Run in the offline enviroment
      --wallet.disable              Disable wallet
      --wallet.rescan               Rescan wallet
      --web.closed                  Lanch web browser or not
```

Given the `server` node is running, the general workflow is as follows:

- create key, then you can create account and asset.
- send transaction, i.e., build, sign and submit transaction.
- query all kinds of information, let's say, avaliable key, account, key, balances, transactions, etc.

What is more,

+ if you are using _Mac_, please make sure _llvm_ is installed by `brew install llvm`.
+ if you are using _Windows_, please make sure _mingw-w64_ is installed and set up the _PATH_ environment variable accordingly.

For more details about using `client` command please refer to [API Reference](https://github.com/doslink/doslink/wiki/API-Reference)

### Dashboard

Access the dashboard:

```
$ open http://localhost:6051/
```

### In Docker

Ensure your [Docker](https://www.docker.com/) version is 17.05 or higher.

```bash
$ docker build -t doslink .
```

For the usage please refer to [running-in-docker-wiki](https://github.com/doslink/doslink/wiki/Running-in-Docker).

## Contributing

Thank you for considering helping out with the source code! Any contributions are highly appreciated, and we are grateful for even the smallest of fixes!

If you run into an issue, feel free to [doslink issues](https://github.com/doslink/doslink/issues/) in this repository. We are glad to help!

## License

[AGPL v3](./LICENSE)
