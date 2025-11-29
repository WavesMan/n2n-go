package management

import (
    "encoding/json"
    "fmt"
    "net"
    "strings"
    "n2n-go/pkg/logx"
)

type Server struct {
    Password string
    KeepRunning *bool
    TraceLevel *int
    subscribers map[string]*net.UDPAddr
    Events chan MgmtEvent
    HandleFunc func(method string, params []string) []map[string]any
}

type replyRow map[string]any

func (s *Server) Listen(addr string, port int) (*net.UDPConn, error) {
    udpAddr := &net.UDPAddr{IP: net.ParseIP(addr), Port: port}
    return net.ListenUDP("udp", udpAddr)
}

func genJSONRow(tag string, kv replyRow) []byte {
    m := map[string]any{"_tag": tag, "_type": "row"}
    for k, v := range kv {
        m[k] = v
    }
    b, _ := json.Marshal(m)
    return append(b, '\n')
}

func genJSONErr(tag, msg string) []byte {
    m := map[string]any{"_tag": tag, "_type": "error", "error": msg}
    b, _ := json.Marshal(m)
    return append(b, '\n')
}

func (s *Server) Handle(conn *net.UDPConn, keepAlive chan struct{}) {
    buf := make([]byte, 2048)
    if s.subscribers == nil { s.subscribers = make(map[string]*net.UDPAddr) }
    if s.Events != nil {
        go func() {
            for ev := range s.Events {
                if addr, ok := s.subscribers[ev.Topic]; ok {
                    m := map[string]any{"_type": "event"}
                    for k, v := range ev.Row { m[k] = v }
                    b, _ := json.Marshal(m)
                    conn.WriteToUDP(append(b, '\n'), addr)
                    logx.Printf(2, "mgmt event topic=%s", ev.Topic)
                }
            }
        }()
    }
    for {
        n, addr, err := conn.ReadFromUDP(buf)
        if err != nil {
            return
        }
        line := strings.TrimSpace(string(buf[:n]))
        logx.Printf(2, "mgmt recv %s", line)
        parts := strings.Fields(line)
        if len(parts) < 3 { conn.WriteToUDP(genJSONErr("mgmt", "badreq"), addr); continue }
        mtype := parts[0]
        opts := parts[1]
        method := parts[2]
        params := parts[3:]
        tag := "mgmt"
        // options: tag[:flags[:auth]]
        optFields := strings.Split(opts, ":")
        if len(optFields) >= 1 && optFields[0] != "" { tag = optFields[0] }
        var flags string
        var auth string
        if len(optFields) >= 2 { flags = optFields[1] }
        if len(optFields) >= 3 { auth = optFields[2] }
        if s.Password != "" {
            if flags == "1" && auth != "" {
                if pearson64([]byte(auth)) != pearson64([]byte(s.Password)) { conn.WriteToUDP(genJSONErr(tag, "badauth"), addr); continue }
            } else if strings.HasPrefix(line, "auth ") {
                // backward compatibility: auth <user> <pass> <rest>
                if len(parts) < 5 { conn.WriteToUDP(genJSONErr(tag, "badauth"), addr); continue }
                if pearson64([]byte(parts[2])) != pearson64([]byte(s.Password)) { conn.WriteToUDP(genJSONErr(tag, "badauth"), addr); continue }
                // re-parse after auth prefix
                mtype = parts[3]
                if len(parts) < 6 { conn.WriteToUDP(genJSONErr(tag, "badreq"), addr); continue }
                opts = parts[4]
                method = parts[5]
                params = parts[6:]
                optFields = strings.Split(opts, ":")
                if len(optFields) >= 1 && optFields[0] != "" { tag = optFields[0] }
            } else {
                conn.WriteToUDP(genJSONErr(tag, "unauth"), addr); continue
            }
        }
        // begin
        conn.WriteToUDP(genJSONRow(tag, replyRow{"_type": "begin", "cmd": method}), addr)
        switch method {
        case "help":
            conn.WriteToUDP(genJSONRow(tag, replyRow{"cmd": "help", "help": "stop|verbose <n>|subscribe <topic>"}), addr)
        case "stop":
            if mtype == "w" {
                if s.KeepRunning != nil { *s.KeepRunning = false }
                conn.WriteToUDP(genJSONRow(tag, replyRow{"keep_running": s.safeBool(s.KeepRunning)}), addr)
                conn.WriteToUDP(genJSONRow(tag, replyRow{"_type": "end"}), addr)
                close(keepAlive)
                logx.Printf(1, "mgmt stop")
                return
            } else {
                conn.WriteToUDP(genJSONErr(tag, "badtype"), addr)
            }
        case "verbose":
            if len(params) >= 1 {
                var v int
                fmt.Sscanf(params[0], "%d", &v)
                if s.TraceLevel != nil { *s.TraceLevel = v }
            }
            logx.Printf(1, "mgmt verbose %d", s.safeInt(s.TraceLevel))
            logx.SetLevel(s.safeInt(s.TraceLevel))
            conn.WriteToUDP(genJSONRow(tag, replyRow{"traceLevel": s.safeInt(s.TraceLevel)}), addr)
        case "subscribe":
            if mtype != "s" { conn.WriteToUDP(genJSONErr(tag, "badtype"), addr); break }
            topic := "debug"
            if len(params) >= 1 { topic = params[0] }
            s.subscribers[topic] = addr
            conn.WriteToUDP(genJSONRow(tag, replyRow{"_type": "subscribed", "topic": topic}), addr)
            logx.Printf(1, "mgmt subscribe %s", topic)
        default:
            if s.HandleFunc != nil {
                rows := s.HandleFunc(method, params)
                for _, r := range rows { conn.WriteToUDP(genJSONRow(tag, replyRow(r)), addr) }
                logx.Printf(2, "mgmt call %s", method)
            } else {
                conn.WriteToUDP(genJSONErr(tag, "unimplemented"), addr)
            }
        }
        // end
        conn.WriteToUDP(genJSONRow(tag, replyRow{"_type": "end"}), addr)
    }
}

func (s *Server) safeBool(p *bool) bool {
    if p == nil {
        return false
    }
    return *p
}

func (s *Server) safeInt(p *int) int {
    if p == nil {
        return 0
    }
    return *p
}
type MgmtEvent struct {
    Topic string
    Row map[string]any
}
