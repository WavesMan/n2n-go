package tap

import (
    "errors"
    "os"
)

type Device struct {
    File *os.File
    Name string
}

func (d *Device) Read(b []byte) (int, error) {
    return d.File.Read(b)
}

func (d *Device) Write(b []byte) (int, error) {
    return d.File.Write(b)
}

func (d *Device) Close() error {
    if d.File != nil {
        return d.File.Close()
    }
    return nil
}

func Open(name string, mtu int) (*Device, error) {
    return nil, errors.New("tap not supported on this platform yet")
}

func Configure(name string, mtu int) error {
    return nil
}
