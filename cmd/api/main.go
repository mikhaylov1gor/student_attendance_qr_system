package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	appattendance "attendance/internal/application/attendance"
	appaudit "attendance/internal/application/audit"
	"attendance/internal/application/auth"
	appcatalog "attendance/internal/application/catalog"
	"attendance/internal/application/hub"
	apppolicy "attendance/internal/application/policy"
	appqr "attendance/internal/application/qr"
	appreport "attendance/internal/application/report"
	appsession "attendance/internal/application/session"
	appuser "attendance/internal/application/user"
	"attendance/internal/config"
	"attendance/internal/domain/policy"
	"attendance/internal/domain/policy/checks"
	"attendance/internal/infrastructure/crypto"
	"attendance/internal/infrastructure/db"
	"attendance/internal/infrastructure/db/repo"
	apphttp "attendance/internal/infrastructure/http"
	"attendance/internal/infrastructure/http/handlers"
	"attendance/internal/infrastructure/http/httperr"
	appmid "attendance/internal/infrastructure/http/middleware"
	"attendance/internal/infrastructure/ws"
	"attendance/internal/platform/clock"
	"attendance/internal/platform/httpserver"
	"attendance/internal/platform/logging"
)

func main() {
	_ = godotenv.Load()

	if err := run(); err != nil {
		slog.Default().Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}

	logger, err := logging.New(cfg.LogLevel, cfg.LogFormat, os.Stdout)
	if err != nil {
		return fmt.Errorf("logging: %w", err)
	}
	slog.SetDefault(logger)

	if err := validateConfig(cfg); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// ---- Инфраструктура ----

	gormDB, err := db.Open(ctx, cfg.DatabaseDSN, logger, db.DefaultOptions())
	if err != nil {
		return fmt.Errorf("db open: %w", err)
	}
	defer db.Close(gormDB)

	encryptor, err := crypto.NewAESGCMEncryptorFromBase64(cfg.PIIEncryptionKeyB64)
	if err != nil {
		return fmt.Errorf("pii encryptor: %w", err)
	}

	hasher := crypto.NewArgon2idHasher(crypto.Argon2idParams{
		Memory:      cfg.Argon2MemoryKiB,
		Iterations:  cfg.Argon2Iterations,
		Parallelism: cfg.Argon2Parallelism,
	})

	signer, err := crypto.NewJWTSignerFromBase64(cfg.JWTAccessSecretB64, cfg.JWTIssuer)
	if err != nil {
		return fmt.Errorf("jwt signer: %w", err)
	}

	realClock := clock.New()

	// ---- Репозитории ----

	userRepo := repo.NewUserRepo(gormDB, encryptor)
	refreshRepo := repo.NewRefreshTokenRepo(gormDB)
	policyRepo := repo.NewPolicyRepo(gormDB)
	auditRepo := repo.NewAuditRepo(gormDB)
	courseRepo := repo.NewCourseRepo(gormDB)
	groupRepo := repo.NewGroupRepo(gormDB)
	streamRepo := repo.NewStreamRepo(gormDB)
	classroomRepo := repo.NewClassroomRepo(gormDB)
	sessionRepo := repo.NewSessionRepo(gormDB)
	attendanceRepo := repo.NewAttendanceRepo(gormDB)
	reportRepo := repo.NewReportRepo(gormDB)

	txRunner := db.NewTxRunner(gormDB)
	qrCodec := crypto.NewQRTokenCodec()

	// ---- Policy Engine (stateless, регистрация в этом месте) ----
	// Добавление нового механизма защиты = новая строка здесь + новая секция
	// в MechanismsConfig. Attendance-сервис (stage 9) получит ровно этот engine.
	policyEngine := policy.NewEngine(
		checks.NewQRTTLCheck(),
		checks.NewGeoCheck(),
		checks.NewWiFiCheck(),
	)

	// ---- Hub + Rotator Manager ----
	wsHub := hub.New(logger.With(slog.String("component", "hub")))
	rotatorMgr := appqr.NewManager(appqr.Deps{
		Sessions: sessionRepo,
		Codec:    qrCodec,
		Hub:      wsHub,
		Clock:    realClock,
		Log:      logger.With(slog.String("component", "rotator")),
	})

	// ---- Use cases ----

	auditSvc := appaudit.NewService(appaudit.Deps{
		Repo:  auditRepo,
		Clock: realClock,
	})

	authSvc := auth.NewService(auth.Deps{
		Users:      userRepo,
		Tokens:     refreshRepo,
		Hasher:     hasher,
		Signer:     signer,
		Clock:      realClock,
		Tx:         txRunner,
		Audit:      auditSvc,
		AccessTTL:  cfg.JWTAccessTTL,
		RefreshTTL: cfg.JWTRefreshTTL,
	})

	policySvc := apppolicy.NewService(apppolicy.Deps{
		Repo:  policyRepo,
		Clock: realClock,
		Tx:    txRunner,
		Audit: auditSvc,
	})

	catalogSvc := appcatalog.NewService(appcatalog.Deps{
		Courses:    courseRepo,
		Groups:     groupRepo,
		Streams:    streamRepo,
		Classrooms: classroomRepo,
		Tx:         txRunner,
		Audit:      auditSvc,
		Clock:      realClock,
	})

	sessionSvc := appsession.NewService(appsession.Deps{
		Sessions: sessionRepo,
		Streams:  streamRepo,
		Policies: policyRepo,
		Tx:       txRunner,
		Audit:    auditSvc,
		Clock:    realClock,
		Rotator:  rotatorMgr,
	})

	userSvc := appuser.NewService(appuser.Deps{
		Users:  userRepo,
		Hasher: hasher,
		Clock:  realClock,
		Tx:     txRunner,
		Audit:  auditSvc,
	})

	reportSvc := appreport.NewService(appreport.Deps{
		Repo:      reportRepo,
		Encryptor: encryptor,
	})

	attendanceSvc := appattendance.NewService(appattendance.Deps{
		Attendance: attendanceRepo,
		Sessions:   sessionRepo,
		Policies:   policyRepo,
		Classrooms: classroomRepo,
		Engine:     policyEngine,
		Codec:      qrCodec,
		Tx:         txRunner,
		Audit:      auditSvc,
		Hub:        wsHub,
		Clock:      realClock,
	})

	// ---- Bootstrap rotator'ов для ранее активных сессий ----
	bootstrapCtx, cancelBootstrap := context.WithTimeout(ctx, 10*time.Second)
	if err := rotatorMgr.Bootstrap(bootstrapCtx); err != nil {
		logger.Warn("rotator bootstrap failed", "err", err)
	}
	cancelBootstrap()

	// ---- HTTP + WS ----

	authHandler := handlers.NewAuthHandler(authSvc, logger)
	policyHandler := handlers.NewPolicyHandler(policySvc, logger)
	auditHandler := handlers.NewAuditHandler(auditSvc, logger)
	catalogHandler := handlers.NewCatalogHandler(catalogSvc, logger)
	sessionHandler := handlers.NewSessionHandler(sessionSvc, logger)
	attendanceHandler := handlers.NewAttendanceHandler(attendanceSvc, logger)
	userHandler := handlers.NewUserHandler(userSvc, logger)
	studentMeHandler := handlers.NewStudentMeHandler(attendanceSvc, logger)
	reportHandler := handlers.NewReportHandler(reportSvc, logger)

	// ---- Централизованная регистрация ошибок (httperr) ----
	httperr.RegisterAll()

	// ---- Rate limit на /auth/login (brute-force защита) ----
	loginLimiter := appmid.NewIPRateLimiter(cfg.LoginRateLimitWindow, cfg.LoginRateLimit)
	defer loginLimiter.Stop()

	teacherWS := ws.NewTeacherHandler(ws.Deps{
		Hub:      wsHub,
		Signer:   signer,
		Sessions: sessionRepo,
		Log:      logger.With(slog.String("component", "ws")),
	})

	router := apphttp.NewRouter(apphttp.Deps{
		Log:          logger,
		Signer:       signer,
		AuthH:        authHandler,
		PolicyH:      policyHandler,
		AuditH:       auditHandler,
		CatalogH:     catalogHandler,
		SessionH:     sessionHandler,
		AttendanceH:  attendanceHandler,
		UserH:        userHandler,
		StudentMeH:   studentMeHandler,
		ReportH:      reportHandler,
		WSHandler:    teacherWS.Serve,
		Health:       httpserver.HealthHandler(cfg.AppVersion),
		CORSOrigins:  cfg.CORSAllowedOrigins,
		LoginLimiter: loginLimiter,
	})

	srv := httpserver.New(httpserver.DefaultConfig(cfg.HTTPPort), router)

	err = httpserver.Run(ctx, logger, srv)
	// После HTTP-shutdown останавливаем все rotator'ы и WS-коннекты.
	rotatorMgr.Shutdown(context.Background())
	ws.Shutdown()
	wsHub.Shutdown()
	if err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	logger.Info("shutdown complete")
	return nil
}

// validateConfig — ранний sanity-check обязательных полей, чтобы не падать
// глубже с невнятной ошибкой.
func validateConfig(c config.Config) error {
	if strings.TrimSpace(c.DatabaseDSN) == "" {
		return fmt.Errorf("DATABASE_DSN is required")
	}
	if strings.TrimSpace(c.PIIEncryptionKeyB64) == "" {
		return fmt.Errorf("PII_ENCRYPTION_KEY is required (base64(32 bytes))")
	}
	if strings.TrimSpace(c.JWTAccessSecretB64) == "" {
		return fmt.Errorf("JWT_ACCESS_SECRET is required (base64 ≥32 bytes)")
	}
	if c.JWTAccessTTL <= 0 || c.JWTAccessTTL > 24*time.Hour {
		return fmt.Errorf("JWT_ACCESS_TTL must be in (0, 24h]")
	}
	if c.JWTRefreshTTL <= 0 {
		return fmt.Errorf("JWT_REFRESH_TTL must be > 0")
	}
	return nil
}
