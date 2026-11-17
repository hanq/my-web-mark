package crypto

import (
	"bytes"
	"testing"
)

func TestPasswordRoundTrip(t *testing.T) {
	salt, err := NewSalt()
	if err != nil {
		t.Fatal(err)
	}
	p := DefaultKDFParams()
	hash := HashPassword("admin123", salt, p)

	if !VerifyPassword("admin123", salt, hash, p) {
		t.Fatal("正确密码校验失败")
	}
	if VerifyPassword("wrong", salt, hash, p) {
		t.Fatal("错误密码通过了校验")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	salt, _ := NewSalt()
	_, key := DeriveKey("admin123", salt, DefaultKDFParams())

	plain := []byte("p@ssw0rd-中文-🔐symbols")
	ct, err := Encrypt(key, plain)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Decrypt(key, ct)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, got) {
		t.Fatalf("解密不匹配: %q", got)
	}

	// 篡改密文应失败
	ct[len(ct)-1] ^= 0xFF
	if _, err := Decrypt(key, ct); err == nil {
		t.Fatal("篡改密文未被拒绝")
	}
}

func TestStringRoundTrip(t *testing.T) {
	salt, _ := NewSalt()
	_, key := DeriveKey("admin123", salt, DefaultKDFParams())

	enc, err := EncryptString(key, "hello 密码")
	if err != nil {
		t.Fatal(err)
	}
	dec, err := DecryptString(key, enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec != "hello 密码" {
		t.Fatalf("解密结果错误: %q", dec)
	}
}

func TestWipeKey(t *testing.T) {
	salt, _ := NewSalt()
	_, key := DeriveKey("admin123", salt, DefaultKDFParams())
	WipeKey(key)
	for _, b := range key {
		if b != 0 {
			t.Fatal("密钥未清零")
		}
	}
}

func TestParamsMarshal(t *testing.T) {
	p := DefaultKDFParams()
	s, err := MarshalParams(p)
	if err != nil {
		t.Fatal(err)
	}
	got := UnmarshalParams(s)
	if got != p {
		t.Fatalf("参数往返不一致: %+v vs %+v", got, p)
	}
	if UnmarshalParams("garbage") != DefaultKDFParams() {
		t.Fatal("非法参数应回退默认")
	}
}
