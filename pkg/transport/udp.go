package transport

import (
    "net"
    "n2n-go/pkg/logx"
)

type UDPListener struct {
    Conn *net.UDPConn
}

func ListenUDP(addr string, port int) (*UDPListener, error) {
    udpAddr := &net.UDPAddr{IP: net.ParseIP(addr), Port: port}
    c, err := net.ListenUDP("udp", udpAddr)
    if err != nil {
        return nil, err
    }
    logx.Printf(2, "udp listen %s:%d", addr, port)
    return &UDPListener{Conn: c}, nil
}

func (l *UDPListener) Close() error {
    if l.Conn != nil {
        return l.Conn.Close()
    }
    return nil
}

func (l *UDPListener) Read(buf []byte) (int, *net.UDPAddr, error) {
    n, addr, err := l.Conn.ReadFromUDP(buf)
    return n, addr, err
}

func (l *UDPListener) WriteTo(buf []byte, addr *net.UDPAddr) (int, error) {
    return l.Conn.WriteToUDP(buf, addr)
}
