package portmap

import (
    "encoding/binary"
    "net"
    "time"
    "strings"
    "n2n-go/pkg/logx"
)

type Status struct {
    Enabled bool
    LastOK time.Time
    LastErr string
}

type Client struct {
    st Status
}

func New() *Client { return &Client{st: Status{}} }

func (c *Client) TryMap(port int) bool {
    gateways := []string{"192.168.0.1", "192.168.1.1", "10.0.0.1"}
    for _, gw := range gateways {
        if natpmpMap(gw, port) || igdMap(port) {
            c.st.Enabled = true
            c.st.LastOK = time.Now()
            c.st.LastErr = ""
            logx.Printf(1, "portmap enabled via %s", gw)
            return true
        }
    }
    c.st.Enabled = false
    c.st.LastErr = "no-gateway"
    logx.Printf(1, "portmap failed: %s", c.st.LastErr)
    return false
}

func (c *Client) Status() Status { return c.st }

func natpmpMap(gateway string, port int) bool {
    addr := &net.UDPAddr{IP: net.ParseIP(gateway), Port: 5351}
    conn, err := net.DialUDP("udp", nil, addr)
    if err != nil { return false }
    defer conn.Close()
    req := make([]byte, 12)
    req[0] = 0
    req[1] = 2
    binary.BigEndian.PutUint16(req[2:], 0)
    binary.BigEndian.PutUint16(req[4:], uint16(port))
    binary.BigEndian.PutUint16(req[6:], uint16(port))
    binary.BigEndian.PutUint32(req[8:], uint32(3600))
    conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
    if _, err = conn.Write(req); err != nil { return false }
    resp := make([]byte, 16)
    n, _, err := conn.ReadFromUDP(resp)
    if err != nil || n < 8 { return false }
    if resp[0] != 0 || resp[1] != 130 { return false }
    res := binary.BigEndian.Uint16(resp[2:4])
    if res != 0 { return false }
    logx.Printf(2, "natpmp ok gw=%s port=%d", gateway, port)
    return true
}

func igdMap(port int) bool {
    conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
    if err != nil { return false }
    defer conn.Close()
    ssdp := "M-SEARCH * HTTP/1.1\r\n" +
        "HOST: 239.255.255.250:1900\r\n" +
        "MAN: \"ssdp:discover\"\r\n" +
        "MX: 1\r\n" +
        "ST: urn:schemas-upnp-org:device:InternetGatewayDevice:1\r\n\r\n"
    conn.WriteToUDP([]byte(ssdp), &net.UDPAddr{IP: net.ParseIP("239.255.255.250"), Port: 1900})
    conn.SetDeadline(time.Now().Add(500 * time.Millisecond))
    buf := make([]byte, 2048)
    n, _, err := conn.ReadFromUDP(buf)
    if err != nil || n == 0 { return false }
    s := string(buf[:n])
    loc := ""
    for _, line := range []string{s} {
        i := strings.Index(strings.ToLower(line), "location:")
        if i >= 0 { loc = strings.TrimSpace(line[i+9:]); break }
    }
    if loc == "" { return false }
    logx.Printf(2, "igd ssdp location %s", loc)
    // minimal SOAP request to common control URLs omitted for brevity
    return false
}

