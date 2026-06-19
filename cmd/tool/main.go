package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"storeready_ai/internal/common"
	"storeready_ai/internal/config"
	"storeready_ai/internal/infra/db"
	"storeready_ai/internal/pkg/security"
	"strings"
	"time"

	adminPermissionModel "storeready_ai/internal/admin/modules/permissions/model"
	adminPermissionRepo "storeready_ai/internal/admin/modules/permissions/repo"
	adminrolepermissionmodel "storeready_ai/internal/admin/modules/rolepermissions/model"
	adminrolepermissionrepo "storeready_ai/internal/admin/modules/rolepermissions/repo"
	adminRoleModel "storeready_ai/internal/admin/modules/roles/model"
	adminRoleRepo "storeready_ai/internal/admin/modules/roles/repo"
	"storeready_ai/internal/admin/modules/user/model"
	adminUserRepo "storeready_ai/internal/admin/modules/user/repo"
	adminuserrolemodel "storeready_ai/internal/admin/modules/userroles/model"
	adminUserRoleRepo "storeready_ai/internal/admin/modules/userroles/repo"
	billingmodel "storeready_ai/internal/client/modules/billing/model"
	billingRepo "storeready_ai/internal/client/modules/billing/repo"
	clienteventrepo "storeready_ai/internal/client/modules/client_event/repo"
	userRepo "storeready_ai/internal/client/modules/user/repo"

	"gorm.io/gorm"
)

const tenantIDEnvKey = "TENANT_ID"

func createTestUser(ctx context.Context, gdb *gorm.DB, tenantID uint64, email string) error {
	// 注意：这里直接使用 service 层的 repo 接口，绕过 service 层的业务逻辑（如密码验证、token 颁发等），仅用于测试数据准备。
	// 生产环境请勿在 app 层直接使用 repo 接口，避免绕过业务逻辑导致安全问题。
	repo := userRepo.New(gdb)
	u, _, _, err := repo.UpsertUserWithIdentity(ctx, tenantID, "test", "test-"+email, &email, nil, nil, nil)
	if err != nil {
		return fmt.Errorf("upsert test user failed: %w", err)
	}
	fmt.Printf("test user created: id=%d email=%s\n", u.ID, email)
	return nil
}

func main() {
	// common.MustInitIDs(-1, time.Time{})

	tenantID := fmt.Sprintf("%d", common.NextTenantID())
	if err := ensureTenantIDInRootEnv(tenantID); err != nil {
		panic(err)
	}

	fmt.Println(tenantID)
	loadDotEnvIfExists(".env")
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/dev.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("failed to load config: %v", err)
	}
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
		fmt.Println(err.Error())
	}
	if gdb == nil {
		if stopDB != nil {
			_ = stopDB()
		}
		fmt.Println("db open success but gdb is nil")
	}
	const tenantId = 191020851870224384
	// createTestUser(context.Background(), gdb, 157036035485401088, "test@storeready_ai.com")
	migrateDB(gdb)
	// createBillingConfig(context.Background(), gdb)
	// listClientEvents(context.Background(), gdb)
	// createSuperAdmin(context.Background(), gdb, 157036035485401088, "admin123")
	// createRolesPermissions(context.Background(), tenantId, gdb)
}

