package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	adminjwt "storeready_ai/internal/admin/auth/jwt"
	adminhandler "storeready_ai/internal/admin/handler"
	adminappuserhandler "storeready_ai/internal/admin/modules/appuser/handler"
	adminappuserservice "storeready_ai/internal/admin/modules/appuser/service"
	adminaudithandler "storeready_ai/internal/admin/modules/audit/handler"
	adminauditrepo "storeready_ai/internal/admin/modules/audit/repo"
	adminauditservice "storeready_ai/internal/admin/modules/audit/service"
	adminauthhandler "storeready_ai/internal/admin/modules/auth/handler"
	authservice "storeready_ai/internal/admin/modules/auth/service"
	adminfeedbackhandler "storeready_ai/internal/admin/modules/feedback/handler"
	adminfeedbackrepo "storeready_ai/internal/admin/modules/feedback/repo"
	adminfeedbackservice "storeready_ai/internal/admin/modules/feedback/service"
	adminmehandler "storeready_ai/internal/admin/modules/me/handler"
	meservice "storeready_ai/internal/admin/modules/me/service"
	adminpermissionhandler "storeready_ai/internal/admin/modules/permissions/handler"
	adminpermissionrepo "storeready_ai/internal/admin/modules/permissions/repo"
	adminpermissionservice "storeready_ai/internal/admin/modules/permissions/service"
	rbac "storeready_ai/internal/admin/modules/rbac/service"

	adminrolepermissionrepo "storeready_ai/internal/admin/modules/rolepermissions/repo"
	adminrolepermissionservice "storeready_ai/internal/admin/modules/rolepermissions/service"
	adminrolehandler "storeready_ai/internal/admin/modules/roles/handler"
	adminrolerepo "storeready_ai/internal/admin/modules/roles/repo"
	adminrolesservice "storeready_ai/internal/admin/modules/roles/service"

	adminuserhandler "storeready_ai/internal/admin/modules/user/handler"
	adminuserrepo "storeready_ai/internal/admin/modules/user/repo"
	adminuserservice "storeready_ai/internal/admin/modules/user/service"
	adminuserrolerepo "storeready_ai/internal/admin/modules/userroles/repo"
	adminuserroleservice "storeready_ai/internal/admin/modules/userroles/service"
	"storeready_ai/internal/app/bootstrap"
	"storeready_ai/internal/client/auth/jwt"
	"storeready_ai/internal/client/router"
	"storeready_ai/internal/common"
	"storeready_ai/internal/config"
	"storeready_ai/internal/i18n"
	"storeready_ai/internal/infra/audit"
	"storeready_ai/internal/infra/cache"
	"storeready_ai/internal/infra/db"
	"storeready_ai/internal/infra/securityevent"
	"storeready_ai/internal/pkg/ai"
	"storeready_ai/internal/pkg/security"

	httphandler "storeready_ai/internal/client/http/handler"
	devicerepo "storeready_ai/internal/client/modules/device/repo"
	devicesvc "storeready_ai/internal/client/modules/device/service"
	entrepo "storeready_ai/internal/client/modules/entitlement/repo"
	entsvc "storeready_ai/internal/client/modules/entitlement/service"
	feedbackhandler "storeready_ai/internal/client/modules/feedback/handler"
	fbclient "storeready_ai/internal/client/modules/firebase"
	projecthandler "storeready_ai/internal/client/modules/project/handler"
	projectrepo "storeready_ai/internal/client/modules/project/repo"
	projectsvc "storeready_ai/internal/client/modules/project/service"
	settingsrepo "storeready_ai/internal/client/modules/settings/repo"
	settingssvc "storeready_ai/internal/client/modules/settings/service"
	userhandler "storeready_ai/internal/client/modules/user/handler"
	userrepo "storeready_ai/internal/client/modules/user/repo"
	usersvc "storeready_ai/internal/client/modules/user/service"

	billingplatform "storeready_ai/internal/client/modules/billing/platform"
	billingrepo "storeready_ai/internal/client/modules/billing/repo"
	billingsvc "storeready_ai/internal/client/modules/billing/service"
	clienteventrepo "storeready_ai/internal/client/modules/client_event/repo"
	clienteventsvc "storeready_ai/internal/client/modules/client_event/service"
	feedbackrepo "storeready_ai/internal/client/modules/feedback/repo"
	feedbacksvc "storeready_ai/internal/client/modules/feedback/service"

	infrolog "storeready_ai/internal/infra/log"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// ---- TokenIssuer adapter ----
