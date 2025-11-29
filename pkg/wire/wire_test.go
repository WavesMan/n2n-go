package wire

import "testing"

func TestCommonEncodeDecode(t *testing.T) {
    c := Common{TTL: 2, PC: MsgRegisterSuper, Flags: 0}
    copy(c.Community[:], []byte("community"))
    b := make([]byte, 64)
    n := EncodeCommon(c, b)
    i := 0
    d, ok := DecodeCommon(b[:n], &i)
    if !ok { t.Fatal("decode") }
    if d.TTL != c.TTL || d.PC != c.PC || d.Flags != c.Flags { t.Fatal("mismatch") }
}

func TestPacketEncodeDecode(t *testing.T) {
    c := Common{TTL: 2, PC: MsgPacket, Flags: 0}
    p := Packet{}
    p.Transform = TransformNull
    p.Compression = CompressionNone
    payload := []byte("abc")
    b := make([]byte, 64)
    n := EncodePacket(c, p, payload, b)
    i := 0
    _, ok := DecodeCommon(b[:n], &i)
    if !ok { t.Fatal("common") }
    out := make([]byte, 64)
    got, gok, gn := DecodePacket(b[:n], &i, out)
    if !gok || gn != len(payload) { t.Fatal("packet") }
    if got.Transform != p.Transform || got.Compression != p.Compression { t.Fatal("fields") }
}

func TestRegisterSuperEncodeDecode(t *testing.T) {
    c := Common{TTL: 2, PC: MsgRegisterSuper, Flags: 0}
    r := RegisterSuper{}
    r.Cookie = 1
    r.AuthScheme = 2
    r.AuthToken = []byte("tok")
    r.KeyTime = 123456
    b := make([]byte, 128)
    n := EncodeRegisterSuper(c, r, b)
    i := 0
    _, ok := DecodeCommon(b[:n], &i)
    if !ok { t.Fatal("common") }
    got, rok := DecodeRegisterSuper(b[:n], &i)
    if !rok { t.Fatal("regsup") }
    if got.AuthScheme != r.AuthScheme || got.KeyTime != r.KeyTime || string(got.AuthToken) != string(r.AuthToken) { t.Fatal("fields") }
}

func TestRegisterAckEncodeDecode(t *testing.T) {
    c := Common{TTL: 2, PC: MsgRegisterAck, Flags: 0}
    a := RegisterAck{Cookie: 7, DevAddr: IPSubnet{NetAddr: 0x0a000000, Bitlen: 24}, Lifetime: 60}
    b := make([]byte, 128)
    n := EncodeRegisterAck(c, a, b)
    i := 0
    _, ok := DecodeCommon(b[:n], &i)
    if !ok { t.Fatal("common") }
    got, gok := DecodeRegisterAck(b[:n], &i)
    if !gok { t.Fatal("regack") }
    if got.Cookie != a.Cookie || got.DevAddr.Bitlen != a.DevAddr.Bitlen || got.Lifetime != a.Lifetime { t.Fatal("fields") }
}

func TestUnregisterSuperEncodeDecode(t *testing.T) {
    c := Common{TTL: 2, PC: MsgUnregisterSuper, Flags: 0}
    u := UnregisterSuper{Cookie: 9}
    b := make([]byte, 64)
    n := EncodeUnregisterSuper(c, u, b)
    i := 0
    _, ok := DecodeCommon(b[:n], &i)
    if !ok { t.Fatal("common") }
    got, gok := DecodeUnregisterSuper(b[:n], &i)
    if !gok || got.Cookie != u.Cookie { t.Fatal("unreg") }
}

func TestRegisterSuperNakEncodeDecode(t *testing.T) {
    c := Common{TTL: 2, PC: MsgRegisterSuperNak, Flags: 0}
    n := RegisterSuperNak{Cookie: 3, Reason: 2}
    b := make([]byte, 64)
    m := EncodeRegisterSuperNak(c, n, b)
    i := 0
    _, ok := DecodeCommon(b[:m], &i)
    if !ok { t.Fatal("common") }
    got, gok := DecodeRegisterSuperNak(b[:m], &i)
    if !gok || got.Cookie != n.Cookie || got.Reason != n.Reason { t.Fatal("nak") }
}

func TestRegisterSuperAckAuth(t *testing.T) {
    c := Common{TTL: 2, PC: MsgRegisterSuperAck, Flags: 0}
    a := RegisterSuperAck{Cookie: 7, DevAddr: IPSubnet{NetAddr: 0x0a000000, Bitlen: 24}, Lifetime: 60}
    a.AuthScheme = 1
    a.AuthToken = []byte("x")
    b := make([]byte, 256)
    n := EncodeRegisterSuperAck(c, a, b)
    i := 0
    _, ok := DecodeCommon(b[:n], &i)
    if !ok { t.Fatal("common") }
    got, gok := DecodeRegisterSuperAck(b[:n], &i)
    if !gok || got.AuthScheme != a.AuthScheme || string(got.AuthToken) != string(a.AuthToken) { t.Fatal("ackauth") }
}

func TestUnregisterSuperAuth(t *testing.T) {
    c := Common{TTL: 2, PC: MsgUnregisterSuper, Flags: 0}
    u := UnregisterSuper{Cookie: 9, AuthScheme: 3, AuthToken: []byte("abc")}
    b := make([]byte, 128)
    n := EncodeUnregisterSuper(c, u, b)
    i := 0
    _, ok := DecodeCommon(b[:n], &i)
    if !ok { t.Fatal("common") }
    got, gok := DecodeUnregisterSuper(b[:n], &i)
    if !gok || got.AuthScheme != u.AuthScheme || string(got.AuthToken) != string(u.AuthToken) { t.Fatal("unauth") }
}
