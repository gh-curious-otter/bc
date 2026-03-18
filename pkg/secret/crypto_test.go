package secret

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := DeriveKey("test-passphrase", []byte("test-salt-12345"))
	plaintext := []byte("super-secret-api-key-12345")

	encrypted, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if encrypted == "" {
		t.Fatal("encrypted result is empty")
	}

	decrypted, err := Decrypt(key, encrypted)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := DeriveKey("passphrase-1", []byte("salt"))
	key2 := DeriveKey("passphrase-2", []byte("salt"))

	encrypted, err := Encrypt(key1, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = Decrypt(key2, encrypted)
	if err == nil {
		t.Fatal("expected error decrypting with wrong key")
	}
}

func TestDecrypt_TamperedCiphertext(t *testing.T) {
	key := DeriveKey("passphrase", []byte("salt"))

	encrypted, err := Encrypt(key, []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	// Decode, tamper, re-encode
	data, _ := base64.StdEncoding.DecodeString(encrypted)
	data[len(data)-1] ^= 0xFF // flip last byte (GCM auth tag)
	tampered := base64.StdEncoding.EncodeToString(data)

	_, err = Decrypt(key, tampered)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecrypt_TruncatedData(t *testing.T) {
	key := DeriveKey("passphrase", []byte("salt"))

	_, err := Decrypt(key, base64.StdEncoding.EncodeToString([]byte("short")))
	if err == nil {
		t.Fatal("expected error for truncated data")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	key := DeriveKey("passphrase", []byte("salt"))

	_, err := Decrypt(key, "not-valid-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDeriveKey_Deterministic(t *testing.T) {
	salt := []byte("consistent-salt")
	key1 := DeriveKey("same-passphrase", salt)
	key2 := DeriveKey("same-passphrase", salt)

	if !bytes.Equal(key1, key2) {
		t.Error("same inputs should produce same key")
	}
}

func TestDeriveKey_DifferentSalts(t *testing.T) {
	key1 := DeriveKey("passphrase", []byte("salt-a"))
	key2 := DeriveKey("passphrase", []byte("salt-b"))

	if bytes.Equal(key1, key2) {
		t.Error("different salts should produce different keys")
	}
}

func TestDeriveKey_Length(t *testing.T) {
	key := DeriveKey("passphrase", []byte("salt"))
	if len(key) != keySize {
		t.Errorf("key length = %d, want %d", len(key), keySize)
	}
}

func TestGenerateSalt_Unique(t *testing.T) {
	salt1, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	salt2, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(salt1, salt2) {
		t.Error("two salts should not be equal")
	}
}

func TestGenerateSalt_Length(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatal(err)
	}
	if len(salt) != saltSize {
		t.Errorf("salt length = %d, want %d", len(salt), saltSize)
	}
}

func TestEncrypt_DifferentNonces(t *testing.T) {
	key := DeriveKey("passphrase", []byte("salt"))
	plaintext := []byte("same input")

	enc1, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	enc2, err := Encrypt(key, plaintext)
	if err != nil {
		t.Fatal(err)
	}
	if enc1 == enc2 {
		t.Error("encrypting same plaintext twice should produce different ciphertexts (random nonce)")
	}
}