// UserService 需要一个 `IssueAccessToken(uid, tenantID)` 的颁发器接口；而 bootstrap.NewJWTManager 返回的是 *jwt.Manager。
// 为了保持 Service 层只依赖接口，这里在 app 层做一个轻量适配。

type tokenIssuerAdapter struct {
	fn func(uid uint64, tenantID uint64) (string, time.Time, error)
}

func (a tokenIssuerAdapter) IssueAccessToken(uid uint64, tenantID uint64) (string, time.Time, error) {
	if a.fn == nil {
		return "", time.Time{}, fmt.Errorf("jwt: IssueAccessToken adapter not configured")
	}
	return a.fn(uid, tenantID)
}

// adminPasswordHasherAdapter 适配 pkg/security 的包级函数到 admin auth service 所需接口。
type adminPasswordHasherAdapter struct{}

func (adminPasswordHasherAdapter) HashPassword(password string, cost int) (string, error) {
	return security.HashPassword(password, cost)
}

func (adminPasswordHasherAdapter) ComparePassword(hash, password string) bool {
	return security.ComparePassword(hash, password)
}

// buildTokenIssuerAdapter adapts *jwt.Manager to usersvc.TokenIssuer using configured accessTTL.
func buildTokenIssuerAdapter(m *jwt.Manager, accessTTL time.Duration) (usersvc.TokenIssuer, error) {
	if m == nil {
		return nil, fmt.Errorf("jwt manager is nil")
	}
	if accessTTL <= 0 {
		accessTTL = time.Hour
	}

	// 适配真实实现：func(in jwt.SignInput) (string, error)
	// 这里不把 jwt.SignInput 泄漏到 service 层，只在 app 层构造。
	type vSignInput interface {
		SignAccessToken(in jwt.SignInput) (string, error)
	}
	x, ok := any(m).(vSignInput)
	if !ok {
		return nil, fmt.Errorf("jwt manager does not implement SignAccessToken(jwt.SignInput) (string, error)")
	}

	return tokenIssuerAdapter{fn: func(uid uint64, tenantID uint64) (string, time.Time, error) {
		in := jwt.SignInput{}

		// SignInput 里 UID/TenantID 是 string，这里按约定填充：
		// - UID: 使用对外 uid 的十进制字符串
		// - TenantID: 由上层显式透传，避免在 app 层写死默认值
		uidStr := fmt.Sprintf("%d", uid)
		tenantIDStr := fmt.Sprintf("%d", tenantID)
		setStringField(&in, "UID", uidStr)
		setStringField(&in, "TenantID", tenantIDStr)

		// Subject 为空时，常见默认是 UID；这里也做一次兜底。
		setStringFieldIfEmpty(&in, "Subject", uidStr)

		token, err := x.SignAccessToken(in)
		exp := time.Now().Add(accessTTL)
		return token, exp, err
	}}, nil
}

func setStringField(ptr any, field string, v string) {
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	ev := rv.Elem()
	if ev.Kind() != reflect.Struct {
		return
	}
	fv := ev.FieldByName(field)
	if !fv.IsValid() || !fv.CanSet() {
		return
	}
	if fv.Kind() == reflect.String {
		fv.SetString(v)
	}
}

func setStringFieldIfEmpty(ptr any, field string, v string) {
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	ev := rv.Elem()
	if ev.Kind() != reflect.Struct {
		return
	}
	fv := ev.FieldByName(field)
	if !fv.IsValid() || !fv.CanSet() {
		return
	}
	if fv.Kind() == reflect.String {
		if fv.String() == "" {
			fv.SetString(v)
		}
	}
}

