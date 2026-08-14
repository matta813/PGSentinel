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
