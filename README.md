
# rtop

[![Join the chat at https://gitter.im/rapidloop/rtop](https://badges.gitter.im/Join%20Chat.svg)](https://gitter.im/rapidloop/rtop?utm_source=badge&utm_medium=badge&utm_campaign=pr-badge&utm_content=badge)

`rtop` is a remote system monitor. It connects over SSH to a remote system
and displays vital system metrics (CPU, disk, memory, network). No special
software is needed on the remote system, other than an SSH server and
working credentials.

Only Linux systems can be monitored, and most modern distros will work.

`rtop` is MIT-licensed and can be used anywhere with attribution.

*`rtop`'s [home page](http://www.rtop-monitor.org/) has more information
and screenshots!*

## build

`rtop` is written in [go](http://golang.org/), and requires Go version 1.2
or higher. To build, `go get` it:

    go get github.com/rapidloop/rtop

You should find the binary `rtop` under `$GOPATH/bin` when the command
completes. There are no runtime dependencies or configuration needed.

## contribute

Pull requests welcome. Keep it simple.

## changelog
* 9-May-2015: first public release