func setInt64Field(ptr any, field string, v int64) {
	rv := reflect.ValueOf(ptr)
	if rv.Kind() != reflect.Pointer || rv.IsNil() {
		return
	}
	ev := rv.Elem()
	if ev.Kind() != reflect.Struct {
		return
	}
	fv := ev.FieldByName(field)
	if !fv.IsValid() || !fv.CanSet() {
		return
	}
	if fv.Kind() == reflect.Int64 {
		fv.SetInt(v)
	}
}

// App 应用容器：负责装配依赖、启动/停止 HTTP Server、释放资源。
//
// 设计原则：
//   - app.go 只做“装配/生命周期管理”，不写业务逻辑
//   - 依赖释放统一集中在 Stop
//   - 保持外部调用 API 稳定（New/Start/Stop）
type App struct {
	Cfg  *config.Config
	I18n *i18n.Translator

	LN   net.Listener
	HTTP *http.Server
	// Redis 客户端（用于缓存/限流/防刷等）
	Redis redis.UniversalClient
	// RedisStop 关闭 Redis 资源
	RedisStop func() error
	// Cache 统一缓存封装（带前缀/默认TTL/JSON便捷方法）
	Cache  *cache.RedisCache
	DBStop func() error
}

// New 使用默认配置路径（""）创建应用。
func New() (*App, error) {
	return NewWithPath("")
}

