package main

import (
    crand "crypto/rand"
    "crypto/sha256"
    "crypto/hkdf"
    "flag"
    "fmt"
    "math/rand"
    "net"
    "io"
    "os"
    "time"
    "n2n-go/pkg/tap"
    "n2n-go/pkg/transport"
    "n2n-go/pkg/wire"
    "n2n-go/pkg/crypto"
    "n2n-go/pkg/compress"
    "n2n-go/pkg/portmap"
    "n2n-go/pkg/management"
    "n2n-go/pkg/logx"
)

func main() {
    var dev string
    var lport int
    var bind string
    var snAddr string
    var community string
    var key string
    var cipher string
    var cmpr string
    var secure bool
    var mport int
    var v int
    flag.StringVar(&dev, "dev", "tap0", "tap device name")
    flag.StringVar(&bind, "bind", "0.0.0.0", "bind address")
    flag.IntVar(&lport, "p", 7655, "local UDP port")
    flag.StringVar(&snAddr, "l", "127.0.0.1:7654", "supernode host:port")
    flag.StringVar(&community, "c", "community", "community name")
    flag.StringVar(&key, "k", "", "encryption key")
    flag.StringVar(&cipher, "A", "null", "cipher: aes|chacha|null")
    flag.StringVar(&cmpr, "z", "none", "compression: none|zstd")
    flag.BoolVar(&secure, "H", false, "secure header mode")
    flag.IntVar(&mport, "t", 5644, "management UDP port")
    flag.IntVar(&v, "v", 0, "verbose level")
    flag.Parse()
    os.Setenv("N2N_EDGE_TRACE", fmt.Sprintf("%d", v))
    os.Setenv("N2N_TRACE", fmt.Sprintf("%d", v))
    logx.InitFromEnv()
    logx.Printf(1, "edge start dev=%s bind=%s lport=%d sn=%s", dev, bind, lport, snAddr)

    d, err := tap.Open(dev, 1500)
    if err != nil {
        fmt.Println("tap open error:", err)
        os.Exit(2)
    }
    defer d.Close()
    logx.Printf(1, "tap opened name=%s", dev)

    udp, err := transport.ListenUDP(bind, lport)
    if err != nil {
        fmt.Println("udp open error:", err)
        os.Exit(2)
    }
    defer udp.Close()
    logx.Printf(1, "udp opened bind=%s lport=%d", bind, lport)
    pm := portmap.New()
    pm.TryMap(lport)
    traceLevel := v
    mgmt := &management.Server{Password: "", KeepRunning: nil, TraceLevel: &traceLevel}
    mgmt.HandleFunc = func(method string, params []string) []map[string]any {
        var rows []map[string]any
        switch method {
        case "portmap.status":
            st := pm.Status()
            rows = append(rows, map[string]any{"enabled": st.Enabled, "last_ok": st.LastOK.Unix(), "last_err": st.LastErr})
        case "portmap.refresh":
            ok := pm.TryMap(lport)
            st := pm.Status()
            rows = append(rows, map[string]any{"ok": ok, "enabled": st.Enabled, "last_err": st.LastErr})
        case "tap.configure":
            var name, ip, mask string
            var metric int
            if len(params) >= 1 { name = params[0] }
            if len(params) >= 3 { ip = params[1]; mask = params[2] }
            if len(params) >= 4 { fmt.Sscanf(params[3], "%d", &metric) }
            tap.ConfigureIPv4(name, ip, mask, metric)
            rows = append(rows, map[string]any{"ok": true})
        }
        return rows
    }
    mgmtConn, _ := mgmt.Listen("127.0.0.1", mport)
    stopCh := make(chan struct{})
    go mgmt.Handle(mgmtConn, stopCh)

    raddr, err := net.ResolveUDPAddr("udp", snAddr)
    if err != nil {
        fmt.Println("resolve supernode error:", err)
        os.Exit(2)
    }

    regc := wire.Common{TTL: 2, PC: wire.MsgRegisterSuper, Flags: 0}
    copy(regc.Community[:], []byte(community))
    reg := wire.RegisterSuper{}
    reg.Cookie = uint32(rand.Uint32())
    reg.Sock.Family = 2
    reg.Sock.Type = 2
    reg.Sock.Port = uint16(lport)
    if ip := net.ParseIP(bind).To4(); ip != nil {
        copy(reg.Sock.AddrV4[:], ip)
    }
    reg.DevAddr.Bitlen = 24
    reg.KeyTime = uint32(time.Now().Unix())
    b := make([]byte, 256)
    rl := wire.EncodeRegisterSuper(regc, reg, b)
    udp.WriteTo(b[:rl], raddr)
    logx.Printf(1, "register sent cookie=%d community=%s", reg.Cookie, community)

    lastReg := time.Now()
    regInterval := 20

    tapBuf := make([]byte, 2048)
    var aead crypto.AEAD
    var codec compress.Codec = compress.Null{}
    if cmpr == "zstd" {
        z, err := compress.NewZstd()
        if err == nil { codec = z }
    }
    var mk []byte
    if key != "" {
        mk = make([]byte, 32)
        salt := []byte(community)
        rdr := hkdf.New(sha256.New, []byte(key), salt, nil)
        io.ReadFull(rdr, mk)
    }
    if cipher == "aes" && len(mk) == 32 {
        a, err := crypto.NewAESGCM(mk)
        if err == nil { aead = a }
    }
    if cipher == "chacha" && len(mk) == 32 {
        a, err := crypto.NewChaCha(mk)
        if err == nil { aead = a }
    }
    var lastSrcMac wire.Mac
    go func() {
        for {
            n, err := d.Read(tapBuf)
            if err != nil {
                return
            }
            payload := tapBuf[:n]
            cdata, _ := codec.Compress(nil, payload)
            pc := wire.Common{TTL: 2, PC: wire.MsgPacket, Flags: 0}
            copy(pc.Community[:], []byte(community))
            pkt := wire.Packet{}
            if n >= 14 {
                copy(pkt.DstMac[:], payload[0:6])
                copy(pkt.SrcMac[:], payload[6:12])
                lastSrcMac = pkt.SrcMac
            }
            pkt.Sock = reg.Sock
            pkt.Transform = wire.TransformNull
            pkt.Compression = wire.CompressionNone
            if aead != nil {
                if cipher == "aes" { pkt.Transform = wire.TransformAES }
                if cipher == "chacha" { pkt.Transform = wire.TransformChaCha20 }
            }
            if cmpr == "zstd" { pkt.Compression = wire.CompressionZstd }
            if aead != nil && secure {
                nonce := make([]byte, aead.NonceSize())
                _, _ = crand.Read(nonce)
                inner := make([]byte, 2+len(cdata))
                inner[0] = pkt.Compression
                inner[1] = pkt.Transform
                copy(inner[2:], cdata)
                ad := make([]byte, 64)
                ai := 0
                ai += wire.EncodeCommon(pc, ad[ai:])
                copy(ad[ai:ai+6], pkt.SrcMac[:])
                ai += 6
                copy(ad[ai:ai+6], pkt.DstMac[:])
                ai += 6
                ai += wire.EncodeSock(pkt.Sock, ad[ai:])
                ad[ai] = wire.CompressionNone
                ai++
                ad[ai] = wire.TransformNull
                ai++
                cp := aead.Seal(nil, nonce, inner, ad[:ai])
                cdata = append(nonce, cp...)
                pkt.Compression = wire.CompressionNone
                pkt.Transform = wire.TransformNull
            } else if aead != nil {
                nonce := make([]byte, aead.NonceSize())
                _, _ = crand.Read(nonce)
                ad := make([]byte, 64)
                ai := 0
                ai += wire.EncodeCommon(pc, ad[ai:])
                copy(ad[ai:ai+6], pkt.SrcMac[:])
                ai += 6
                copy(ad[ai:ai+6], pkt.DstMac[:])
                ai += 6
                ai += wire.EncodeSock(pkt.Sock, ad[ai:])
                ad[ai] = pkt.Compression
                ai++
                ad[ai] = pkt.Transform
                ai++
                cp := aead.Seal(nil, nonce, cdata, ad[:ai])
                cdata = append(nonce, cp...)
            }
            out := make([]byte, 4096)
            m := wire.EncodePacket(pc, pkt, cdata, out)
            udp.WriteTo(out[:m], raddr)
            logx.Printf(2, "packet out src=%02x:%02x:%02x:%02x:%02x:%02x dst=%02x:%02x:%02x:%02x:%02x:%02x bytes=%d", pkt.SrcMac[0], pkt.SrcMac[1], pkt.SrcMac[2], pkt.SrcMac[3], pkt.SrcMac[4], pkt.SrcMac[5], pkt.DstMac[0], pkt.DstMac[1], pkt.DstMac[2], pkt.DstMac[3], pkt.DstMac[4], pkt.DstMac[5], n)
        }
    }()

    rbuf := make([]byte, 2048)
    for {
        udp.Conn.SetReadDeadline(time.Now().Add(time.Second))
        n, _, err := udp.Read(rbuf)
        if err != nil {
            if ne, ok := err.(net.Error); ok && ne.Timeout() {
                if time.Since(lastReg) >= time.Duration(regInterval)*time.Second {
                    reg.Cookie = uint32(rand.Uint32())
                    rl = wire.EncodeRegisterSuper(regc, reg, b)
                    udp.WriteTo(b[:rl], raddr)
                    lastReg = time.Now()
                }
                continue
            }
            break
        }
        i := 0
        c, ok := wire.DecodeCommon(rbuf[:n], &i)
        if !ok {
            continue
        }
        if c.PC == wire.MsgRegisterSuperAck {
            _, aok := wire.DecodeRegisterSuperAck(rbuf[:n], &i)
            if !aok {
                continue
            }
            lastReg = time.Now()
            logx.Printf(1, "register ack received")
            qc := wire.Common{TTL: 2, PC: wire.MsgQueryPeer, Flags: 0}
            copy(qc.Community[:], []byte(community))
            q := wire.QueryPeer{}
            q.AFlags = 0
            copy(q.SrcMac[:], lastSrcMac[:])
            q.Sock = reg.Sock
            out := make([]byte, 128)
            l := wire.EncodeQueryPeer(qc, q, out)
            udp.WriteTo(out[:l], raddr)
            logx.Printf(1, "query peer sent src=%02x:%02x:%02x:%02x:%02x:%02x", lastSrcMac[0], lastSrcMac[1], lastSrcMac[2], lastSrcMac[3], lastSrcMac[4], lastSrcMac[5])
            continue
        }
        if c.PC == wire.MsgPeerInfo {
            _, pok := wire.DecodePeerInfo(rbuf[:n], &i)
            if !pok {
                continue
            }
            logx.Printf(1, "peer info received")
            continue
        }
        if c.PC == wire.MsgPacket {
            p := make([]byte, 4096)
            pkt, ok, _ := wire.DecodePacket(rbuf[:n], &i, p)
            if !ok { continue }
            if aead == nil && (pkt.Transform == wire.TransformAES || pkt.Transform == wire.TransformChaCha20) { continue }
            data := pkt.Payload
            if aead != nil && secure {
                nonceSize := aead.NonceSize()
                if len(data) < nonceSize { continue }
                nonce := data[:nonceSize]
                body := data[nonceSize:]
                ad := make([]byte, 64)
                ai := 0
                ai += wire.EncodeCommon(c, ad[ai:])
                copy(ad[ai:ai+6], pkt.SrcMac[:])
                ai += 6
                copy(ad[ai:ai+6], pkt.DstMac[:])
                ai += 6
                ai += wire.EncodeSock(pkt.Sock, ad[ai:])
                ad[ai] = wire.CompressionNone
                ai++
                ad[ai] = wire.TransformNull
                ai++
                dec, err := aead.Open(nil, nonce, body, ad[:ai])
                if err == nil { data = dec } else { continue }
                if len(data) < 2 { continue }
                ccode := data[0]
                tcode := data[1]
                data = data[2:]
                if tcode == wire.TransformAES || tcode == wire.TransformChaCha20 {
                    if aead == nil { continue }
                }
                if ccode == wire.CompressionZstd && codec == compress.Null{} {
                }
            } else if aead != nil {
                nonceSize := aead.NonceSize()
                if len(data) < nonceSize { continue }
                nonce := data[:nonceSize]
                body := data[nonceSize:]
                ad := make([]byte, 64)
                ai := 0
                ai += wire.EncodeCommon(c, ad[ai:])
                copy(ad[ai:ai+6], pkt.SrcMac[:])
                ai += 6
                copy(ad[ai:ai+6], pkt.DstMac[:])
                ai += 6
                ai += wire.EncodeSock(pkt.Sock, ad[ai:])
                ad[ai] = pkt.Compression
                ai++
                ad[ai] = pkt.Transform
                ai++
                dec, err := aead.Open(nil, nonce, body, ad[:ai])
                if err == nil { data = dec } else { continue }
            }
            dec, _ := codec.Decompress(nil, data)
            if len(dec) > 0 { d.Write(dec) }
            logx.Printf(2, "packet in bytes=%d", len(dec))
            continue
        }
    }
}
