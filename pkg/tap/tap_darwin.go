//go:build darwin

package tap

import (
    "errors"
)

func Open(name string, mtu int) (*Device, error) {
    return nil, errors.New("tap not implemented for darwin in this build")
}
