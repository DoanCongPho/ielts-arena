package auth

import "context"

type ctxKey int

const userKey ctxKey = 0

func SetCurrentUser(ctx context.Context, u *User) context.Context {
	return context.WithValue(ctx, userKey, u.ID)
}

func CurrentUser(ctx context.Context) (*User, bool) {
	u, ok := ctx.Value(userKey).(*User)
	return u, ok
}
