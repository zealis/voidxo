package auth

import (
	"errors"
	"strings"
	"testing"
)

func testHasher() *PasswordHasher {
	return NewPasswordHasher(Argon2Params{
		Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32,
	})
}

func TestPasswordHashAndVerify(t *testing.T) {
	hasher := testHasher()
	encoded, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$") {
		t.Fatalf("unexpected PHC prefix: %s", encoded)
	}
	ok, err := hasher.Verify(encoded, "correct horse battery staple")
	if err != nil || !ok {
		t.Fatalf("valid password rejected: ok=%v err=%v", ok, err)
	}
	ok, err = hasher.Verify(encoded, "wrong password")
	if err != nil || ok {
		t.Fatalf("wrong password accepted: ok=%v err=%v", ok, err)
	}
}

func TestPasswordPolicyAndMalformedHashes(t *testing.T) {
	hasher := testHasher()
	if _, err := hasher.Hash("short"); !errors.Is(err, ErrPasswordInvalid) {
		t.Fatalf("short password accepted: %v", err)
	}
	if _, err := hasher.Verify("$argon2id$v=19$m=1,t=1,p=1$bad$bad", "password"); !errors.Is(err, ErrHashInvalid) {
		t.Fatalf("malformed hash accepted: %v", err)
	}
}
