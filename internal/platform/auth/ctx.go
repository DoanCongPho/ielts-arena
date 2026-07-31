package auth

import (
	"context"
	"net/http"
)

type ctxKey int

const userKey ctxKey = 0

func SetCurrentUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userKey, u)
}

func CurrentUser(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(userKey).(*User)
	return u, ok
}

func CurrentUserID(r *http.Request) (uint64, bool) {
	user, ok := CurrentUser(r.Context())
	if !ok {
		return 0, false
	}
	return user.ID, true
}
