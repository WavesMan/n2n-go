package wire

import (
    "encoding/binary"
)

const (
    Version = 3
    DescSize = 16
    MsgRegister = 1
    MsgDeregister = 2
    MsgPacket = 3
    MsgRegisterAck = 4
    MsgRegisterSuper = 5
    MsgUnregisterSuper = 6
    MsgRegisterSuperAck = 7
    MsgRegisterSuperNak = 8
    MsgPeerInfo = 10
    MsgQueryPeer = 11
    MsgReRegisterSuper = 12
)

type Mac [6]byte

type Sock struct {
    Family uint8
    Type   uint8
    Port   uint16
    AddrV4 [4]byte
    AddrV6 [16]byte
}

type IPSubnet struct {
    NetAddr uint32
    Bitlen  uint8
}

type Common struct {
    // Version is encoded/decoded but not stored in struct to preserve API
    TTL       uint8
    PC        uint8
    Flags     uint16
    Community [20]byte
}

type RegisterSuper struct {
    Cookie  uint32
    EdgeMac Mac
    Sock    Sock
    DevAddr IPSubnet
    DevDesc [DescSize]byte
    AuthScheme uint16
    AuthToken  []byte
    KeyTime uint32
}

type RegisterSuperAck struct {
    Cookie   uint32
    SrcMac   Mac
    DevAddr  IPSubnet
    Lifetime uint16
    Sock     Sock
    AuthScheme uint16
    AuthToken  []byte
}

type RegisterAck struct {
    Cookie   uint32
    DevAddr  IPSubnet
    Lifetime uint16
    Sock     Sock
}

type UnregisterSuper struct {
    Cookie  uint32
    EdgeMac Mac
    AuthScheme uint16
    AuthToken  []byte
}

type RegisterSuperNak struct {
    Cookie uint32
    Reason uint16
}

func putUint8(b []byte, i *int, v uint8) {
    b[*i] = v
    *i++
}

func putUint16(b []byte, i *int, v uint16) {
    binary.BigEndian.PutUint16(b[*i:*i+2], v)
    *i += 2
}

func putUint32(b []byte, i *int, v uint32) {
    binary.BigEndian.PutUint32(b[*i:*i+4], v)
    *i += 4
}

func getUint8(b []byte, i *int) uint8 {
    v := b[*i]
    *i++
    return v
}

func getUint16(b []byte, i *int) uint16 {
    v := binary.BigEndian.Uint16(b[*i:*i+2])
    *i += 2
    return v
}

func getUint32(b []byte, i *int) uint32 {
    v := binary.BigEndian.Uint32(b[*i:*i+4])
    *i += 4
    return v
}

func EncodeCommon(c Common, dst []byte) int {
    i := 0
    putUint8(dst, &i, Version)
    putUint8(dst, &i, c.TTL)
    f := (c.Flags & 0xFFE0) | uint16(c.PC&0x1F)
    putUint16(dst, &i, f)
    copy(dst[i:i+20], c.Community[:])
    i += 20
    return i
}

func DecodeCommon(src []byte, i *int) (Common, bool) {
    if len(src)-*i < 24 {
        return Common{}, false
    }
    c := Common{}
    v := getUint8(src, i)
    if v != Version && v != 1 {
        return Common{}, false
    }
    c.TTL = getUint8(src, i)
    // Try new format: pc byte + flags
    j := *i
    if len(src)-j >= 1+2+20 {
        pc := src[j]
        if pc >= 1 && pc <= 12 {
            *i = j + 1
            c.PC = pc
            c.Flags = getUint16(src, i)
            copy(c.Community[:], src[*i:*i+20])
            *i += 20
            return c, true
        }
    }
    // Fallback to old format: flags only (type encoded in low 5 bits)
    if len(src)-j >= 2+20 {
        c.Flags = getUint16(src, i)
        c.PC = uint8(c.Flags & 0x1F)
        copy(c.Community[:], src[*i:*i+20])
        *i += 20
        return c, true
    }
    return Common{}, false
}

