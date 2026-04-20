// Package txctx — единственное место, где gorm-транзакция живёт в ctx.
// Репозитории через хелпер ниже определяют: работать ли внутри чужой транзакции
// или брать корневой *gorm.DB.
package txctx

import (
	"context"

	"gorm.io/gorm"
)

type ctxKey int

const txKey ctxKey = 0

// With кладёт tx в ctx. Вызывается реализацией TxRunner из infrastructure/db.
func With(ctx context.Context, tx *gorm.DB) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

// From достаёт tx. ok=false — нет внешней транзакции, репо работает на своём db.
func From(ctx context.Context) (*gorm.DB, bool) {
	tx, ok := ctx.Value(txKey).(*gorm.DB)
	return tx, ok
}

// DBX — универсальный хелпер для репозиториев: если есть tx в ctx, берём её,
// иначе — базовый db. В любом случае добавляем WithContext(ctx).
func DBX(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := From(ctx); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}
