package auth

import (
	"context"
	"errors"
	"testing"
)

type verifierStub struct {
	err     error
	calls   int
	address string
	purpose string
	code    string
}

func (v *verifierStub) Verify(_ context.Context, address, purpose, code string) error {
	v.calls++
	v.address = address
	v.purpose = purpose
	v.code = code
	return v.err
}

func TestRegisterRejectsInvalidUsername(t *testing.T) {
	h := newAuthHarness(t)
	h.service.SetCodeVerifier(&verifierStub{})

	for _, username := range []string{"ab", "a b", "user@name", "名字太长名字太长名字太长名字太长名字太长名字"} {
		if _, err := h.service.Register(context.Background(), RegisterInput{
			Username: username, Email: "new@example.com", Password: "long-enough-password", Code: "123456",
		}); !errors.Is(err, ErrUsernameInvalid) {
			t.Fatalf("username %q must be rejected, got %v", username, err)
		}
	}
}

func TestRegisterRejectsWeakPassword(t *testing.T) {
	h := newAuthHarness(t)
	verifier := &verifierStub{}
	h.service.SetCodeVerifier(verifier)

	if _, err := h.service.Register(context.Background(), RegisterInput{
		Username: "newuser", Email: "new@example.com", Password: "short", Code: "123456",
	}); !errors.Is(err, ErrPasswordInvalid) {
		t.Fatalf("expected password error, got %v", err)
	}
	if verifier.calls != 0 {
		t.Fatal("password must be validated before the code is consumed")
	}
}

func TestRegisterPropagatesCodeError(t *testing.T) {
	h := newAuthHarness(t)
	h.service.SetCodeVerifier(&verifierStub{err: ErrCodeInvalid})

	_, err := h.service.Register(context.Background(), RegisterInput{
		Username: "newuser", Email: "new@example.com", Password: "long-enough-password", Code: "000000",
	})
	if !errors.Is(err, ErrCodeInvalid) {
		t.Fatalf("expected code error, got %v", err)
	}
	if len(h.users.created) != 0 {
		t.Fatal("account must not be created when the code is wrong")
	}
}

func TestRegisterCreatesUserAndSession(t *testing.T) {
	h := newAuthHarness(t)
	verifier := &verifierStub{}
	h.service.SetCodeVerifier(verifier)

	result, err := h.service.Register(context.Background(), RegisterInput{
		Username: "newuser", Email: " New@Example.com ", Password: "long-enough-password",
		Code: "123456", DeviceID: "device-1", IPAddress: "127.0.0.1", UserAgent: "test",
	})
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}
	if len(h.users.created) != 1 {
		t.Fatalf("expected one account, got %d", len(h.users.created))
	}
	created := h.users.created[0]
	if created.Email != "new@example.com" {
		t.Fatalf("email not normalized: %q", created.Email)
	}
	if created.RoleCode != "user" {
		t.Fatalf("role must be user, got %q", created.RoleCode)
	}
	if verifier.purpose != CodePurposeRegister || verifier.address != "new@example.com" {
		t.Fatalf("unexpected verify call: %+v", verifier)
	}
	if result.Token == "" || result.Principal.UserID == 0 {
		t.Fatalf("session not issued: %+v", result)
	}
	if !h.users.loginRecorded {
		t.Fatal("registration should record the first login")
	}
}

func TestRegisterDuplicateAccount(t *testing.T) {
	h := newAuthHarness(t)
	h.service.SetCodeVerifier(&verifierStub{})
	h.users.duplicateOnAdd = true

	_, err := h.service.Register(context.Background(), RegisterInput{
		Username: "newuser", Email: "new@example.com", Password: "long-enough-password", Code: "123456",
	})
	if !errors.Is(err, ErrAccountExists) {
		t.Fatalf("expected duplicate error, got %v", err)
	}
}

func TestResetPasswordUpdatesHashAndRevokesSessions(t *testing.T) {
	h := newAuthHarness(t)
	h.service.SetCodeVerifier(&verifierStub{})

	if _, err := h.service.Login(context.Background(), LoginInput{
		Login: "alice", Password: "correct-password", IPAddress: "127.0.0.1",
	}); err != nil {
		t.Fatalf("login failed: %v", err)
	}

	if err := h.service.ResetPassword(context.Background(), ResetPasswordInput{
		Email: "ALICE@example.com", Code: "123456", NewPassword: "brand-new-password",
	}); err != nil {
		t.Fatalf("reset failed: %v", err)
	}
	if !h.users.passwordSet {
		t.Fatal("password hash not updated")
	}
	revoked := 0
	for _, value := range h.sessionRepo.revoked {
		if value {
			revoked++
		}
	}
	if revoked == 0 {
		t.Fatal("old sessions must be revoked after reset")
	}
}

func TestResetPasswordRejectsBadCode(t *testing.T) {
	h := newAuthHarness(t)
	h.service.SetCodeVerifier(&verifierStub{err: ErrCodeExpired})

	err := h.service.ResetPassword(context.Background(), ResetPasswordInput{
		Email: "alice@example.com", Code: "123456", NewPassword: "brand-new-password",
	})
	if !errors.Is(err, ErrCodeExpired) {
		t.Fatalf("expected code error, got %v", err)
	}
	if h.users.passwordSet {
		t.Fatal("password must not change when the code is invalid")
	}
}

func TestResetPasswordValidatesPasswordBeforeCode(t *testing.T) {
	h := newAuthHarness(t)
	verifier := &verifierStub{}
	h.service.SetCodeVerifier(verifier)

	if err := h.service.ResetPassword(context.Background(), ResetPasswordInput{
		Email: "alice@example.com", Code: "123456", NewPassword: "short",
	}); !errors.Is(err, ErrPasswordInvalid) {
		t.Fatalf("expected password error, got %v", err)
	}
	if verifier.calls != 0 {
		t.Fatal("code should not be consumed for an invalid password")
	}
}
