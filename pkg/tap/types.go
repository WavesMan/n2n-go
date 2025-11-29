package tap

import (
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

