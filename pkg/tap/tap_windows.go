//go:build windows

package tap

import (
    "os"
    "strings"
    "golang.org/x/sys/windows"
    "os/exec"
    "fmt"
)

func Open(name string, mtu int) (*Device, error) {
    if name == "" {
        // Expect a TAP GUID name is provided; fallback to the first Global TAP
        name = "Global"
    }
    path := "\\\\.\\" + name
    if !strings.HasSuffix(strings.ToLower(path), ".tap") {
        path = path + ".tap"
    }
    h, err := windows.CreateFile(windows.StringToUTF16Ptr(path), windows.GENERIC_READ|windows.GENERIC_WRITE, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE, nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
    if err != nil {
        return nil, err
    }
    f := os.NewFile(uintptr(h), path)
    return &Device{File: f, Name: name}, nil
}

func Configure(name string, mtu int) error {
    if name == "" { return nil }
    exec.Command("netsh", "interface", "ipv4", "set", "subinterface", name, "mtu=", fmt.Sprintf("%d", mtu), "store=persistent").Run()
    exec.Command("netsh", "interface", "ipv4", "set", "interface", name, "metric=10").Run()
    return nil
}

func ConfigureIPv4(name string, ip string, mask string, metric int) error {
    if name == "" { return nil }
    if ip != "" && mask != "" {
        exec.Command("netsh", "interface", "ip", "set", "address", name, "static", ip, mask).Run()
    }
    if metric > 0 {
        exec.Command("netsh", "interface", "ipv4", "set", "interface", name, "metric=", fmt.Sprintf("%d", metric)).Run()
    }
    exec.Command("netsh", "interface", "set", "interface", name, "enable").Run()
    return nil
}
