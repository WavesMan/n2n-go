package integration

import (
    "net"
    "testing"
    "time"
    "n2n-go/pkg/sn"
    "n2n-go/pkg/wire"
)

func TestPacketForward(t *testing.T) {
    bind := "127.0.0.1"
    lp := 8766
    mp := 5766
    go sn.Run(bind, lp, mp)
    time.Sleep(100 * time.Millisecond)

    c1, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(bind), Port: 0})
    if err != nil { t.Fatal(err) }
    defer c1.Close()
    c2, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP(bind), Port: 0})
    if err != nil { t.Fatal(err) }
    defer c2.Close()
    dest := &net.UDPAddr{IP: net.ParseIP(bind), Port: lp}

    community := []byte("community")
    mac1 := wire.Mac{0x00,0x11,0x22,0x33,0x44,0x55}
    mac2 := wire.Mac{0xaa,0xbb,0xcc,0xdd,0xee,0xff}

    // Register edge1
    rc := wire.Common{TTL: 2, PC: wire.MsgRegisterSuper, Flags: 0}
    copy(rc.Community[:], community)
    r := wire.RegisterSuper{}
    copy(r.EdgeMac[:], mac1[:])
    b := make([]byte, 256)
    n := wire.EncodeRegisterSuper(rc, r, b)
    c1.WriteToUDP(b[:n], dest)
    buf := make([]byte, 512)
    c1.SetReadDeadline(time.Now().Add(time.Second))
    _, _, err = c1.ReadFromUDP(buf)
    if err != nil { t.Fatal(err) }

    // Register edge2
    r2 := wire.RegisterSuper{}
    copy(r2.EdgeMac[:], mac2[:])
    n2 := wire.EncodeRegisterSuper(rc, r2, b)
    c2.WriteToUDP(b[:n2], dest)
    c2.SetReadDeadline(time.Now().Add(time.Second))
    _, _, err = c2.ReadFromUDP(buf)
    if err != nil { t.Fatal(err) }

    // Send PACKET from edge1 to edge2
    pc := wire.Common{TTL: 2, PC: wire.MsgPacket, Flags: 0}
    copy(pc.Community[:], community)
    pkt := wire.Packet{}
    copy(pkt.SrcMac[:], mac1[:])
    copy(pkt.DstMac[:], mac2[:])
    payload := []byte("hello")
    out := make([]byte, 1024)
    m := wire.EncodePacket(pc, pkt, payload, out)
    c1.WriteToUDP(out[:m], dest)

    // Expect forwarded to c2
    c2.SetReadDeadline(time.Now().Add(time.Second))
    fn, _, err := c2.ReadFromUDP(buf)
    if err != nil { t.Fatal(err) }
    i := 0
    _, ok := wire.DecodeCommon(buf[:fn], &i)
    if !ok { t.Fatal("common") }
    pay := make([]byte, 32)
    got, gok, gn := wire.DecodePacket(buf[:fn], &i, pay)
    if !gok { t.Fatal("packet") }
    if gn != len(payload) || string(got.Payload) != string(payload) { t.Fatal("payload mismatch") }
}

