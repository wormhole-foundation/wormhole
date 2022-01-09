package common

import (
	"testing"
)

func TestAESGCM(t *testing.T) {
	data := []byte("cat")
	key := []byte("01234567890123456789012345678901")

	enc, err := EncryptAESGCM(data, key)
	if err != nil {
		t.Fatal(err)
	}

	dec, err := DecryptAESGCM(enc, key)
	if err != nil {
		t.Fatal(err)
	}

	if string(dec) != string(data) {
		t.Fatalf("expected %s got %s", string(data), string(dec))
	}
}
