//go:build !windows

package tap

func ConfigureIPv4(name string, ip string, mask string, metric int) error {
    return nil
}

