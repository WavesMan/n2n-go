//go:build linux

package tap

import (
    "os"
    "syscall"
    "unsafe"
)

const (
    IFF_TAP  = 0x0002
    IFF_NO_PI = 0x1000
    TUNSETIFF = 0x400454ca
)

func Open(name string, mtu int) (*Device, error) {
    f, err := os.OpenFile("/dev/net/tun", os.O_RDWR, 0)
    if err != nil {
        return nil, err
    }
    if name == "" {
        name = "tap0"
    }
    var ifr [40]byte
    bs := []byte(name)
    if len(bs) > 15 {
        bs = bs[:15]
    }
    copy(ifr[:], bs)
    *(*uint16)(unsafe.Pointer(&ifr[16])) = uint16(IFF_TAP | IFF_NO_PI)
    _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, f.Fd(), uintptr(TUNSETIFF), uintptr(unsafe.Pointer(&ifr[0])))
    if errno != 0 {
        f.Close()
        return nil, errno
    }
    return &Device{File: f, Name: name}, nil
}