// NewWithPath 使用指定配置路径创建应用（便于测试/多环境）。
func NewWithPath(cfgPath string) (*App, error) {
	loadDotEnvIfExists(".env")
	if cfgPath == "" {
		configPath := os.Getenv("CONFIG_PATH")
		if configPath != "" {
			cfgPath = configPath
		}
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		// 此时日志系统尚未初始化，直接输出到 stderr
		_, _ = os.Stderr.WriteString("加载配置失败：" + err.Error() + "\n")
		os.Exit(1)
	}
	log_path := os.Getenv("LOG_PATH")
	// 初始化企业级日志（zap）
	logger, err := infrolog.Init(infrolog.Config{
		Service:          cfg.Server.Name,
		Env:              cfg.Server.Mode,
		Level:            cfg.Observability.Log.Level,
		JSON:             cfg.Observability.Log.JSON,
		Caller:           true,
		Stacktrace:       true,
		Sampling:         false,
		OutputPaths:      []string{"stdout", log_path + "/app.log"},
		ErrorOutputPaths: []string{"stderr", log_path + "/app.err.log"},
		RedactKeys:       []string{"password", "token", "authorization", "jwt", "secret", "sign_key"},
	})
	if err != nil {
		_, _ = os.Stderr.WriteString("初始化日志失败：" + err.Error() + "\n")
		os.Exit(1)
	}

	// 将当前 logger 设置为全局 zap logger，保证 middleware/第三方组件使用 zap.L() 时也会写入同一份日志。
	zap.ReplaceGlobals(logger)
	// 可选：把标准库 log.* 也重定向到 zap（方便统一输出）；不影响 fmt.Println。
	_ = zap.RedirectStdLog(logger)

	// 初始化 ID 生成器（如果有需要的话）
	common.MustInitIDs(-1, time.Time{})

	// 初始化 i18n（服务端错误提示 / 通知文案）
	i18nBundle, err := i18n.NewBundle(
		"internal/i18n/messages",
		[]string{i18n.LocaleENUS, i18n.LocaleZHCN, i18n.LocaleZHHK, i18n.LocaleJAJP},
	)
	if err != nil {
		return nil, fmt.Errorf("init i18n failed: %w", err)
	}
	i18nTranslator := i18n.NewTranslator(i18nBundle, i18n.LocaleENUS)

	// gdb, stopDB, err := db.NewGorm(cfg)
	// if err != nil {
	// 	return nil, err
	// }
	gdb, stopDB, err := db.OpenMySQL(context.Background(), cfg.DB.DSN, db.PoolConfig{
		MaxOpenConns:    cfg.DB.MaxOpenConns,
		MaxIdleConns:    cfg.DB.MaxIdleConns,
		ConnMaxLifetime: cfg.DB.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.DB.ConnMaxIdleTime,
	}, db.MySQLOptions{
		// PrepareStmt：预编译语句缓存。
		// 优点：减少重复 prepare 开销；缺点：会增加内存占用（且在高并发/大量不同 SQL 时可能放大内存）。
		// 建议：默认关闭；确认热点 SQL 明显、且内存评估可控后再开启。
		PrepareStmt: false,

		// SlowThreshold：慢 SQL 阈值（用于日志/观测）。
		// 建议：与配置 db.slow_threshold 对齐。
		SlowThreshold: cfg.DB.SlowThreshold,

		// LogLevel：GORM 日志级别。
		// 建议：生产环境通常使用 Warn；开发环境可用 Info。
		// 说明：这里保持为 0（由 db 层默认值决定），避免在 app 层硬绑定具体枚举。
		LogLevel: 0,

		// DisableDefaultTransaction：禁用 GORM 默认事务。
		// 优点：写多场景吞吐更高；风险：如果业务依赖“每次写都隐式事务”，需要谨慎。
		// 建议：默认关闭（安全优先）；在确认业务不依赖隐式事务后可开启。
		DisableDefaultTransaction: false,

		// SkipPing：是否跳过启动时 Ping。
		// 企业最佳实践：启动时带超时 Ping（因此默认不跳过）。
		SkipPing: false,
	})

	// 必须先处理 DB 初始化错误，避免后续 err 被覆盖导致 gdb=nil 继续装配（会在运行期触发空指针崩溃）
	if err != nil {
		return nil, err
	}
	if gdb == nil {
		if stopDB != nil {
			_ = stopDB()
		}
		return nil, fmt.Errorf("db open success but gdb is nil")
	}

	rdb, stopRedis, err := cache.NewRedis(cfg.Redis)
	if err != nil {
		_ = stopDB()
		return nil, err
	}
	// 默认 key 前缀使用服务名，默认 TTL 10 分钟（业务可自行覆盖 ttl）
	rc := cache.NewRedisCache(rdb, cfg.Server.Name, 10*time.Minute)

	// 组装 App JWT（只在 app 层做依赖装配；router 只接线）
	jwtMgr, err := bootstrap.NewJWTManager(cfg.Security)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}
	issuer, err := buildTokenIssuerAdapter(jwtMgr, cfg.Security.JWT.AccessTTL)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}

	// adminHandler := adminhandler.New()

	// -------------------------
	// Firebase / User / Auth 装配（MVP：HTTP；后期 gRPC 复用 Service/Repo）
	// -------------------------

	// Firebase client（用于验签 Firebase ID Token）
	var fb *fbclient.Client
	if cfg.Firebase.Auth.Enabled {
		fb, err = fbclient.NewClient(context.Background(), cfg.Firebase)
		if err != nil {
			_ = stopRedis()
			_ = stopDB()
			return nil, err
		}
	}
	///========================增加repo,service,handler 开始=======================================================///
	// user repo + service
	userRepo := userrepo.New(gdb)
	// refresh token TTL：这里先给默认 30 天；后续可放到配置里
	userService := usersvc.New(userRepo, fb, issuer, 30*24*time.Hour)

	// auth handler
	authHandler := httphandler.NewAuthHandler(userService)

	// entitlement repo（用于 VIP 判定，订阅创建时做免费条数限制；entitlement service 也复用）
	entRepo := entrepo.New(gdb)

	// settings repo + service + handler
	sr := settingsrepo.New(gdb)
	settingsService := settingssvc.New(sr)
	settingsHandler := httphandler.NewSettingsHandler(settingsService)

	// device repo + service + handler
	dr := devicerepo.New(gdb)
	deviceService := devicesvc.New(dr)
	deviceHandler := httphandler.NewDeviceHandler(deviceService)

	// entitlement repo + service + handler
	er := entRepo
	entitlementService := entsvc.New(er)
	entitlementHandler := httphandler.NewEntitlementHandler(entitlementService)

	// -------------------------
	// Billing 装配
	//
	// 说明：
	// 1. 当前在 app 层完成 GooglePlay executor -> client -> verifier -> billing service -> handler 的组装；
	// 2. Billing 配置统一从 cfg.Billing 读取；
	// 3. 若未启用 billing 或未完整配置 google_play，则 billing 仍可启动，
	//    但 verify/restore 会在 service 层返回 "billing verifier not configured"；
	// 4. entitlement 继续复用现有 entitlementService，billing 不单独持有权益真相。
	// -------------------------
	billingRepos := billingrepo.NewRepos(gdb)

	var billingVerifier billingsvc.PlatformVerifier
	if cfg.Billing.Enabled {
		if cfg.Billing.GooglePlay.Enabled {
			googlePlayPackageName := strings.TrimSpace(cfg.Billing.GooglePlay.PackageName)
			googlePlayCredentialsFile := strings.TrimSpace(cfg.Billing.GooglePlay.CredentialsFile)
			googlePlayScopes := cfg.Billing.GooglePlay.Scopes
			if googlePlayPackageName != "" && googlePlayCredentialsFile != "" {
				googleExecutor, execErr := billingplatform.NewGooglePlaySubscriptionExecutor(
					billingplatform.GooglePlayExecutorConfig{
						CredentialsFile: googlePlayCredentialsFile,
						Scopes:          googlePlayScopes,
					},
				)
				if execErr != nil {
					logger.Warn("billing google play executor init failed",
						zap.String("credentials_file", googlePlayCredentialsFile),
						zap.Error(execErr),
					)
				} else {
					googleClient := billingplatform.NewGooglePlayClient(
						googlePlayPackageName,
						googleExecutor,
					)
					billingVerifier = billingsvc.NewGooglePlatformVerifier(googleClient)
					logger.Info("billing google verifier enabled",
						zap.String("package_name", googlePlayPackageName),
					)
				}
			} else {
				logger.Warn("billing google verifier disabled: incomplete config",
					zap.Bool("has_package_name", googlePlayPackageName != ""),
					zap.Bool("has_credentials_file", googlePlayCredentialsFile != ""),
				)
			}
		} else {
			logger.Info("billing google verifier disabled: google_play.enabled=false")
		}
	} else {
		logger.Info("billing disabled: billing.enabled=false")
	}

	defaultEntitlementCode := strings.TrimSpace(cfg.Billing.DefaultEntitlementCode)
	if defaultEntitlementCode == "" {
		defaultEntitlementCode = "vip"
	}
	billingService := billingsvc.New(billingRepos, entitlementService, billingVerifier, defaultEntitlementCode)
	billingHandler := httphandler.NewBillingHandler(billingService, *i18nTranslator)

	clienteventrepos := clienteventrepo.NewClientEventRepo(gdb)
	clientEventService := clienteventsvc.New(clienteventrepos)
	clientEventHandler := httphandler.NewClientEventHandler(clientEventService)

	feedbackRepo := feedbackrepo.NewRepository(gdb)
	feedbackSvc := feedbacksvc.NewService(feedbackRepo)
	feedbackHandler := feedbackhandler.NewHandler(feedbackSvc)

	projectRepo := projectrepo.New(gdb)

	var storeInfoAIGen ai.StoreInfoGenerator
	if cfg.AI.Enabled {
		storeInfoAIGen, err = ai.NewStoreInfoGenerator(ai.Config{
			Provider:       cfg.AI.Provider,
			Model:          cfg.AI.Model,
			APIKey:         cfg.AI.APIKey,
			Endpoint:       cfg.AI.Endpoint,
			TimeoutSeconds: cfg.AI.TimeoutSeconds,
		})
		if err != nil {
			_ = stopRedis()
			_ = stopDB()
			return nil, err
		}
		logger.Info("project store info ai generator enabled",
			zap.String("provider", cfg.AI.Provider),
			zap.String("model", cfg.AI.Model),
		)
	} else {
		logger.Info("project store info ai generator disabled: ai.enabled=false")
	}

	projectService := projectsvc.New(projectRepo, storeInfoAIGen)
	projectHandler := projecthandler.NewProjectHandler(projectService)

	userHandler := userhandler.NewUserHandler(userService)

	// 组装 Admin JWT：先在 app 层初始化，后续 admin router / middleware 接入时直接复用。
	// 当前先完成配置校验与管理器装配，避免 admin 配置遗漏到运行期才暴露。
	adminJWTMgr, err := adminjwt.NewManager(
		cfg.Security.AdminJWT.Issuer,
		cfg.Security.AdminJWT.HMACSecret,
		cfg.Security.AdminJWT.AccessTTL,
		cfg.Security.AdminJWT.RefreshTTL,
	)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}
	// -------------------------
	// Admin 模块装配
	// -------------------------
	adminUserRepo := adminuserrepo.New(gdb)
	adminPasswordHasher := adminPasswordHasherAdapter{}

	// MVP 阶段：admin 角色提供器与 refresh token revoke 先留空实现。
	// 后续补齐 role service / token blacklist 后，直接在这里替换即可。
	var adminRoleProvider authservice.RoleProvider
	// Admin UserRole 模块
	adminUserRoleRepo := adminuserrolerepo.New(gdb)
	adminUserRoles, err := adminuserroleservice.New(adminUserRoleRepo)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}
	// Admin Role 模块
	adminRoleRepo := adminrolerepo.New(gdb)
	adminRoleService, err := adminrolesservice.New(adminRoleRepo)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}

	adminuserservicex, err := adminuserservice.New(adminUserRepo, adminPasswordHasher)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}

	adminRoleHandler := adminrolehandler.New(adminRoleService)

	// Admin Permission 模块
	adminPermissionRepo := adminpermissionrepo.New(gdb)
	adminPermissionService, err := adminpermissionservice.New(adminPermissionRepo)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}
	// TODO: 补齐 permission repo/service 后替换为 New(service) 形式。
	adminPermissionHandler := adminpermissionhandler.New(adminPermissionService)

	// Admin Audit 模块
	adminauditrepo := adminauditrepo.New(gdb)
	adminAuditService, err := adminauditservice.New(adminauditrepo)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}
	// TODO: 补齐 audit repo/service 后替换为 New(service) 形式。
	adminAuditHandler := adminaudithandler.New(adminAuditService)

	// Admin RolePermission 模块
	adminRolePermissionRepo := adminrolepermissionrepo.New(gdb)
	adminRolePermission, err := adminrolepermissionservice.New(adminRolePermissionRepo)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}
	rbacsv := rbac.New(adminuserservicex, adminUserRoles, adminRoleService, adminRolePermission, adminPermissionService)
	meService := meservice.New(rbacsv)
	adminMeHandler := adminmehandler.New(meService)

	adminAppUserService := adminappuserservice.New(userService)
	adminAppUserHandler := adminappuserhandler.New(adminAppUserService)

	adminFeedbackRepo := adminfeedbackrepo.NewRepository(gdb)
	adminFeedbackService := adminfeedbackservice.NewService(adminFeedbackRepo)
	adminFeedbackHandler := adminfeedbackhandler.New(adminFeedbackService)

	///========================增加repo,service,handler 结束=======================================================///

	adminRoleProvider = rbacsv
	adminTokenIssuer, err := authservice.NewTokenIssuer(
		adminJWTMgr,
		func(ctx context.Context, tenantID uint64, adminUserID uint64) (authservice.AdminUserSnapshot, error) {
			u, getErr := adminUserRepo.GetByID(ctx, tenantID, adminUserID)
			if getErr != nil {
				return authservice.AdminUserSnapshot{}, getErr
			}
			if u == nil {
				return authservice.AdminUserSnapshot{}, authservice.ErrAdminUserNotFound
			}
			return authservice.AdminUserSnapshot{
				ID:           u.ID,
				Username:     u.Username,
				Status:       u.Status,
				IsSuperAdmin: u.IsSuperAdmin,
			}, nil
		},
		func(ctx context.Context, tenantID uint64, adminUserID uint64) ([]string, error) {
			if adminRoleProvider == nil {
				return nil, nil
			}
			return adminRoleProvider.GetRoleCodesByAdminUserID(ctx, tenantID, adminUserID)
		},
		nil,
	)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}

	adminAuthService, err := authservice.New(
		adminUserRepo,
		adminPasswordHasher,
		adminTokenIssuer,
		adminRoleProvider,
	)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}

	adminHTTPHandler := &adminhandler.Handler{
		AuthHandler:       adminauthhandler.New(adminAuthService),
		UserHandler:       adminuserhandler.New(adminuserservicex),
		RoleHandler:       adminRoleHandler,
		PermissionHandler: adminPermissionHandler,
		AuditHandler:      adminAuditHandler,
		PermHandler:       adminPermissionHandler,
		RolesHandler:      adminRoleHandler,
		MeHandler:         adminMeHandler,
		AppUserHandler:    adminAppUserHandler,
		FeedbackHandler:   adminFeedbackHandler,
	}

	engine := router.New(router.Deps{
		Cfg:            cfg,
		I18n:           i18nTranslator,
		DB:             gdb,
		Redis:          rdb,
		AuthJWT:        jwtMgr,
		AdminJWT:       adminJWTMgr,
		AdminHandler:   adminHTTPHandler,
		AuditWriter:    audit.NewGormAuditWriter(gdb),
		SecurityWriter: securityevent.NewGormSecurityEventWriter(gdb),
		AuthHandler:    authHandler,

		SettingsHandler: settingsHandler,

		DeviceHandler:      deviceHandler,
		BillingHandler:     billingHandler,
		EntitlementHandler: entitlementHandler,

		ClientEventHandler: clientEventHandler,
		Logger:             logger,
		FeedbackHandler:    feedbackHandler,
		UserHandler:        userHandler,
		ProjectHandler:     projectHandler,
		UserIDResolver: func(ctx context.Context, tenantID, uid string) (uint64, bool, error) {
			tenantID = strings.TrimSpace(tenantID)
			uid = strings.TrimSpace(uid)
			if tenantID == "" || uid == "" {
				return 0, false, nil
			}

			tid, parseTenantErr := strconv.ParseUint(tenantID, 10, 64)
			if parseTenantErr != nil || tid == 0 {
				return 0, false, parseTenantErr
			}

			bizUID, parseUIDErr := strconv.ParseUint(uid, 10, 64)
			if parseUIDErr != nil || bizUID == 0 {
				return 0, false, parseUIDErr
			}

			u, getErr := userRepo.GetUserByUID(ctx, tid, bizUID)
			if getErr != nil {
				return 0, false, getErr
			}
			if u == nil || u.ID == 0 {
				return 0, false, nil
			}

			return u.ID, true, nil
		}, // 供 router 层 JWT middleware 解析用户 ID 和租户 ID 使用
	})

	srv := buildHTTPServer(cfg, engine)

	addr := ":8080"
	if strings.TrimSpace(cfg.Server.Listen) != "" {
		addr = strings.TrimSpace(cfg.Server.Listen)
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		_ = stopRedis()
		_ = stopDB()
		return nil, err
	}
	if cfg.Server.HTTP.Limits.MaxConns > 0 {
		ln = newLimitListener(ln, cfg.Server.HTTP.Limits.MaxConns)
	}

	return &App{
		Cfg:       cfg,
		I18n:      i18nTranslator,
		LN:        ln,
		HTTP:      srv,
		Redis:     rdb,
		RedisStop: stopRedis,
		Cache:     rc,
		DBStop:    stopDB,
	}, nil
}

