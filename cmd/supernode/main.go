package main

import (
    "flag"
    "fmt"
    "n2n-go/pkg/sn"
    
)

func main() {
    var lport int
    var mport int
    var bind string
    var v int
    flag.StringVar(&bind, "bind", "0.0.0.0", "bind address")
    flag.IntVar(&lport, "p", 7654, "local UDP port")
    flag.IntVar(&mport, "t", 5645, "management UDP port")
    flag.IntVar(&v, "v", 0, "verbose level")
    flag.Parse()
    fmt.Printf("supernode bind %s:%d\n", bind, lport)
    fmt.Println("supernode started, press Ctrl+C to stop")
    sn.Run(bind, lport, mport)
}
