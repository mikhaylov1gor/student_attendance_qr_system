package repo

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"attendance/internal/domain/audit"
	"attendance/internal/infrastructure/db/models"
	"attendance/internal/infrastructure/db/txctx"
)

type AuditRepo struct{ db *gorm.DB }

func NewAuditRepo(db *gorm.DB) *AuditRepo { return &AuditRepo{db: db} }

var _ audit.Repository = (*AuditRepo)(nil)

func (r *AuditRepo) dbx(ctx context.Context) *gorm.DB { return txctx.DBX(ctx, r.db) }

// Append вставляет запись в audit_log.
//
// Важно: Append ДОЛЖЕН вызываться внутри транзакции (TxRunner.Run), иначе
// advisory_xact_lock не удержит SELECT и INSERT вместе, и две параллельные
// записи получат одинаковый prev_hash. При отсутствии внешней транзакции мы
// открываем собственную, но вызывающему коду это редко нужно — обычно
// основная мутация всё равно в tx.
//
// Сервис слоя application должен предварительно вычислить record_hash через
// audit.ComputeRecordHash(prev, entry) — в этом слое только персистентность.
func (r *AuditRepo) Append(ctx context.Context, e audit.Entry) (audit.Entry, error) {
	if _, inTx := txctx.From(ctx); inTx {
		return r.appendInTx(ctx, e)
	}
	var saved audit.Entry
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		innerCtx := txctx.With(ctx, tx)
		var err error
		saved, err = r.appendInTx(innerCtx, e)
		return err
	})
	return saved, err
}

func (r *AuditRepo) appendInTx(ctx context.Context, e audit.Entry) (audit.Entry, error) {
	// Advisory lock — сериализация append'ов внутри транзакции.
	if err := r.dbx(ctx).Exec("SELECT pg_advisory_xact_lock(hashtext('audit_log'))").Error; err != nil {
		return audit.Entry{}, fmt.Errorf("audit advisory_xact_lock: %w", err)
	}

	m, err := models.AuditEntryToModel(e)
	if err != nil {
		return audit.Entry{}, err
	}
	if err := r.dbx(ctx).Create(&m).Error; err != nil {
		return audit.Entry{}, fmt.Errorf("audit insert: %w", err)
	}
	return models.AuditEntryFromModel(m)
}

// Last возвращает последнюю запись по id (источник порядка цепочки).
// ok=false, если таблица пуста (нужно для genesis-записи).
func (r *AuditRepo) Last(ctx context.Context) (audit.Entry, bool, error) {
	var m models.AuditLogModel
	err := r.dbx(ctx).Order("id DESC").Limit(1).First(&m).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return audit.Entry{}, false, nil
		}
		return audit.Entry{}, false, fmt.Errorf("audit last: %w", err)
	}
	e, err := models.AuditEntryFromModel(m)
	if err != nil {
		return audit.Entry{}, false, err
	}
	return e, true, nil
}

func (r *AuditRepo) List(ctx context.Context, f audit.ListFilter) ([]audit.Entry, int, error) {
	q := r.dbx(ctx).Model(&models.AuditLogModel{})
	if f.ActorID != nil {
		q = q.Where("actor_id = ?", *f.ActorID)
	}
	if f.Action != nil {
		q = q.Where("action = ?", string(*f.Action))
	}
	if f.EntityType != nil {
		q = q.Where("entity_type = ?", *f.EntityType)
	}
	if f.EntityID != nil {
		q = q.Where("entity_id = ?", *f.EntityID)
	}
	if f.FromTime != nil {
		q = q.Where("occurred_at >= ?", *f.FromTime)
	}
	if f.ToTime != nil {
		q = q.Where("occurred_at <= ?", *f.ToTime)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("audit list count: %w", err)
	}
	if f.Limit > 0 {
		q = q.Limit(f.Limit)
	}
	if f.Offset > 0 {
		q = q.Offset(f.Offset)
	}

	var ms []models.AuditLogModel
	if err := q.Order("id DESC").Find(&ms).Error; err != nil {
		return nil, 0, fmt.Errorf("audit list: %w", err)
	}
	out := make([]audit.Entry, 0, len(ms))
	for _, m := range ms {
		e, err := models.AuditEntryFromModel(m)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, e)
	}
	return out, int(total), nil
}

// Scan проходит всю цепочку батчами в порядке id ASC (т.е. хронологически),
// вызывая fn для каждой записи. Используется верификатором.
// Если fn возвращает ошибку — скан прерывается и ошибка проксируется наверх.
func (r *AuditRepo) Scan(ctx context.Context, batchSize int, fn func(audit.Entry) error) error {
	if batchSize <= 0 {
		batchSize = 500
	}
	var lastID int64 = 0
	for {
		var batch []models.AuditLogModel
		err := r.dbx(ctx).
			Where("id > ?", lastID).
			Order("id ASC").
			Limit(batchSize).
			Find(&batch).Error
		if err != nil {
			return fmt.Errorf("audit scan: %w", err)
		}
		if len(batch) == 0 {
			return nil
		}
		for _, m := range batch {
			e, err := models.AuditEntryFromModel(m)
			if err != nil {
				return err
			}
			if err := fn(e); err != nil {
				return err
			}
			lastID = m.ID
		}
	}
}