// Start 启动 HTTP 服务。
//
// 注意：
//   - http.ErrServerClosed 代表正常关闭，不视为错误
func (a *App) Start() error {
	if a == nil || a.HTTP == nil {
		return errors.New("app: http server is nil")
	}
	var err error
	if a.LN != nil {
		err = a.HTTP.Serve(a.LN)
	} else {
		err = a.HTTP.ListenAndServe()
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

// Stop 优雅停止：先关闭 HTTP，再释放 DB 等资源。
//
// 说明：
//   - ctx 由上层传入（例如 signal handler）
//   - 如果上层没有设置超时，建议使用 cfg.Server.GracefulTimeout 作为兜底
func (a *App) Stop(ctx context.Context) error {
	if a == nil {
		return nil
	}

	// 兜底：如果没有传入 ctx，则给一个合理超时
	if ctx == nil {
		ctx = context.Background()
	}

	// 兜底：为 Shutdown 增加超时（避免永久阻塞）
	if a.Cfg != nil {
		gt := a.Cfg.Server.GracefulTimeout
		if gt > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, gt)
			defer cancel()
		}
	}

	// 1) 先关闭 HTTP
	if a.HTTP != nil {
		_ = a.HTTP.Shutdown(ctx)
	}

	// 2) 关闭 Redis
	if a.RedisStop != nil {
		_ = a.RedisStop()
	}

	// 3) 刷新日志缓冲
	infrolog.Sync()

	// 4) 再释放 DB
	if a.DBStop != nil {
		_ = a.DBStop()
	}

	return nil
}

