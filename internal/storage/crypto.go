package storage

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
)

// MagicBytes identifies an encrypted wallet file ("BBE\x01").
var MagicBytes = [4]byte{0x42, 0x42, 0x45, 0x01}

const (
	defaultTime   uint32 = 3
	defaultMemory uint32 = 65536 // 64 MiB
	defaultPar    uint8  = 4

	// headerSize = magic(4) + time(4) + memory(4) + par(1) + salt(32) + nonce(24)
	headerSize = 4 + 4 + 4 + 1 + 32 + 24
)

type encHeader struct {
	Time   uint32
	Memory uint32
	Par    uint8
	Salt   [32]byte
	Nonce  [24]byte
}

// DeriveKey derives a 32-byte key from the given password and salt using Argon2id.
func DeriveKey(password, salt []byte, time, memory uint32, par uint8) []byte {
	return argon2.IDKey(password, salt, time, memory, par, 32)
}

// Encrypt encrypts plaintext with XChaCha20-Poly1305 using a key derived from password.
// It returns a self-contained blob with a header containing all parameters needed for decryption.
func Encrypt(plaintext, password []byte) ([]byte, error) {
	h := encHeader{
		Time:   defaultTime,
		Memory: defaultMemory,
		Par:    defaultPar,
	}
	if _, err := rand.Read(h.Salt[:]); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	if _, err := rand.Read(h.Nonce[:]); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	key := DeriveKey(password, h.Salt[:], h.Time, h.Memory, h.Par)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	ciphertext := aead.Seal(nil, h.Nonce[:], plaintext, nil)

	// Layout: magic(4) | time(4) | memory(4) | par(1) | salt(32) | nonce(24) | ciphertext
	out := make([]byte, 0, headerSize+len(ciphertext))
	out = append(out, MagicBytes[:]...)

	var u32buf [4]byte
	binary.BigEndian.PutUint32(u32buf[:], h.Time)
	out = append(out, u32buf[:]...)
	binary.BigEndian.PutUint32(u32buf[:], h.Memory)
	out = append(out, u32buf[:]...)

	out = append(out, h.Par)
	out = append(out, h.Salt[:]...)
	out = append(out, h.Nonce[:]...)
	out = append(out, ciphertext...)

	return out, nil
}

// Decrypt decrypts a blob produced by Encrypt.
// Returns a user-friendly error on wrong password or corrupt data.
func Decrypt(blob, password []byte) ([]byte, error) {
	if len(blob) < headerSize {
		return nil, errors.New("encrypted data too short")
	}

	if blob[0] != MagicBytes[0] || blob[1] != MagicBytes[1] ||
		blob[2] != MagicBytes[2] || blob[3] != MagicBytes[3] {
		return nil, errors.New("invalid magic bytes: not an encrypted wallet file")
	}

	h := encHeader{
		Time:   binary.BigEndian.Uint32(blob[4:8]),
		Memory: binary.BigEndian.Uint32(blob[8:12]),
		Par:    blob[12],
	}
	copy(h.Salt[:], blob[13:45])
	copy(h.Nonce[:], blob[45:69])

	ciphertext := blob[headerSize:]

	key := DeriveKey(password, h.Salt[:], h.Time, h.Memory, h.Par)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	plaintext, err := aead.Open(nil, h.Nonce[:], ciphertext, nil)
	if err != nil {
		return nil, errors.New("incorrect password or corrupted wallet data")
	}

	return plaintext, nil
}

// IsEncrypted reports whether data begins with the wallet encryption magic bytes.
func IsEncrypted(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return data[0] == MagicBytes[0] && data[1] == MagicBytes[1] &&
		data[2] == MagicBytes[2] && data[3] == MagicBytes[3]
}