func EncodeSock(s Sock, dst []byte) int {
    i := 0
    var f uint16
    if s.Family == 2 {
        f = 0
    } else {
        f = 0x8000
    }
    if s.Type != 2 {
        f |= 0x4000
    }
    putUint16(dst, &i, f)
    putUint16(dst, &i, s.Port)
    if f&0x8000 != 0 {
        copy(dst[i:i+16], s.AddrV6[:])
        i += 16
    } else {
        copy(dst[i:i+4], s.AddrV4[:])
        i += 4
    }
    return i
}

func DecodeSock(src []byte, i *int) (Sock, bool) {
    if len(src)-*i < 2+2+4 {
        return Sock{}, false
    }
    s := Sock{}
    f := getUint16(src, i)
    // Heuristic: if top bits look like flags, parse as new format; else fallback to old format
    if (f&0xC000) != 0 || (f == 0) {
        s.Port = getUint16(src, i)
        if f&0x8000 != 0 {
            s.Family = 10
            copy(s.AddrV6[:], src[*i:*i+16])
            *i += 16
        } else {
            s.Family = 2
            copy(s.AddrV4[:], src[*i:*i+4])
            *i += 4
        }
        if f&0x4000 != 0 {
            s.Type = 1
        } else {
            s.Type = 2
        }
        return s, true
    }
    // old format fallback
    s.Type = uint8(f & 0xFF)
    s.Family = uint8(f >> 8)
    s.Port = getUint16(src, i)
    if s.Family == 2 {
        copy(s.AddrV4[:], src[*i:*i+4])
        *i += 4
        if len(src)-*i >= 12 { *i += 12 }
    } else {
        copy(s.AddrV6[:], src[*i:*i+16])
        *i += 16
    }
    return s, true
}

func EncodeRegisterSuper(c Common, r RegisterSuper, dst []byte) int {
    i := 0
    i += EncodeCommon(c, dst[i:])
    putUint32(dst, &i, r.Cookie)
    copy(dst[i:i+6], r.EdgeMac[:])
    i += 6
    i += EncodeSock(r.Sock, dst[i:])
    putUint32(dst, &i, r.DevAddr.NetAddr)
    putUint8(dst, &i, r.DevAddr.Bitlen)
    copy(dst[i:i+DescSize], r.DevDesc[:])
    i += DescSize
    putUint16(dst, &i, r.AuthScheme)
    putUint16(dst, &i, uint16(len(r.AuthToken)))
    copy(dst[i:i+len(r.AuthToken)], r.AuthToken)
    i += len(r.AuthToken)
    putUint32(dst, &i, r.KeyTime)
    return i
}

func DecodeRegisterSuper(src []byte, i *int) (RegisterSuper, bool) {
    r := RegisterSuper{}
    if len(src)-*i < 4+6+2+2+4+DescSize {
        return r, false
    }
    r.Cookie = getUint32(src, i)
    copy(r.EdgeMac[:], src[*i:*i+6])
    *i += 6
    s, ok := DecodeSock(src, i)
    if !ok {
        return r, false
    }
    r.Sock = s
    r.DevAddr.NetAddr = getUint32(src, i)
    r.DevAddr.Bitlen = getUint8(src, i)
    copy(r.DevDesc[:], src[*i:*i+DescSize])
    *i += DescSize
    if len(src)-*i >= 2 {
        r.AuthScheme = getUint16(src, i)
    }
    if len(src)-*i >= 2 {
        tlen := int(getUint16(src, i))
        if tlen > 0 && len(src)-*i >= tlen {
            r.AuthToken = make([]byte, tlen)
            copy(r.AuthToken, src[*i:*i+tlen])
            *i += tlen
        }
    }
    if len(src)-*i >= 4 { r.KeyTime = getUint32(src, i) }
    return r, true
}

func EncodeRegisterSuperAck(c Common, a RegisterSuperAck, dst []byte) int {
    i := 0
    i += EncodeCommon(c, dst[i:])
    putUint32(dst, &i, a.Cookie)
    copy(dst[i:i+6], a.SrcMac[:])
    i += 6
    putUint32(dst, &i, a.DevAddr.NetAddr)
    putUint8(dst, &i, a.DevAddr.Bitlen)
    putUint16(dst, &i, a.Lifetime)
    i += EncodeSock(a.Sock, dst[i:])
    putUint16(dst, &i, a.AuthScheme)
    putUint16(dst, &i, uint16(len(a.AuthToken)))
    copy(dst[i:i+len(a.AuthToken)], a.AuthToken)
    i += len(a.AuthToken)
    return i
}

