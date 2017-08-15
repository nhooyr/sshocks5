# sshocks5

*note: macOS only*

## Usage

```zsh
sshocks5 --help
```

## Explanation

sshocks5 will use `ssh` to connect to a host of your choosing
and open a SOCKS5 proxy on localhost:5030 (configurable). Then, it will
use `networksetup` to modify your `Wi-Fi` (configurable) network
to make use of the proxy. Once you press Ctrl+C in the terminal, it will
stop your network from using SOCKS5 Proxy and then kill the SSH process.

sshocks5 will use `sudo` to modify the network configuration.

## TODO

- [ ] Only run `sudo` once instead of once for setting the proxy
      and another for removing it.