# sshocks5

*note: macOS only*

## Usage

```fish
$ sshocks5 --help
Usage of sshocks5:
  -D string
      socks5 listening address (addr:port) (default "localhost:5030")
  -host string
      host to connect to
  -net string
      network to configure to use SOCKS5 proxy (default "Wi-Fi")
  -port string
      port to connect to
```

### Example

```fish
$ sshocks5 --host coder.com
```

## Explanation

sshocks5 will use `ssh` to connect to a host/port of your choosing
and open a SOCKS5 proxy on localhost:5030 (configurable). Then, it will
use `networksetup` to modify your `Wi-Fi` (configurable) network
to make use of the proxy. Once you press Ctrl+C in the terminal, it will
stop your network from using SOCKS5 Proxy and then kill the SSH process.

sshocks5 will use `sudo` to modify the network configuration.