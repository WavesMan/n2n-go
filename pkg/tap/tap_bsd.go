//go:build freebsd || netbsd || openbsd

package tap

import (
    "fmt"
    "os"
    "regexp"
)

func Open(name string, mtu int) (*Device, error) {
    if name == "" {
        name = "tun0"
    }
    re := regexp.MustCompile(`^(tun|tap)(\d+)$`)
    m := re.FindStringSubmatch(name)
    if m == nil {
        return nil, fmt.Errorf("invalid device name")
    }
    path := "/dev/" + m[1] + m[2]
    f, err := os.OpenFile(path, os.O_RDWR, 0)
    if err != nil {
        return nil, err
    }
    return &Device{File: f, Name: name}, nil
}