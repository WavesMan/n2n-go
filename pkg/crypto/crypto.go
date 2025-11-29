package crypto

import (
    "crypto/aes"
    "crypto/cipher"
    "golang.org/x/crypto/chacha20poly1305"
)

type AEAD interface{
    Seal(dst, nonce, plaintext, ad []byte) []byte
    Open(dst, nonce, ciphertext, ad []byte) ([]byte, error)
    NonceSize() int
}

type aesgcm struct{ aead cipher.AEAD }

func (a *aesgcm) Seal(dst, nonce, plaintext, ad []byte) []byte { return a.aead.Seal(dst, nonce, plaintext, ad) }
func (a *aesgcm) Open(dst, nonce, ciphertext, ad []byte) ([]byte, error) { return a.aead.Open(dst, nonce, ciphertext, ad) }
func (a *aesgcm) NonceSize() int { return a.aead.NonceSize() }

type chachaaead struct{ aead cipher.AEAD }

func (c *chachaaead) Seal(dst, nonce, plaintext, ad []byte) []byte { return c.aead.Seal(dst, nonce, plaintext, ad) }
func (c *chachaaead) Open(dst, nonce, ciphertext, ad []byte) ([]byte, error) { return c.aead.Open(dst, nonce, ciphertext, ad) }
func (c *chachaaead) NonceSize() int { return c.aead.NonceSize() }

func NewAESGCM(key []byte) (AEAD, error) {
    blk, err := aes.NewCipher(key)
    if err != nil { return nil, err }
    a, err := cipher.NewGCM(blk)
    if err != nil { return nil, err }
    return &aesgcm{aead: a}, nil
}

func NewChaCha(key []byte) (AEAD, error) {
    a, err := chacha20poly1305.New(key)
    if err != nil { return nil, err }
    return &chachaaead{aead: a}, nil
}

