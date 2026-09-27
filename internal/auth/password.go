package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/crypto/argon2"
)

const MaxPasswordBytes = 1024

var ErrInvalidHash = errors.New("invalid password hash")

type PasswordHash struct {
	memory, iterations uint32
	parallel           uint8
	salt, key          []byte
}

func ParseHash(phc string) (PasswordHash, error) {
	var h PasswordHash
	if len(phc) > 512 {
		return h, ErrInvalidHash
	}
	parts := strings.Split(phc, "$")
	if len(parts) != 6 || parts[0] != "" || parts[1] != "argon2id" || parts[2] != "v=19" {
		return h, ErrInvalidHash
	}
	params := strings.Split(parts[3], ",")
	if len(params) != 3 {
		return h, ErrInvalidHash
	}
	values := make([]uint64, 3)
	for i, prefix := range []string{"m=", "t=", "p="} {
		if !strings.HasPrefix(params[i], prefix) {
			return h, ErrInvalidHash
		}
		v, err := strconv.ParseUint(strings.TrimPrefix(params[i], prefix), 10, 32)
		if err != nil {
			return h, ErrInvalidHash
		}
		values[i] = v
	}
	if values[0] < 65536 || values[0] > 262144 || values[1] < 3 || values[1] > 6 || values[2] < 1 || values[2] > 4 {
		return h, ErrInvalidHash
	}
	h.memory = uint32(values[0])
	h.iterations = uint32(values[1])
	h.parallel = uint8(values[2])
	var err error
	h.salt, err = base64.RawStdEncoding.Strict().DecodeString(parts[4])
	if err != nil || len(h.salt) != 16 {
		return PasswordHash{}, ErrInvalidHash
	}
	h.key, err = base64.RawStdEncoding.Strict().DecodeString(parts[5])
	if err != nil || len(h.key) != 32 {
		return PasswordHash{}, ErrInvalidHash
	}
	return h, nil
}

func (h PasswordHash) Verify(password []byte) bool {
	if len(password) == 0 || len(password) > MaxPasswordBytes {
		return false
	}
	key := argon2.IDKey(password, h.salt, h.iterations, h.memory, h.parallel, uint32(len(h.key)))
	defer clear(key)
	return subtle.ConstantTimeCompare(key, h.key) == 1
}

func HashPassword(password []byte) (string, error) {
	if len(password) < 12 || len(password) > MaxPasswordBytes {
		return "", errors.New("密码长度必须为12到1024字节")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey(password, salt, 3, 65536, 1, 32)
	defer clear(key)
	return fmt.Sprintf("$argon2id$v=19$m=65536,t=3,p=1$%s$%s", base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)), nil
}
