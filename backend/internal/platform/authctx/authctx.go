// Package authctx — единственное место, где хранится Principal в context.Context.
// Ключ непубличный: снаружи работают только With/From.
package authctx

import (
	"context"

	"attendance/internal/domain/auth"
)

type ctxKey int

const principalKey ctxKey = 0

// With кладёт Principal в контекст. Используется auth-middleware'ом.
func With(ctx context.Context, p auth.Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// From достаёт Principal. ok=false — запрос анонимный.
func From(ctx context.Context) (auth.Principal, bool) {
	p, ok := ctx.Value(principalKey).(auth.Principal)
	return p, ok
}

// Require удобен в хендлерах: возвращает Principal или domain-ошибку,
// чтобы не дублировать проверку ok в каждом месте.
func Require(ctx context.Context) (auth.Principal, error) {
	if p, ok := From(ctx); ok {
		return p, nil
	}
	return auth.Principal{}, auth.ErrUnauthorized
}
