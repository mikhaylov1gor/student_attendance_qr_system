package db

import (
	"context"

	"gorm.io/gorm"

	"attendance/internal/domain"
	"attendance/internal/infrastructure/db/txctx"
)

// TxRunner реализует domain.TxRunner поверх Gorm.
type TxRunner struct{ db *gorm.DB }

func NewTxRunner(db *gorm.DB) *TxRunner { return &TxRunner{db: db} }

var _ domain.TxRunner = (*TxRunner)(nil)

// Run открывает транзакцию, кладёт её в ctx через txctx.With, вызывает fn.
// На ошибке из fn — rollback, на success — commit. Вложенные Run не открывают
// новую транзакцию (gorm использует savepoints; для наших нужд это ок).
func (r *TxRunner) Run(ctx context.Context, fn func(context.Context) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(txctx.With(ctx, tx))
	})
}
