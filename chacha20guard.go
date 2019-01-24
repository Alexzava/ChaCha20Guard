// https://github.com/alexzava/chacha20guard
//
// Package chacha20guard provides a pure Go implementation of ChaCha20
// and its extended nonce variant XChaCha20 with memguard 
// in order to protect the key in memory.
// 
// The code is based on https://github.com/codahale/chacha20


package chacha20guard

import (
	"crypto/cipher"
	"encoding/binary"
	"errors"
	"unsafe"

	"github.com/awnumar/memguard"
)

const (
	// KeySize is the length of ChaCha20 keys, in bytes.
	KeySize = 32
	// NonceSize is the length of ChaCha20 nonces, in bytes.
	NonceSize = 8
	// XNonceSize is the length of XChaCha20 nonces, in bytes.
	XNonceSize = 24
)

var (
	// ErrInvalidKey is returned when the provided key is not 256 bits long.
	ErrInvalidKey = errors.New("invalid key length (must be 256 bits)")
	// ErrInvalidNonce is returned when the provided nonce is not 64 bits long.
	ErrInvalidNonce = errors.New("invalid nonce length (must be 64 bits)")
	// ErrInvalidXNonce is returned when the provided nonce is not 192 bits
	// long.
	ErrInvalidXNonce = errors.New("invalid nonce length (must be 192 bits)")
	// ErrInvalidRounds is returned when the provided rounds is not
	// 8, 12, or 20.
	ErrInvalidRounds = errors.New("invalid rounds number (must be 8, 12, or 20)")
)

// New creates and returns a new cipher.Stream. The key must be 256
// bits long, and the nonce argument must be 64 bits long. The nonce must be
// randomly generated or used only once.
func New(key *memguard.LockedBuffer, nonce []byte) (cipher.Stream, error) {
	return NewWithRounds(key, nonce, 20)
}

// NewWithRounds creates and returns a new cipher.Stream just like New but
// the rounds number of 8, 12, or 20 can be specified.
func NewWithRounds(key *memguard.LockedBuffer, nonce []byte, rounds uint8) (cipher.Stream, error) {
	if len(key.Buffer()) != KeySize {
		return nil, ErrInvalidKey
	}

	if len(nonce) != NonceSize {
		return nil, ErrInvalidNonce
	}

	if (rounds != 8) && (rounds != 12) && (rounds != 20) {
		return nil, ErrInvalidRounds
	}

	s := new(stream)
	s.init(key, nonce, rounds)
	s.advance()

	return s, nil
}

// NewX creates and returns a new cipher.Stream. The key must be
// 256 bits long, and the nonce argument must be 192 bits long.
func NewX(key *memguard.LockedBuffer, nonce []byte) (cipher.Stream, error) {
	return NewXWithRounds(key, nonce, 20)
}

// NewXWithRounds creates and returns a new cipher.Stream just like
// NewX but the rounds number of 8, 12, or 20 can be specified.
func NewXWithRounds(key *memguard.LockedBuffer, nonce []byte, rounds uint8) (cipher.Stream, error) {
	if len(key.Buffer()) != KeySize {
		return nil, ErrInvalidKey
	}

	if len(nonce) != XNonceSize {
		return nil, ErrInvalidXNonce
	}

	if (rounds != 8) && (rounds != 12) && (rounds != 20) {
		return nil, ErrInvalidRounds
	}

	s := new(stream)
	s.init(key, nonce, rounds)

	// Call HChaCha to derive the subkey using the key and the first 16 bytes
	// of the nonce, and re-initialize the state using the subkey and the
	// remaining nonce.
	blockArr := (*[stateSize]uint32)(unsafe.Pointer(&s.block))
	core(&s.state, blockArr, s.rounds, true)
	copy(s.state[4:8], blockArr[0:4])
	copy(s.state[8:12], blockArr[12:16])
	s.state[12] = 0
	s.state[13] = 0
	s.state[14] = binary.LittleEndian.Uint32(nonce[16:])
	s.state[15] = binary.LittleEndian.Uint32(nonce[20:])

	s.advance()

	return s, nil
}

