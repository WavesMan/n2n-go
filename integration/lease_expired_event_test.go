package integration

import (
    "net"
    "testing"
    "time"
    "n2n-go/pkg/sn"
)

func TestLeaseExpiredEvent(t *testing.T) {
    bind := "127.0.0.1"
    lp := 8768
    mp := 5768
    go sn.Run(bind, lp, mp)
    time.Sleep(100 * time.Millisecond)
    c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: mp})
    if err != nil { t.Fatal(err) }
    defer c.Close()
    c.Write([]byte("w 1 pool.set community 10.0.0.0 24 1"))
    buf := make([]byte, 1024)
    c.SetReadDeadline(time.Now().Add(time.Second))
    _, _, _ = c.ReadFrom(buf)
    c.Write([]byte("s 2 lease"))
    c.SetReadDeadline(time.Now().Add(time.Second))
    _, _, _ = c.ReadFrom(buf)
    c.Write([]byte("w 3 lease.reserve 00:11:22:33:44:55 community 10.0.0.10"))
    c.SetReadDeadline(time.Now().Add(3 * time.Second))
    n, _, err := c.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    if n == 0 { t.Fatal("no event") }
}

