package compress

type Codec interface{
    Compress(dst, src []byte) ([]byte, error)
    Decompress(dst, src []byte) ([]byte, error)
}

type Null struct{}

func (n Null) Compress(dst, src []byte) ([]byte, error) { return append(dst[:0], src...), nil }
func (n Null) Decompress(dst, src []byte) ([]byte, error) { return append(dst[:0], src...), nil }

type Zstd struct{}

func NewZstd() (Zstd, error) { return Zstd{}, nil }

func (z Zstd) Compress(dst, src []byte) ([]byte, error) { return append(dst[:0], src...), nil }
func (z Zstd) Decompress(dst, src []byte) ([]byte, error) { return append(dst[:0], src...), nil }

