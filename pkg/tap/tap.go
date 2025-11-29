//go:build !linux && !windows && !darwin && !freebsd && !netbsd && !openbsd

package tap

import (
    "errors"
)

func Open(name string, mtu int) (*Device, error) {
    return nil, errors.New("tap not supported on this platform yet")
}

func Configure(name string, mtu int) error {
    return nil
}