// buildHTTPServer 统一构造 http.Server，并把配置映射到 Server 参数。
func buildHTTPServer(cfg *config.Config, handler http.Handler) *http.Server {
	addr := ":8080"
	readTimeout := 5 * time.Second
	writeTimeout := 10 * time.Second
	idleTimeout := 60 * time.Second
	maxHeaderBytes := 1048576
	readHeaderTimeout := 2 * time.Second

	if cfg != nil {
		if strings.TrimSpace(cfg.Server.Listen) != "" {
			addr = strings.TrimSpace(cfg.Server.Listen)
		}
		if cfg.Server.ReadTimeout > 0 {
			readTimeout = cfg.Server.ReadTimeout
		}
		if cfg.Server.WriteTimeout > 0 {
			writeTimeout = cfg.Server.WriteTimeout
		}
		if cfg.Server.IdleTimeout > 0 {
			idleTimeout = cfg.Server.IdleTimeout
		}
		if cfg.Server.HTTP.Limits.MaxHeaderBytes > 0 {
			maxHeaderBytes = int(cfg.Server.HTTP.Limits.MaxHeaderBytes)
		}
		if cfg.Server.HTTP.Limits.ReadHeaderTimeout > 0 {
			readHeaderTimeout = cfg.Server.HTTP.Limits.ReadHeaderTimeout
		}
	}

	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		MaxHeaderBytes:    maxHeaderBytes,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}