func DecodeRegisterSuperAck(src []byte, i *int) (RegisterSuperAck, bool) {
    a := RegisterSuperAck{}
    if len(src)-*i < 4+6+5+2+2+4 {
        return a, false
    }
    a.Cookie = getUint32(src, i)
    copy(a.SrcMac[:], src[*i:*i+6])
    *i += 6
    a.DevAddr.NetAddr = getUint32(src, i)
    a.DevAddr.Bitlen = getUint8(src, i)
    a.Lifetime = getUint16(src, i)
    s, ok := DecodeSock(src, i)
    if !ok {
        return a, false
    }
    a.Sock = s
    if len(src)-*i >= 2 {
        a.AuthScheme = getUint16(src, i)
        if len(src)-*i >= 2 {
            tlen := int(getUint16(src, i))
            if len(src)-*i >= tlen {
                if tlen > 0 {
                    a.AuthToken = make([]byte, tlen)
                    copy(a.AuthToken, src[*i:*i+tlen])
                    *i += tlen
                }
            }
        }
    }
    return a, true
}

func EncodeRegisterAck(c Common, a RegisterAck, dst []byte) int {
    i := 0
    i += EncodeCommon(c, dst[i:])
    putUint32(dst, &i, a.Cookie)
    putUint32(dst, &i, a.DevAddr.NetAddr)
    putUint8(dst, &i, a.DevAddr.Bitlen)
    putUint16(dst, &i, a.Lifetime)
    i += EncodeSock(a.Sock, dst[i:])
    return i
}

func DecodeRegisterAck(src []byte, i *int) (RegisterAck, bool) {
    a := RegisterAck{}
    if len(src)-*i < 4+5+2+4+16 {
        return a, false
    }
    a.Cookie = getUint32(src, i)
    a.DevAddr.NetAddr = getUint32(src, i)
    a.DevAddr.Bitlen = getUint8(src, i)
    a.Lifetime = getUint16(src, i)
    s, ok := DecodeSock(src, i)
    if !ok { return a, false }
    a.Sock = s
    return a, true
}

func EncodeUnregisterSuper(c Common, u UnregisterSuper, dst []byte) int {
    i := 0
    i += EncodeCommon(c, dst[i:])
    putUint32(dst, &i, u.Cookie)
    copy(dst[i:i+6], u.EdgeMac[:])
    i += 6
    putUint16(dst, &i, u.AuthScheme)
    putUint16(dst, &i, uint16(len(u.AuthToken)))
    copy(dst[i:i+len(u.AuthToken)], u.AuthToken)
    i += len(u.AuthToken)
    return i
}

func DecodeUnregisterSuper(src []byte, i *int) (UnregisterSuper, bool) {
    u := UnregisterSuper{}
    if len(src)-*i < 4+6 { return u, false }
    u.Cookie = getUint32(src, i)
    copy(u.EdgeMac[:], src[*i:*i+6])
    *i += 6
    if len(src)-*i >= 2 {
        u.AuthScheme = getUint16(src, i)
        if len(src)-*i >= 2 {
            tlen := int(getUint16(src, i))
            if len(src)-*i >= tlen {
                if tlen > 0 {
                    u.AuthToken = make([]byte, tlen)
                    copy(u.AuthToken, src[*i:*i+tlen])
                    *i += tlen
                }
            }
        }
    }
    return u, true
}

func EncodeRegisterSuperNak(c Common, n RegisterSuperNak, dst []byte) int {
    i := 0
    i += EncodeCommon(c, dst[i:])
    putUint32(dst, &i, n.Cookie)
    putUint16(dst, &i, n.Reason)
    return i
}

func DecodeRegisterSuperNak(src []byte, i *int) (RegisterSuperNak, bool) {
    n := RegisterSuperNak{}
    if len(src)-*i < 4+2 { return n, false }
    n.Cookie = getUint32(src, i)
    n.Reason = getUint16(src, i)
    return n, true
}

type Packet struct {
    SrcMac      Mac
    DstMac      Mac
    Sock        Sock
    Transform   uint8
    Compression uint8
    Payload     []byte
}

