package logx

import (
    "fmt"
    "os"
)

var Level int

func SetLevel(l int) { Level = l }

func InitFromEnv() {
    v := os.Getenv("N2N_TRACE")
    if v == "" { return }
    var x int
    fmt.Sscanf(v, "%d", &x)
    Level = x
}

func Printf(l int, format string, v ...any) {
    if Level >= l {
        fmt.Printf(format+"\n", v...)
    }
}
