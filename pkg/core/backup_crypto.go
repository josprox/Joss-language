package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"

	"golang.org/x/crypto/pbkdf2"
)

// encryptBytes AES-256-GCM encryption helper
func (r *Runtime) encryptBytes(plain []byte, password string) ([]byte, error) {
	salt := make([]byte, 12)
	if _, err := rand.Read(salt); err != nil {
		return nil, err
	}

	key := pbkdf2.Key([]byte(password), salt, 1000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, aesGCM.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}

	ciphertext := aesGCM.Seal(nil, nonce, plain, nil)

	res := make([]byte, len(salt)+len(nonce)+len(ciphertext))
	copy(res[0:12], salt)
	copy(res[12:24], nonce)
	copy(res[24:], ciphertext)

	return res, nil
}

// decryptBytes AES-256-GCM decryption helper
func (r *Runtime) decryptBytes(enc []byte, password string) ([]byte, error) {
	if len(enc) < 24 {
		return nil, errors.New("datos encriptados corruptos")
	}

	salt := enc[0:12]
	nonce := enc[12:24]
	ciphertext := enc[24:]

	key := pbkdf2.Key([]byte(password), salt, 1000, 32, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return aesGCM.Open(nil, nonce, ciphertext, nil)
}

// calculateFileSHA256 returns hex SHA256 of file
func (r *Runtime) calculateFileSHA256(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return ""
	}
	return hex.EncodeToString(h.Sum(nil))
}

// CopyFile utility for moving/duplicating files
func (r *Runtime) CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
