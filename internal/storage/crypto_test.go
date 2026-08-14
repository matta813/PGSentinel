package storage

import "testing"

func TestCipherRoundTrip(t *testing.T) {
	c, err := NewCipher("a sufficiently long test key")
	if err != nil {
		t.Fatal(err)
	}
	v, err := c.Encrypt("secret-password")
	if err != nil {
		t.Fatal(err)
	}
	if string(v) == "secret-password" {
		t.Fatal("plaintext leaked")
	}
	got, err := c.Decrypt(v)
	if err != nil || got != "secret-password" {
		t.Fatalf("got %q: %v", got, err)
	}
}

func FuzzCipherRoundTrip(f *testing.F) {
	for _, seed := range []string{"", "secret-password", "pässwörd-🔐", string([]byte{0, 1, 2, 255})} {
		f.Add(seed)
	}
	c, err := NewCipher("a sufficiently long fuzz test key")
	if err != nil {
		f.Fatal(err)
	}
	f.Fuzz(func(t *testing.T, plaintext string) {
		if len(plaintext) > 64*1024 {
			t.Skip()
		}
		encrypted, err := c.Encrypt(plaintext)
		if err != nil {
			t.Fatal(err)
		}
		decrypted, err := c.Decrypt(encrypted)
		if err != nil {
			t.Fatal(err)
		}
		if decrypted != plaintext {
			t.Fatalf("round trip changed %d-byte plaintext", len(plaintext))
		}

		encrypted[len(encrypted)-1] ^= 1
		if _, err := c.Decrypt(encrypted); err == nil {
			t.Fatal("tampered ciphertext was accepted")
		}
	})
}