type stream struct {
	state  [stateSize]uint32 // the state as an array of 16 32-bit words
	block  [blockSize]byte   // the keystream as an array of 64 bytes
	offset int               // the offset of used bytes in block
	rounds uint8
}

func (s *stream) XORKeyStream(dst, src []byte) {
	// Stride over the input in 64-byte blocks, minus the amount of keystream
	// previously used. This will produce best results when processing blocks
	// of a size evenly divisible by 64.
	i := 0
	max := len(src)
	for i < max {
		gap := blockSize - s.offset

		limit := i + gap
		if limit > max {
			limit = max
		}

		o := s.offset
		for j := i; j < limit; j++ {
			dst[j] = src[j] ^ s.block[o]
			o++
		}

		i += gap
		s.offset = o

		if o == blockSize {
			s.advance()
		}
	}
}

func (s *stream) init(key *memguard.LockedBuffer, nonce []byte, rounds uint8) {
	// the magic constants for 256-bit keys
	s.state[0] = 0x61707865
	s.state[1] = 0x3320646e
	s.state[2] = 0x79622d32
	s.state[3] = 0x6b206574

	s.state[4] = binary.LittleEndian.Uint32(key.Buffer()[0:])
	s.state[5] = binary.LittleEndian.Uint32(key.Buffer()[4:])
	s.state[6] = binary.LittleEndian.Uint32(key.Buffer()[8:])
	s.state[7] = binary.LittleEndian.Uint32(key.Buffer()[12:])
	s.state[8] = binary.LittleEndian.Uint32(key.Buffer()[16:])
	s.state[9] = binary.LittleEndian.Uint32(key.Buffer()[20:])
	s.state[10] = binary.LittleEndian.Uint32(key.Buffer()[24:])
	s.state[11] = binary.LittleEndian.Uint32(key.Buffer()[28:])

	switch len(nonce) {
	case NonceSize:
		// ChaCha20 uses 8 byte nonces.
		s.state[12] = 0
		s.state[13] = 0
		s.state[14] = binary.LittleEndian.Uint32(nonce[0:])
		s.state[15] = binary.LittleEndian.Uint32(nonce[4:])
	case XNonceSize:
		// XChaCha20 derives the subkey via HChaCha initialized
		// with the first 16 bytes of the nonce.
		s.state[12] = binary.LittleEndian.Uint32(nonce[0:])
		s.state[13] = binary.LittleEndian.Uint32(nonce[4:])
		s.state[14] = binary.LittleEndian.Uint32(nonce[8:])
		s.state[15] = binary.LittleEndian.Uint32(nonce[12:])
	default:
		// Never happens, both ctors validate the nonce length.
		panic("invalid nonce size")
	}

	s.rounds = rounds
}

// advances the keystream
func (s *stream) advance() {
	core(&s.state, (*[stateSize]uint32)(unsafe.Pointer(&s.block)), s.rounds, false)

	if bigEndian {
		j := blockSize - 1
		for i := 0; i < blockSize/2; i++ {
			s.block[j], s.block[i] = s.block[i], s.block[j]
			j--
		}
	}

	s.offset = 0
	i := s.state[12] + 1
	s.state[12] = i
	if i == 0 {
		s.state[13]++
	}
}

const (
	wordSize  = 4                    // the size of ChaCha20's words
	stateSize = 16                   // the size of ChaCha20's state, in words
	blockSize = stateSize * wordSize // the size of ChaCha20's block, in bytes
)

var (
	bigEndian bool // whether or not we're running on a bigEndian CPU
)

// Do some up-front bookkeeping on what sort of CPU we're using. ChaCha20 treats
// its state as a little-endian byte array when it comes to generating the
// keystream, which allows for a zero-copy approach to the core transform. On
// big-endian architectures, we have to take a hit to reverse the bytes.
func init() {
	x := uint32(0x04030201)
	y := [4]byte{0x1, 0x2, 0x3, 0x4}
	bigEndian = *(*[4]byte)(unsafe.Pointer(&x)) != y
}