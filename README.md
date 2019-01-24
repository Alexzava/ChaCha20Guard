# ChaCha20Guard

A pure Go implementation of ChaCha20 and its extended nonce variant XChaCha20 with [MemGuard](https://github.com/awnumar/memguard) in order to protect the key in memory.

Before using read the [Warning](README.md#Warning)

The implementation is based on [https://github.com/codahale/chacha20](https://github.com/codahale/chacha20)

<a href="https://godoc.org/github.com/alexzava/ChaCha20Guard"><img src="https://godoc.org/github.com/alexzava/ChaCha20Guard?status.svg"></a>

## Download/Install
```
go get -u github.com/alexzava/chacha20guard
```

## Usage

### Import
```
import (
	"log"
	"crypto/rand"

	"github.com/awnumar/memguard"
	"github.com/alexzava/chacha20guard"
)
```

### ChaCha20

```
message := []byte("Hello World!")

//Generate random nonce
nonce := make([]byte, 8)
_, err := rand.Read(nonce)
if err != nil {
	log.Fatal(err)
}

//Generate random key with memguard
key, err := memguard.NewImmutableRandom(32)
if err != nil {
	log.Println(err)
	memguard.SafeExit(1)
}
defer key.Destroy()

c, err := chacha20guard.New(key, nonce)
if err != nil {
	log.Fatal(err)
}

ciphertext := make([]byte, len(message))
c.XORKeyStream(ciphertext, message)
```

### XChaCha20

```
message := []byte("Hello World!")

//Generate random nonce
nonce := make([]byte, 24)
_, err := rand.Read(nonce)
if err != nil {
	log.Fatal(err)
}

//Generate random key with memguard
key, err := memguard.NewImmutableRandom(32)
if err != nil {
	log.Println(err)
	memguard.SafeExit(1)
}
defer key.Destroy()

c, err := chacha20guard.NewX(key, nonce)
if err != nil {
	log.Fatal(err)
}

ciphertext := make([]byte, len(message))
c.XORKeyStream(ciphertext, message)
```

## Warning

The code may contain bugs or vulnerabilities, currently they have not been found but this does not guarantee absolute security.

Check the repository often because the code could be updated frequently.

## Notes

If you find bugs or vulnerabilities please let me know so they can be fixed.

If you want to help improve the code contact me.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.