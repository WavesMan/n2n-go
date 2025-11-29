package integration

import (
    "net"
    "os"
    "testing"
    "time"
    "n2n-go/pkg/wire"
)

func TestExternalRegisterQueryPeer(t *testing.T) {
    addr := os.Getenv("N2N_SN_ADDR")
    if addr == "" { addr = "150.109.108.121:7654" }
    // if addr == "" { addr = "172.26.175.210:7654" }
    raddr, err := net.ResolveUDPAddr("udp", addr)
    if err != nil { t.Fatal(err) }

    c, err := net.ListenUDP("udp", nil)
    if err != nil { t.Fatal(err) }
    defer c.Close()

    community := []byte("community")
    regc := wire.Common{TTL: 2, PC: wire.MsgRegisterSuper, Flags: 0}
    copy(regc.Community[:], community)
    mac := wire.Mac{0x10,0x20,0x30,0x40,0x50,0x60}
    r := wire.RegisterSuper{}
    copy(r.EdgeMac[:], mac[:])
    b := make([]byte, 256)
    n := wire.EncodeRegisterSuper(regc, r, b)
    c.WriteToUDP(b[:n], raddr)

    buf := make([]byte, 512)
    c.SetReadDeadline(time.Now().Add(2 * time.Second))
    rn, _, err := c.ReadFromUDP(buf)
    if err != nil { t.Fatal(err) }
    i := 0
    _, ok := wire.DecodeCommon(buf[:rn], &i)
    if !ok { t.Fatal("common") }
    _, aok := wire.DecodeRegisterSuperAck(buf[:rn], &i)
    if !aok { t.Fatal("ack") }

    qc := wire.Common{TTL: 2, PC: wire.MsgQueryPeer, Flags: 0}
    copy(qc.Community[:], community)
    q := wire.QueryPeer{}
    copy(q.SrcMac[:], mac[:])
    copy(q.TargetMac[:], mac[:])
    out := make([]byte, 128)
    qn := wire.EncodeQueryPeer(qc, q, out)
    c.WriteToUDP(out[:qn], raddr)

    c.SetReadDeadline(time.Now().Add(2 * time.Second))
    pn, _, err := c.ReadFromUDP(buf)
    if err != nil { t.Fatal(err) }
    j := 0
    _, ok = wire.DecodeCommon(buf[:pn], &j)
    if !ok { t.Fatal("common2") }
    _, piok := wire.DecodePeerInfo(buf[:pn], &j)
    if !piok { t.Fatal("peerinfo") }
}
