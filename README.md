
# rtop

`rtop` is a remote system monitor. It connects over SSH to a remote system
and displays vital system metrics (CPU, disk, memory, network). No special
software is needed on the remote system, other than an SSH server and
working credentials.

Only Linux systems can be monitored, but most modern distros will work.

`rtop` is MIT-licensed and can be used anywhere with attribution.

*`rtop`'s [home page](http://www.rtop-project.org/) has more information
and screenshots!*

## build

`rtop` is written in [go](http://golang.org/), and needs just one dependent
library [x/crypto](https://golang.org/pkg/crypto/). `rtop` does not use any
dependency managers, just a git submodule. Follow these steps to build:

    git clone http://github.com/rapidloop/rtop
    cd rtop
    make init
    make

Note that `make init` will init the git submodule, so be sure to build from
a git clone. You need to `make init` only once.

## contribute

Pull requests welcome. Keep it simple.

## changelog
* 9-May-2015: first public release
