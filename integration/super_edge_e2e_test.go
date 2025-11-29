package integration

import (
    "net"
    "testing"
    "time"
    "n2n-go/pkg/sn"
    "n2n-go/pkg/wire"
)

func TestRegisterQueryForward(t *testing.T) {
    bind := "127.0.0.1"
    lp := 8765
    mp := 5765
    go sn.Run(bind, lp, mp)
    time.Sleep(100 * time.Millisecond)
    c, err := net.DialUDP("udp", nil, &net.UDPAddr{IP: net.ParseIP(bind), Port: lp})
    if err != nil { t.Fatal(err) }
    defer c.Close()
    community := []byte("community")
    regc := wire.Common{TTL: 2, PC: wire.MsgRegisterSuper, Flags: 0}
    copy(regc.Community[:], community)
    r := wire.RegisterSuper{}
    b := make([]byte, 256)
    n := wire.EncodeRegisterSuper(regc, r, b)
    c.Write(b[:n])
    buf := make([]byte, 512)
    c.SetReadDeadline(time.Now().Add(time.Second))
    rn, _, err := c.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    i := 0
    _, ok := wire.DecodeCommon(buf[:rn], &i)
    if !ok { t.Fatal("common") }
    _, aok := wire.DecodeRegisterSuperAck(buf[:rn], &i)
    if !aok { t.Fatal("ack") }
    qc := wire.Common{TTL: 2, PC: wire.MsgQueryPeer, Flags: 0}
    copy(qc.Community[:], community)
    q := wire.QueryPeer{}
    out := make([]byte, 128)
    qn := wire.EncodeQueryPeer(qc, q, out)
    c.Write(out[:qn])
    c.SetReadDeadline(time.Now().Add(time.Second))
    pn, _, err := c.ReadFrom(buf)
    if err != nil { t.Fatal(err) }
    j := 0
    _, ok = wire.DecodeCommon(buf[:pn], &j)
    if !ok { t.Fatal("common2") }
    _, piok := wire.DecodePeerInfo(buf[:pn], &j)
    if !piok { t.Fatal("peerinfo") }
}

