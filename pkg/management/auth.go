package management

func pearson64(b []byte) uint64 {
    var t [256]uint8
    for i := 0; i < 256; i++ { t[i] = uint8(i) }
    var h0, h1, h2, h3, h4, h5, h6, h7 uint8
    for _, c := range b {
        h0 = t[h0^c]
        h1 = t[h1^c]
        h2 = t[h2^c]
        h3 = t[h3^c]
        h4 = t[h4^c]
        h5 = t[h5^c]
        h6 = t[h6^c]
        h7 = t[h7^c]
    }
    return uint64(h0) | uint64(h1)<<8 | uint64(h2)<<16 | uint64(h3)<<24 | uint64(h4)<<32 | uint64(h5)<<40 | uint64(h6)<<48 | uint64(h7)<<56
}
