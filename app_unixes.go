// +build aix darwin dragonfly freebsd linux netbsd openbsd solaris

package fx

import "golang.org/x/sys/unix"

const sigINT = unix.SIGINT
const sigTERM = unix.SIGTERM
