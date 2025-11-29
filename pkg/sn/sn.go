package sn

import (
    "bytes"
    "fmt"
    "net"
    "os"
    "time"
    "n2n-go/pkg/management"
    "n2n-go/pkg/transport"
    "n2n-go/pkg/wire"
    "n2n-go/pkg/logx"
)

type addrPool struct {
    NetAddr uint32
    Bitlen  uint8
    next    uint32
    lifetime time.Duration
}

type allocInfo struct {
    ip        uint32
    expires   time.Time
    community string
}

type state struct {
    peers map[[6]byte]*net.UDPAddr
    alloc map[[6]byte]allocInfo
    pools map[string]*addrPool
}

func Run(bind string, lport int, mport int) error {
    keepRunning := true
    traceLevel := 0
    if tv := os.Getenv("N2N_SN_TRACE"); tv != "" { var x int; fmt.Sscanf(tv, "%d", &x); traceLevel = x }
    logx.InitFromEnv()
    s := &state{peers: map[[6]byte]*net.UDPAddr{}, alloc: map[[6]byte]allocInfo{}, pools: map[string]*addrPool{}}
    logf := func(l int, format string, v ...any) {
        if traceLevel >= l { fmt.Printf(format+"\n", v...) }
    }

    mainUDP, err := transport.ListenUDP(bind, lport)
    if err != nil {
        fmt.Println("failed to open main socket", err)
        os.Exit(2)
    }
    mgmt := &management.Server{KeepRunning: &keepRunning, TraceLevel: &traceLevel, Events: make(chan management.MgmtEvent, 16)}
    mgmtConn, err := mgmt.Listen("127.0.0.1", mport)
    if err != nil {
        fmt.Println("failed to open management socket", err)
        os.Exit(2)
    }
    stopCh := make(chan struct{})
    mgmt.HandleFunc = func(method string, params []string) []map[string]any {
        var rows []map[string]any
        switch method {
        case "pool.list":
            for k, p := range s.pools {
                rows = append(rows, map[string]any{"community": k, "netaddr": p.NetAddr, "bitlen": p.Bitlen, "lifetime": int(p.lifetime.Seconds())})
            }
        case "pool.set":
            if len(params) >= 4 {
                comm := params[0]
                ip := parseIPv4(params[1])
                var bl int
                fmt.Sscanf(params[2], "%d", &bl)
                var lt int
                fmt.Sscanf(params[3], "%d", &lt)
                s.pools[comm] = &addrPool{NetAddr: ip, Bitlen: uint8(bl), next: 10, lifetime: time.Duration(lt) * time.Second}
                rows = append(rows, map[string]any{"ok": true})
            } else {
                rows = append(rows, map[string]any{"ok": false})
            }
        case "lease.list":
            for mac, ai := range s.alloc {
                rows = append(rows, map[string]any{"mac": mac, "ip": ai.ip, "expires": ai.expires.Unix(), "community": ai.community})
            }
        case "lease.reserve":
            if len(params) >= 3 {
                mac := parseMAC(params[0])
                comm := params[1]
                ip := parseIPv4(params[2])
                s.alloc[mac] = allocInfo{ip: ip, expires: time.Now().Add(60 * time.Second), community: comm}
                rows = append(rows, map[string]any{"ok": true})
            } else {
                rows = append(rows, map[string]any{"ok": false})
            }
        case "lease.release":
            if len(params) >= 1 {
                mac := parseMAC(params[0])
                delete(s.alloc, mac)
                rows = append(rows, map[string]any{"ok": true})
            } else {
                rows = append(rows, map[string]any{"ok": false})
            }
        }
        return rows
    }
    go mgmt.Handle(mgmtConn, stopCh)
    logf(1, "management listening 127.0.0.1:%d", mport)

    buf := make([]byte, 2048)
    // sweeper for expired leases
    go func() {
        for keepRunning {
            time.Sleep(5 * time.Second)
            now := time.Now()
            for mac, ai := range s.alloc {
                if now.After(ai.expires) {
                    delete(s.alloc, mac)
                    delete(s.peers, mac)
                    if mgmt.Events != nil { mgmt.Events <- management.MgmtEvent{Topic: "lease", Row: map[string]any{"event": "expired", "mac": mac, "ip": ai.ip}} }
                }
            }
        }
    }()
    for keepRunning {
        mainUDP.Conn.SetReadDeadline(time.Now().Add(time.Second))
        n, addr, err := mainUDP.Read(buf)
        if err != nil {
            if _, ok := err.(net.Error); ok {
                select {
                case <-stopCh:
                    keepRunning = false
                default:
                }
                continue
            }
            break
        }
        logf(1, "recv %d bytes from %s:%d", n, addr.IP.String(), addr.Port)
        i := 0
        c, ok := wire.DecodeCommon(buf[:n], &i)
        if !ok { logf(1, "bad common header ver=%d len=%d", int(buf[0]), n); continue }
        logf(1, "pc=%d flags=%d ttl=%d community=%s", int(c.PC), int(c.Flags), int(c.TTL), string(bytes.TrimRight(c.Community[:], "\x00")))
        typ := c.PC
        if typ == 0 { typ = uint8(c.Flags & 0x1f) }
        if typ == wire.MsgRegisterSuper {
            r, rok := wire.DecodeRegisterSuper(buf[:n], &i)
            if !rok { logf(1, "register decode failed flags=%d", c.Flags); continue }
            if r.EdgeMac != (wire.Mac{}) { s.peers[r.EdgeMac] = addr; if mgmt.Events != nil { mgmt.Events <- management.MgmtEvent{Topic: "peer", Row: map[string]any{"event": "up"}} } }
            comm := string(bytes.TrimRight(c.Community[:], "\x00"))
            pool := s.pools[comm]
            if pool == nil { pool = &addrPool{NetAddr: 0x0a000000, Bitlen: 24, next: 10, lifetime: 60 * time.Second}; s.pools[comm] = pool }
            ai := s.alloc[r.EdgeMac]
            if ai.ip == 0 {
                ip := pool.NetAddr | (pool.next & 0xff)
                pool.next++
                ai = allocInfo{ip: ip, expires: time.Now().Add(pool.lifetime), community: comm}
                s.alloc[r.EdgeMac] = ai
            } else {
                ai.expires = time.Now().Add(pool.lifetime)
                s.alloc[r.EdgeMac] = ai
            }
            ipstr := fmt.Sprintf("%d.%d.%d.%d", (ai.ip>>24)&0xff, (ai.ip>>16)&0xff, (ai.ip>>8)&0xff, ai.ip&0xff)
            logf(1, "register mac=%02x:%02x:%02x:%02x:%02x:%02x community=%s ip=%s", r.EdgeMac[0], r.EdgeMac[1], r.EdgeMac[2], r.EdgeMac[3], r.EdgeMac[4], r.EdgeMac[5], comm, ipstr)
            ackc := wire.Common{TTL: 2, PC: wire.MsgRegisterSuperAck, Flags: 0}
            copy(ackc.Community[:], c.Community[:])
            a := wire.RegisterSuperAck{Cookie: r.Cookie, Lifetime: 60}
            copy(a.SrcMac[:], r.EdgeMac[:])
            a.DevAddr.NetAddr = ai.ip
            a.DevAddr.Bitlen = pool.Bitlen
            a.Sock.Family = 2
            a.Sock.Type = 2
            a.Sock.Port = uint16(mainUDP.Conn.LocalAddr().(*net.UDPAddr).Port)
            b := make([]byte, 256)
            l := wire.EncodeRegisterSuperAck(ackc, a, b)
            mainUDP.WriteTo(b[:l], addr)
            continue
        }
        if typ == wire.MsgUnregisterSuper {
            u, uok := wire.DecodeUnregisterSuper(buf[:n], &i)
            if !uok { continue }
            delete(s.peers, u.EdgeMac)
            delete(s.alloc, u.EdgeMac)
            if mgmt.Events != nil { mgmt.Events <- management.MgmtEvent{Topic: "peer", Row: map[string]any{"event": "down"}} }
            logf(1, "unregister mac=%02x:%02x:%02x:%02x:%02x:%02x", u.EdgeMac[0], u.EdgeMac[1], u.EdgeMac[2], u.EdgeMac[3], u.EdgeMac[4], u.EdgeMac[5])
            continue
        }
        if typ == wire.MsgQueryPeer {
            q, qok := wire.DecodeQueryPeer(buf[:n], &i)
            if !qok { continue }
            rc := wire.Common{TTL: 2, PC: wire.MsgPeerInfo, Flags: 0}
            copy(rc.Community[:], c.Community[:])
            pi := wire.PeerInfo{}
            pi.AFlags = 0
            copy(pi.SrcMac[:], q.SrcMac[:])
            copy(pi.Mac[:], q.TargetMac[:])
            pi.Sock.Family = 2
            pi.Sock.Type = 2
            pi.Sock.Port = uint16(mainUDP.Conn.LocalAddr().(*net.UDPAddr).Port)
            pi.PreferredSock = pi.Sock
            if peerAddr, ok := s.peers[q.TargetMac]; ok {
                pi.PreferredSock.Family = 2
                pi.PreferredSock.Type = 2
                pi.PreferredSock.Port = uint16(peerAddr.Port)
                copy(pi.PreferredSock.AddrV4[:], peerAddr.IP.To4())
                logf(1, "query src=%02x:%02x:%02x:%02x:%02x:%02x target found=%02x:%02x:%02x:%02x:%02x:%02x", q.SrcMac[0], q.SrcMac[1], q.SrcMac[2], q.SrcMac[3], q.SrcMac[4], q.SrcMac[5], q.TargetMac[0], q.TargetMac[1], q.TargetMac[2], q.TargetMac[3], q.TargetMac[4], q.TargetMac[5])
            } else {
                logf(1, "query src=%02x:%02x:%02x:%02x:%02x:%02x target missing=%02x:%02x:%02x:%02x:%02x:%02x", q.SrcMac[0], q.SrcMac[1], q.SrcMac[2], q.SrcMac[3], q.SrcMac[4], q.SrcMac[5], q.TargetMac[0], q.TargetMac[1], q.TargetMac[2], q.TargetMac[3], q.TargetMac[4], q.TargetMac[5])
            }
            out := make([]byte, 256)
            l := wire.EncodePeerInfo(rc, pi, out)
            mainUDP.WriteTo(out[:l], addr)
            continue
        }
        if typ == wire.MsgPacket {
            j := i
            if n-j < 12 { continue }
            var src [6]byte
            var dst [6]byte
            copy(src[:], buf[j:j+6])
            copy(dst[:], buf[j+6:j+12])
            if peerAddr, ok := s.peers[dst]; ok {
                mainUDP.WriteTo(buf[:n], peerAddr)
                logf(2, "forward mac=%02x:%02x:%02x:%02x:%02x:%02x -> %02x:%02x:%02x:%02x:%02x:%02x bytes=%d", src[0], src[1], src[2], src[3], src[4], src[5], dst[0], dst[1], dst[2], dst[3], dst[4], dst[5], n-(j+12))
            } else {
                mainUDP.WriteTo(buf[:n], addr)
                logf(2, "echo mac=%02x:%02x:%02x:%02x:%02x:%02x bytes=%d", src[0], src[1], src[2], src[3], src[4], src[5], n-(j+12))
            }
            continue
        }
        logf(1, "unhandled pc=%d (derived=%d)", int(c.PC), int(typ))
    }
    return nil
}

func parseIPv4(s string) uint32 {
    ip := net.ParseIP(s).To4()
    if ip == nil { return 0 }
    return uint32(ip[0])<<24 | uint32(ip[1])<<16 | uint32(ip[2])<<8 | uint32(ip[3])
}

func parseMAC(s string) [6]byte {
    var m [6]byte
    var b0, b1, b2, b3, b4, b5 int
    fmt.Sscanf(s, "%x:%x:%x:%x:%x:%x", &b0, &b1, &b2, &b3, &b4, &b5)
    m[0] = byte(b0)
    m[1] = byte(b1)
    m[2] = byte(b2)
    m[3] = byte(b3)
    m[4] = byte(b4)
    m[5] = byte(b5)
    return m
}

