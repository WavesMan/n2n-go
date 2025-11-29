//go:build darwin

package tap

import (
    "os"
    "golang.org/x/sys/unix"
)

func Open(name string, mtu int) (*Device, error) {
    fd, err := unix.Socket(unix.AF_SYSTEM, unix.SOCK_DGRAM, unix.SYSPROTO_CONTROL)
    if err != nil {
        return nil, err
    }
    var ci unix.CtlInfo
    for i := range ci.Name {
        ci.Name[i] = 0
    }
    n := []byte("com.apple.net.utun_control")
    if len(n) > len(ci.Name) {
        n = n[:len(ci.Name)]
    }
    for i := 0; i < len(n); i++ {
        ci.Name[i] = int8(n[i])
    }
    if err := unix.IoctlCtlInfo(fd, &ci); err != nil {
        unix.Close(fd)
        return nil, err
    }
    addr := &unix.SockaddrCtl{Sc_id: ci.Id, Sc_unit: 0}
    if err := unix.Connect(fd, addr); err != nil {
        unix.Close(fd)
        return nil, err
    }
    ifName, err := unix.GetsockoptString(fd, unix.SYSPROTO_CONTROL, 2)
    if err != nil {
        unix.Close(fd)
        return nil, err
    }
    f := os.NewFile(uintptr(fd), ifName)
    return &Device{File: f, Name: ifName}, nil
}