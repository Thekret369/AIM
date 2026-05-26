// Package crypto 提供 LanLine 内部加密工具。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const encryptedStringPrefix = "enc:v1:"

// EncryptString 使用 AES-GCM 加密短字符串，返回带版本前缀的密文。
func EncryptString(plaintext, secret string) (string, error) {
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return "", nil
	}
	if IsEncryptedString(plaintext) {
		return plaintext, nil
	}
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("encryption secret is empty")
	}

	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	payload := append(nonce, ciphertext...)
	return encryptedStringPrefix + base64.RawURLEncoding.EncodeToString(payload), nil
}

// DecryptString 解密 EncryptString 产生的密文；非密文会原样返回，方便兼容旧数据。
func DecryptString(value, secret string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || !IsEncryptedString(value) {
		return value, nil
	}
	if strings.TrimSpace(secret) == "" {
		return "", errors.New("encryption secret is empty")
	}

	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(value, encryptedStringPrefix))
	if err != nil {
		return "", err
	}
	gcm, err := newGCM(secret)
	if err != nil {
		return "", err
	}
	if len(payload) <= gcm.NonceSize() {
		return "", errors.New("encrypted payload is too short")
	}

	nonce := payload[:gcm.NonceSize()]
	ciphertext := payload[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// IsEncryptedString 判断字段是否已经是当前版本的密文格式。
func IsEncryptedString(value string) bool {
	return strings.HasPrefix(strings.TrimSpace(value), encryptedStringPrefix)
}

func newGCM(secret string) (cipher.AEAD, error) {
	sum := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(sum[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