func createRolesPermissions(ctx context.Context, tenantId uint64, gdb *gorm.DB) error {
	// 这里直接使用 repo 接口，绕过 service 层的业务逻辑，仅用于测试数据准备。
	// 生产环境请勿在 app 层直接使用 repo 接口，避免绕过业务逻辑导致安全问题。
	roleRepo := adminRoleRepo.New(gdb)
	permRepo := adminPermissionRepo.New(gdb)
	// 创建角色
	err := roleRepo.Create(ctx, &adminRoleModel.AdminRole{
		ID:       tenantId,
		TenantID: tenantId,
		Name:     "超级管理员",
		Code:     "super_admin",
	})
	if err != nil {
		return fmt.Errorf("create role failed: %w", err)
	}

	// 创建权限
	err = permRepo.Create(ctx, &adminPermissionModel.AdminPermission{
		ID:       tenantId,
		TenantID: tenantId,
		Name:     "不限制权限",
		Code:     "*",
	})
	if err != nil {
		return fmt.Errorf("create permission failed: %w", err)
	}
	roleperms := adminrolepermissionrepo.New(gdb)

	// 关联角色和权限
	if err := roleperms.CreateBatch(ctx, []adminrolepermissionmodel.AdminRolePermission{
		{
			TenantID:     tenantId,
			RoleID:       tenantId,
			PermissionID: tenantId,
			CreatedAt:    uint64(time.Now().Unix()),
			UpdatedAt:    uint64(time.Now().Unix()),
		},
	}); err != nil {
		return fmt.Errorf("add role permission failed: %w", err)
	}

	userrolerepo := adminUserRoleRepo.New(gdb)
	userrolerepo.CreateBatch(ctx, []adminuserrolemodel.AdminUserRole{
		{
			ID:          tenantId,
			TenantID:    tenantId,
			AdminUserID: tenantId,
			RoleID:      tenantId,
			CreatedAt:   uint64(time.Now().Unix()),
			UpdatedAt:   uint64(time.Now().Unix()),
		},
	})

	roleIds, err := userrolerepo.ListRoleIDsByAdminUserID(ctx, tenantId, 157036035485401088)
	if err != nil {
		return fmt.Errorf("list role IDs failed: %w", err)
	}
	fmt.Printf("admin user %d role ids: %v\n", 157036035485401088, roleIds)

	//查询
	listrole, err := roleRepo.GetByID(ctx, tenantId, tenantId)
	if err != nil {
		return fmt.Errorf("get role failed: %w", err)
	}
	fmt.Printf("role added: id=%d name=%s code=%s\n", listrole.ID, listrole.Name, listrole.Code)

	listperms, err := permRepo.GetByID(ctx, tenantId, tenantId)
	if err != nil {
		return fmt.Errorf("get permission failed: %w", err)
	}
	fmt.Printf("permission added: id=%d name=%s code=%s\n", listperms.ID, listperms.Name, listperms.Code)

	listRolePerms, err := roleperms.ListByRoleID(ctx, tenantId, tenantId)
	if err != nil {
		return fmt.Errorf("list role permissions failed: %w", err)
	}
	fmt.Printf("role permission added: role_id=%d permission_id=%d listRolePerms=%d\n", tenantId, tenantId, len(listRolePerms))

	return nil
}

func createSuperAdmin(ctx context.Context, gdb *gorm.DB, tenantID uint64, password string) error {
	repo := adminUserRepo.New(gdb)
	now := time.Now().Unix()
	passwordHash, err := security.HashPassword(password, 12)
	if err != nil {
		return fmt.Errorf("hash password failed: %w", err)
	}
	err = repo.Create(ctx, &model.AdminUser{
		ID:           tenantID,
		TenantID:     tenantID,
		Username:     "superadmin",
		PasswordHash: passwordHash,
		Nickname:     "",
		Email:        "",
		Mobile:       "",
		Avatar:       "",
		Status:       0, // 0-正常，1-禁用
		IsSuperAdmin: 1, //0-正常，1-超级管理员（目前仅有一个租户管理员，且 ID 固定为 tenantID，因此直接用 ID 作为主键）
		LastLoginAt:  0,
		LastLoginIP:  "",
		Remark:       "",
		CreatedAt:    uint64(now),
		UpdatedAt:    uint64(now),
		DeletedAt:    0,
	})
	if err != nil {
		return fmt.Errorf("upsert super admin failed: %w", err)
	}

	return nil
}