// limitListener 用于限制并发连接数。
//
// 说明：
//   - max<=0 表示不限制
//   - 这是一个轻量兜底，真正高阶的连接治理建议交给网关/LB

type limitListener struct {
	inner net.Listener
	sem   chan struct{}
}

func newLimitListener(inner net.Listener, max int) net.Listener {
	if max <= 0 {
		return inner
	}
	return &limitListener{inner: inner, sem: make(chan struct{}, max)}
}

func (l *limitListener) Accept() (net.Conn, error) {
	c, err := l.inner.Accept()
	if err != nil {
		return nil, err
	}
	select {
	case l.sem <- struct{}{}:
		return &limitConn{Conn: c, release: func() { <-l.sem }}, nil
	default:
		_ = c.Close()
		return nil, errors.New("too many connections")
	}
}

func (l *limitListener) Close() error   { return l.inner.Close() }
func (l *limitListener) Addr() net.Addr { return l.inner.Addr() }

type limitConn struct {
	net.Conn
	releaseOnce bool
	release     func()
}

func (c *limitConn) Close() error {
	if !c.releaseOnce {
		c.releaseOnce = true
		if c.release != nil {
			c.release()
		}
	}
	return c.Conn.Close()
}

func loadDotEnvIfExists(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)

		if key == "" || os.Getenv(key) != "" {
			continue
		}

		_ = os.Setenv(key, value)
	}
}
