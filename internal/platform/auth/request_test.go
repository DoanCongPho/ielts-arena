package auth

import "testing"

func TestRegisterRequestValidateEmail(t *testing.T) {
	cases := []struct {
		name      string
		email     string
		wantEmail string // empty = expect an error
	}{
		{"gmail", "learner@gmail.com", "learner@gmail.com"},
		{"uppercase and spaces are normalized", "  Learner@Gmail.COM ", "learner@gmail.com"},
		{"dots and plus are kept", "first.last+ielts@gmail.com", "first.last+ielts@gmail.com"},
		{"the typo that created a second account", "learner@gmail.con", ""},
		{"another domain", "learner@yahoo.com", ""},
		{"gmail as a subdomain of something else", "learner@gmail.com.vn", ""},
		{"no local part", "@gmail.com", ""},
		{"not an address at all", "learner", ""},
		{"empty", "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := RegisterRequest{Name: "Learner", Email: tc.email, Password: "password123"}
			err := req.Validate()
			if tc.wantEmail == "" {
				if err == nil {
					t.Fatalf("Validate() accepted %q", tc.email)
				}
				return
			}
			if err != nil {
				t.Fatalf("Validate() rejected %q: %v", tc.email, err)
			}
			if req.Email != tc.wantEmail {
				t.Errorf("Email = %q, want %q", req.Email, tc.wantEmail)
			}
		})
	}
}

// Login only normalizes: an account created before the Gmail-only rule
// must still be able to sign in, and a wrong domain is just wrong
// credentials.
func TestLoginRequestNormalizesEmail(t *testing.T) {
	req := LoginRequest{Email: " Learner@Yahoo.com ", Password: "password123"}
	if err := req.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if req.Email != "learner@yahoo.com" {
		t.Errorf("Email = %q, want %q", req.Email, "learner@yahoo.com")
	}
}