const (
    TransformNull     = 1
    TransformAES      = 3
    TransformChaCha20 = 4
)

const (
    CompressionNone = 1
    CompressionZstd = 3
)

func EncodePacket(c Common, p Packet, payload []byte, dst []byte) int {
    i := 0
    i += EncodeCommon(c, dst[i:])
    copy(dst[i:i+6], p.SrcMac[:])
    i += 6
    copy(dst[i:i+6], p.DstMac[:])
    i += 6
    i += EncodeSock(p.Sock, dst[i:])
    putUint8(dst, &i, p.Transform)
    putUint8(dst, &i, p.Compression)
    copy(dst[i:], payload)
    i += len(payload)
    return i
}

func DecodePacket(src []byte, i *int, payload []byte) (Packet, bool, int) {
    p := Packet{}
    if len(src)-*i < 6+6+2+2+4 {
        return p, false, 0
    }
    copy(p.SrcMac[:], src[*i:*i+6])
    *i += 6
    copy(p.DstMac[:], src[*i:*i+6])
    *i += 6
    s, ok := DecodeSock(src, i)
    if !ok {
        return p, false, 0
    }
    p.Sock = s
    t1 := getUint8(src, i)
    c1 := getUint8(src, i)
    // Normalize order: some clients send compression first then transform
    if t1 == TransformNull || t1 == TransformAES || t1 == TransformChaCha20 {
        p.Transform = t1
        p.Compression = c1
    } else {
        p.Transform = c1
        p.Compression = t1
    }
    n := copy(payload, src[*i:])
    p.Payload = payload[:n]
    *i += n
    return p, true, n
}

type QueryPeer struct {
    AFlags    uint16
    SrcMac    Mac
    Sock      Sock
    TargetMac Mac
}

func EncodeQueryPeer(c Common, q QueryPeer, dst []byte) int {
    i := 0
    i += EncodeCommon(c, dst[i:])
    putUint16(dst, &i, q.AFlags)
    copy(dst[i:i+6], q.SrcMac[:])
    i += 6
    i += EncodeSock(q.Sock, dst[i:])
    copy(dst[i:i+6], q.TargetMac[:])
    i += 6
    return i
}

func DecodeQueryPeer(src []byte, i *int) (QueryPeer, bool) {
    q := QueryPeer{}
    if len(src)-*i < 2+6+4+16+6 {
        return q, false
    }
    q.AFlags = getUint16(src, i)
    copy(q.SrcMac[:], src[*i:*i+6])
    *i += 6
    s, ok := DecodeSock(src, i)
    if !ok {
        return q, false
    }
    q.Sock = s
    copy(q.TargetMac[:], src[*i:*i+6])
    *i += 6
    return q, true
}

type PeerInfo struct {
    AFlags        uint16
    SrcMac        Mac
    Mac           Mac
    Sock          Sock
    PreferredSock Sock
    Load          uint32
}

func EncodePeerInfo(c Common, p PeerInfo, dst []byte) int {
    i := 0
    i += EncodeCommon(c, dst[i:])
    putUint16(dst, &i, p.AFlags)
    copy(dst[i:i+6], p.SrcMac[:])
    i += 6
    copy(dst[i:i+6], p.Mac[:])
    i += 6
    i += EncodeSock(p.Sock, dst[i:])
    i += EncodeSock(p.PreferredSock, dst[i:])
    putUint32(dst, &i, p.Load)
    return i
}

func DecodePeerInfo(src []byte, i *int) (PeerInfo, bool) {
    p := PeerInfo{}
    if len(src)-*i < 2+6+6 {
        return p, false
    }
    p.AFlags = getUint16(src, i)
    copy(p.SrcMac[:], src[*i:*i+6])
    *i += 6
    copy(p.Mac[:], src[*i:*i+6])
    *i += 6
    s, ok := DecodeSock(src, i)
    if !ok {
        return p, false
    }
    p.Sock = s
    ps, ok := DecodeSock(src, i)
    if !ok {
        return p, false
    }
    p.PreferredSock = ps
    if len(src)-*i < 4 { return p, false }
    p.Load = getUint32(src, i)
    return p, true
}
