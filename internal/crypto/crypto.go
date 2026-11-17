// Package crypto 提供管理员密码哈希（argon2id）与敏感数据加密（AES-256-GCM）。
// 派生密钥只存内存，锁定时由调用方调用 WipeKey 清零。
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

// KDFParams argon2id 参数，随 salt 一起存库，便于未来升级参数。
type KDFParams struct {
	Time    uint32 `json:"time"`    // 迭代轮数
	Memory  uint32 `json:"memory"`  // 内存 KB
	Threads uint8  `json:"threads"` // 并行度
	KeyLen  uint32 `json:"keyLen"`  // 派生长度（字节）
}

// DefaultKDFParams 默认参数：64MB 内存、1 轮、4 线程、派生 64 字节
// （前 32 字节作验证哈希基准，后 32 字节作 AES 主密钥，见 DeriveKey）。
func DefaultKDFParams() KDFParams {
	return KDFParams{Time: 1, Memory: 64 * 1024, Threads: 4, KeyLen: 64}
}

// NewSalt 生成 16 字节随机盐。
func NewSalt() ([]byte, error) {
	s := make([]byte, 16)
	if _, err := rand.Read(s); err != nil {
		return nil, fmt.Errorf("crypto: 生成盐失败: %w", err)
	}
	return s, nil
}

// DeriveKey 用 argon2id 从密码与盐派生密钥材料。
// 返回 [0:32) 为校验哈希（存库比对），[32:64) 为 AES-256 主密钥（仅内存持有）。
func DeriveKey(password string, salt []byte, p KDFParams) (verifyHash, aesKey []byte) {
	raw := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, p.KeyLen)
	return raw[:32], raw[32:]
}

// HashPassword 仅生成校验哈希（注册/改密时存库用）。
func HashPassword(password string, salt []byte, p KDFParams) []byte {
	h, _ := DeriveKey(password, salt, p)
	return h
}

// VerifyPassword 校验管理员密码。
func VerifyPassword(password string, salt, expectHash []byte, p KDFParams) bool {
	got := HashPassword(password, salt, p)
	return constantTimeEqual(got, expectHash)
}

// MarshalParams 序列化 KDF 参数为 JSON（存库）。
func MarshalParams(p KDFParams) (string, error) {
	b, err := json.Marshal(p)
	return string(b), err
}

// UnmarshalParams 从 JSON 反序列化 KDF 参数；空值回退默认参数。
func UnmarshalParams(s string) KDFParams {
	var p KDFParams
	if err := json.Unmarshal([]byte(s), &p); err != nil || p.KeyLen == 0 {
		return DefaultKDFParams()
	}
	return p
}

// WipeKey 将密钥字节清零（锁定金库时调用）。
func WipeKey(key []byte) {
	for i := range key {
		key[i] = 0
	}
}

// Encrypt 用 AES-256-GCM 加密明文，返回 nonce||ciphertext。
func Encrypt(key, plaintext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("crypto: 生成 nonce 失败: %w", err)
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt 解密 nonce||ciphertext 格式的密文。
func Decrypt(key, data []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(data) < gcm.NonceSize() {
		return nil, errors.New("crypto: 密文长度非法")
	}
	return gcm.Open(nil, data[:gcm.NonceSize()], data[gcm.NonceSize():], nil)
}

// EncryptString 加密字符串并返回 base64（便于 JSON 传输场景）。
func EncryptString(key []byte, plaintext string) (string, error) {
	b, err := Encrypt(key, []byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// DecryptString 解密 base64 密文字符串。
func DecryptString(key []byte, encoded string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("crypto: base64 解码失败: %w", err)
	}
	b, err := Decrypt(key, raw)
	return string(b), err
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("crypto: 初始化 AES 失败: %w", err)
	}
	return cipher.NewGCM(block)
}

// constantTimeEqual 恒时比较，防时序侧信道。
func constantTimeEqual(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