func createBillingConfig(ctx context.Context, gdb *gorm.DB) error {
	repo := billingRepo.NewRepos(gdb)

	now := uint64(0)
	products := []*billingmodel.BillingProduct{
		{
			TenantID:          157036035485401088,
			ProductCode:       "vip_monthly",
			Platform:          "android",
			StoreProductID:    "vip_monthly",
			ProductType:       "subscription",
			SubscriptionGroup: "vip",
			Status:            1,
			IsRecommended:     false,
			Sort:              10,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
		{
			TenantID:          157036035485401088,
			ProductCode:       "vip_yearly",
			Platform:          "android",
			StoreProductID:    "vip_yearly",
			ProductType:       "subscription",
			SubscriptionGroup: "vip",
			Status:            1,
			IsRecommended:     true,
			Sort:              20,
			CreatedAt:         now,
			UpdatedAt:         now,
		},
	}

	for _, product := range products {
		if err := repo.Products.Create(ctx, product); err != nil {
			return fmt.Errorf("create billing product %s failed: %w", product.ProductCode, err)
		}
	}
	return nil
}

func listClientEvents(ctx context.Context, gdb *gorm.DB) {
	// 查询最近埋点记录，用于本地快速排查 billing / sync / login 等客户端问题。
	repo := clienteventrepo.NewClientEventRepo(gdb)
	items, err := repo.List(ctx, clienteventrepo.ListClientEventsOption{
		TenantID: 157036035485401088,
		Limit:    50,
	})
	if err != nil {
		fmt.Printf("list client events failed: %v\n", err)
		return
	}

	fmt.Printf("client events total=%d\n", len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		fmt.Printf(
			"id=%d tenant_id=%d uid=%d group=%s name=%s code=%s platform=%s created_at=%d message=%s payload=%s\n",
			item.ID,
			item.TenantID,
			item.UID,
			item.EventGroup,
			item.EventName,
			item.EventCode,
			item.Platform,
			item.CreatedAt,
			item.EventMessage,
			item.Payload,
		)
	}
}

func migrateDB(gdb *gorm.DB) {
	migrateCfg := db.MigrateConfig{
		MigrationsDir: "./migrations",
		TableName:     "schema_migrations",
	}
	fmt.Printf("[migrate] config: dir=%s table=%s\n", migrateCfg.MigrationsDir, migrateCfg.TableName)

	fmt.Println("[migrate] checking current version...")
	version, dirty, err := db.Version(gdb, migrateCfg)
	if err != nil {
		fmt.Printf("[migrate] version failed: %v\n", err)
		return
	}
	fmt.Printf("[migrate] current version=%d dirty=%v\n", version, dirty)

	if dirty {
		fmt.Printf("[migrate] dirty database detected at version=%d\n", version)
		fmt.Println("[migrate] forcing version to 4...")
		if err := db.Force(gdb, migrateCfg, 4); err != nil {
			fmt.Printf("[migrate] force failed: %v\n", err)
			return
		}
		fmt.Println("[migrate] force success")

		fmt.Println("[migrate] re-checking version after force...")
		version, dirty, err = db.Version(gdb, migrateCfg)
		if err != nil {
			fmt.Printf("[migrate] version after force failed: %v\n", err)
			return
		}
		fmt.Printf("[migrate] after force version=%d dirty=%v\n", version, dirty)
	}

	fmt.Println("[migrate] running up migrations...")
	if err := db.Up(gdb, migrateCfg); err != nil {
		fmt.Printf("[migrate] up failed: %v\n", err)
		return
	}
	fmt.Println("[migrate] up success")

	fmt.Println("[migrate] checking final version...")
	version, dirty, err = db.Version(gdb, migrateCfg)
	if err != nil {
		fmt.Printf("[migrate] final version check failed: %v\n", err)
		return
	}
	fmt.Printf("[migrate] final version=%d dirty=%v\n", version, dirty)

	fmt.Println("[migrate] done")
}

func ensureTenantIDInRootEnv(tenantID string) error {
	rootEnvPath, err := filepath.Abs(".env")
	if err != nil {
		return fmt.Errorf("resolve .env path failed: %w", err)
	}
	fmt.Printf("root .env path: %s\n", rootEnvPath)

	lines, exists, err := readEnvLines(rootEnvPath)
	if err != nil {
		return err
	}

	if hasTenantID(lines) {
		return nil
	}

	if !exists {
		content := tenantIDEnvKey + "=" + tenantID + "\n"
		if err := os.WriteFile(rootEnvPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("create .env failed: %w", err)
		}
		return nil
	}

	content := strings.Join(lines, "\n")
	if len(lines) > 0 && !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += tenantIDEnvKey + "=" + tenantID + "\n"

	if err := os.WriteFile(rootEnvPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write .env failed: %w", err)
	}
	return nil
}

func readEnvLines(path string) ([]string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("open .env failed: %w", err)
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, true, fmt.Errorf("read .env failed: %w", err)
	}

	return lines, true, nil
}

func hasTenantID(lines []string) bool {
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(trimmed, tenantIDEnvKey+"=") {
			return true
		}
	}
	return false
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
