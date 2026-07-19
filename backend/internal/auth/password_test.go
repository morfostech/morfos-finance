package auth

import "testing"

func TestHashAndVerifyPassword(t *testing.T) {
	const pw = "s3nha-forte!"
	hash, err := HashPassword(pw)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == pw {
		t.Fatal("hash equals plaintext")
	}

	ok, err := VerifyPassword(pw, hash)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if !ok {
		t.Fatal("correct password did not verify")
	}

	ok, err = VerifyPassword("senha-errada", hash)
	if err != nil {
		t.Fatalf("verify wrong: %v", err)
	}
	if ok {
		t.Fatal("wrong password verified")
	}
}

func TestHashUniqueSalt(t *testing.T) {
	h1, _ := HashPassword("mesma-senha")
	h2, _ := HashPassword("mesma-senha")
	if h1 == h2 {
		t.Fatal("expected different hashes for repeated password (salt reuse)")
	}
}

func TestVerifyInvalidHashFormat(t *testing.T) {
	if _, err := VerifyPassword("x", "not-a-valid-hash"); err == nil {
		t.Fatal("expected error for malformed hash")
	}
}
