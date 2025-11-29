package integration

import (
    "net"
    "testing"
    "time"
    "n2n-go/pkg/sn"
)

func TestMgmtPoolLeaseCommands(t *testing.T) {
    bind := "127.0.0.1"
    lp := 8767
    mp := 5767
    go sn.Run(bind, lp, mp)
    time.Sleep(100 * time.Millisecond)
    c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: mp})
    if err != nil { t.Fatal(err) }
    defer c.Close()
    // set pool
    c.Write([]byte("w 1 pool.set community 10.0.0.0 24 60"))
    buf := make([]byte, 1024)
    c.SetReadDeadline(time.Now().Add(time.Second))
    _, _, err = c.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    // list pool
    c.Write([]byte("r 2 pool.list"))
    c.SetReadDeadline(time.Now().Add(time.Second))
    _, _, err = c.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    // reserve lease
    c.Write([]byte("w 3 lease.reserve 00:11:22:33:44:55 community 10.0.0.10"))
    c.SetReadDeadline(time.Now().Add(time.Second))
    _, _, err = c.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    // list lease
    c.Write([]byte("r 4 lease.list"))
    c.SetReadDeadline(time.Now().Add(time.Second))
    _, _, err = c.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    // release lease
    c.Write([]byte("w 5 lease.release 00:11:22:33:44:55"))
    c.SetReadDeadline(time.Now().Add(time.Second))
    _, _, err = c.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
}

