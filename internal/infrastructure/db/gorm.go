// Package db — открытие Gorm-соединения с Postgres и мост Gorm-logger → slog.
package db

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Options — настройки пула соединений. Значения «из воздуха» — разумный дефолт
// для монолита с одним инстансом API.
type Options struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	SlowThreshold   time.Duration // порог, после которого запрос логируется как slow
}

// DefaultOptions возвращает адекватные для разработки значения.
func DefaultOptions() Options {
	return Options{
		MaxOpenConns:    20,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		SlowThreshold:   200 * time.Millisecond,
	}
}

// Open открывает Postgres через Gorm с slog-логгером.
func Open(ctx context.Context, dsn string, log *slog.Logger, opts Options) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger:                 &slogGormLogger{log: log, slow: opts.SlowThreshold, level: gormlogger.Warn},
		PrepareStmt:            true,
		SkipDefaultTransaction: false,
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("gorm sqldb: %w", err)
	}
	sqlDB.SetMaxOpenConns(opts.MaxOpenConns)
	sqlDB.SetMaxIdleConns(opts.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(opts.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("gorm ping: %w", err)
	}
	return db, nil
}

// Close — корректное закрытие пула. Безопасно на nil.
func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ====================================================================
// Мост gormlogger.Interface → slog
// ====================================================================

type slogGormLogger struct {
	log   *slog.Logger
	slow  time.Duration
	level gormlogger.LogLevel
}

func (l *slogGormLogger) LogMode(lvl gormlogger.LogLevel) gormlogger.Interface {
	nl := *l
	nl.level = lvl
	return &nl
}

func (l *slogGormLogger) Info(ctx context.Context, msg string, args ...any) {
	if l.level < gormlogger.Info {
		return
	}
	l.log.InfoContext(ctx, fmt.Sprintf(msg, args...))
}

func (l *slogGormLogger) Warn(ctx context.Context, msg string, args ...any) {
	if l.level < gormlogger.Warn {
		return
	}
	l.log.WarnContext(ctx, fmt.Sprintf(msg, args...))
}

func (l *slogGormLogger) Error(ctx context.Context, msg string, args ...any) {
	if l.level < gormlogger.Error {
		return
	}
	l.log.ErrorContext(ctx, fmt.Sprintf(msg, args...))
}

func (l *slogGormLogger) Trace(
	ctx context.Context,
	begin time.Time,
	fc func() (sql string, rowsAffected int64),
	err error,
) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	switch {
	case err != nil && !errors.Is(err, gorm.ErrRecordNotFound):
		l.log.ErrorContext(ctx, "gorm query failed",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
			slog.String("err", err.Error()),
		)
	case l.slow > 0 && elapsed > l.slow:
		l.log.WarnContext(ctx, "gorm slow query",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
		)
	case l.level >= gormlogger.Info:
		l.log.DebugContext(ctx, "gorm query",
			slog.String("sql", sql),
			slog.Int64("rows", rows),
			slog.Duration("elapsed", elapsed),
		)
	}
}
