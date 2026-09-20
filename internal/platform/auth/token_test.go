package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-at-least-32-characters-long!"

func testUser() *User {
	return &User{ID: 42, Email: "learner@example.com", Role: RoleUser}
}

func TestAccessTokenRoundTrip(t *testing.T) {
	Init([]byte(testSecret))

	token, err := GenerateAccessToken(testUser())
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	claims, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if claims.UserID != 42 {
		t.Errorf("UserID = %d, want 42", claims.UserID)
	}
	if claims.Email != "learner@example.com" {
		t.Errorf("Email = %q, want learner@example.com", claims.Email)
	}
	if claims.Role != RoleUser {
		t.Errorf("Role = %q, want %q", claims.Role, RoleUser)
	}
	if claims.Type != TokenTypeAccess {
		t.Errorf("Type = %q, want %q", claims.Type, TokenTypeAccess)
	}
}

// The access/refresh distinction is what stops a long-lived refresh token
// being used as a credential on every API call, so the type must survive
// the round trip intact.
func TestRefreshTokenCarriesRefreshType(t *testing.T) {
	Init([]byte(testSecret))

	token, err := GenerateRefreshToken(testUser())
	if err != nil {
		t.Fatalf("GenerateRefreshToken: %v", err)
	}

	claims, err := VerifyToken(token)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if claims.Type != TokenTypeRefresh {
		t.Errorf("Type = %q, want %q", claims.Type, TokenTypeRefresh)
	}
}

func TestVerifyTokenRejectsExpiredToken(t *testing.T) {
	Init([]byte(testSecret))

	// Negative TTL: issued and already expired.
	token, err := generateToken(testUser(), TokenTypeAccess, -time.Minute)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}

	if _, err := VerifyToken(token); err == nil {
		t.Fatal("expected an expired token to be rejected")
	}
}

func TestVerifyTokenRejectsTokenSignedWithAnotherSecret(t *testing.T) {
	Init([]byte("a-completely-different-secret-key-32b!"))
	token, err := GenerateAccessToken(testUser())
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	// Server restarts with the real secret; the old token must not verify.
	Init([]byte(testSecret))
	if _, err := VerifyToken(token); err == nil {
		t.Fatal("expected a token signed with a different secret to be rejected")
	}
}

func TestVerifyTokenRejectsTamperedPayload(t *testing.T) {
	Init([]byte(testSecret))

	token, err := GenerateAccessToken(testUser())
	if err != nil {
		t.Fatalf("GenerateAccessToken: %v", err)
	}

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("expected a 3-part JWT, got %d parts", len(parts))
	}
	// Flip a character in the payload, leaving the original signature.
	parts[1] = "X" + parts[1][1:]

	if _, err := VerifyToken(strings.Join(parts, ".")); err == nil {
		t.Fatal("expected a tampered token to be rejected")
	}
}

// The classic JWT attack: re-sign the token with alg "none" so no key is
// needed. VerifyToken must refuse anything that isn't HMAC.
func TestVerifyTokenRejectsNoneAlgorithm(t *testing.T) {
	Init([]byte(testSecret))

	claims := Claims{
		UserID: 1,
		Email:  "attacker@example.com",
		Role:   RoleAdmin,
		Type:   TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	unsigned, err := jwt.NewWithClaims(jwt.SigningMethodNone, claims).
		SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("build unsigned token: %v", err)
	}

	if _, err := VerifyToken(unsigned); err == nil {
		t.Fatal("expected an alg=none token to be rejected — it would grant admin for free")
	}
}

func TestVerifyTokenRejectsGarbage(t *testing.T) {
	Init([]byte(testSecret))

	for _, token := range []string{"", "not-a-jwt", "a.b.c"} {
		if _, err := VerifyToken(token); err == nil {
			t.Errorf("expected %q to be rejected", token)
		}
	}
}

func TestIsAdmin(t *testing.T) {
	if (&User{Role: RoleAdmin}).IsAdmin() != true {
		t.Error("an admin user should report IsAdmin() == true")
	}
	if (&User{Role: RoleUser}).IsAdmin() != false {
		t.Error("a regular user should report IsAdmin() == false")
	}
	var nilUser *User
	if nilUser.IsAdmin() != false {
		t.Error("a nil user must not be treated as an admin")
	}
}
