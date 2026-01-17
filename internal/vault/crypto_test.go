package vault

import (
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	key := []byte("thisis32byteslongsecretkey123456") // 32 bytes for AES-256
	plaintext := "Hello, Celerix!"

	ciphertext, err := Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	if ciphertext == plaintext {
		t.Fatal("Ciphertext should not be equal to plaintext")
	}

	decrypted, err := Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected %s, got %s", plaintext, decrypted)
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	key1 := []byte("thisis32byteslongsecretkey123456")
	key2 := []byte("another32byteslongsecretkey65432")
	plaintext := "Secret message"

	ciphertext, err := Encrypt(plaintext, key1)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	_, err = Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("Decryption should have failed with wrong key")
	}
}

func TestInvalidKeySize(t *testing.T) {
	invalidKey := []byte("shortkey")
	plaintext := "test"

	_, err := Encrypt(plaintext, invalidKey)
	if err == nil {
		t.Fatal("Encryption should fail with invalid key size")
	}

	_, err = Decrypt("0123456789abcdef", invalidKey)
	if err == nil {
		t.Fatal("Decryption should fail with invalid key size")
	}
}

func TestGenerateSelfSignedCert(t *testing.T) {
	cert, err := GenerateSelfSignedCert()
	if err != nil {
		t.Fatalf("Failed to generate self-signed cert: %v", err)
	}

	if len(cert.Certificate) == 0 {
		t.Fatal("Generated certificate is empty")
	}

	if cert.PrivateKey == nil {
		t.Fatal("Generated private key is nil")
	}
}

func TestDecryptMalformedHex(t *testing.T) {
	key := []byte("thisis32byteslongsecretkey123456")
	_, err := Decrypt("not-hex", key)
	if err == nil {
		t.Fatal("Decryption should fail with malformed hex")
	}
}

func TestDecryptTooShort(t *testing.T) {
	key := []byte("thisis32byteslongsecretkey123456")
	// AES-GCM nonce is usually 12 bytes, so anything shorter than that (in hex: 24 chars) is definitely too short.
	_, err := Decrypt("abcdef", key)
	if err == nil {
		t.Fatal("Decryption should fail with too short ciphertext")
	}
}
