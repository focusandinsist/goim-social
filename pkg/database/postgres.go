package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgreSQL PostgreSQL连接管理器
type PostgreSQL struct {
	db     *gorm.DB
	sqlDB  *sql.DB
	dbName string
}

// NewPostgreSQL 创建PostgreSQL连接
func NewPostgreSQL(dsn, dbName string) (*PostgreSQL, error) {
	// 首先尝试创建数据库（如果不存在）
	if err := createDatabaseIfNotExists(dsn, dbName); err != nil {
		return nil, fmt.Errorf("failed to create database: %v", err)
	}

	// 配置GORM
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().Local()
		},
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to PostgreSQL: %v", err)
	}

	// 获取底层sql.DB对象
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get sql.DB: %v", err)
	}

	// 配置连接池
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping PostgreSQL: %v", err)
	}

	return &PostgreSQL{
		db:     db,
		sqlDB:  sqlDB,
		dbName: dbName,
	}, nil
}

// GetDB 获取GORM数据库实例
func (p *PostgreSQL) GetDB() *gorm.DB {
	return p.db
}

// GetSQLDB 获取原生SQL数据库实例
func (p *PostgreSQL) GetSQLDB() *sql.DB {
	return p.sqlDB
}

// GetDBName 获取数据库名称
func (p *PostgreSQL) GetDBName() string {
	return p.dbName
}

// Transaction 执行事务
func (p *PostgreSQL) Transaction(fn func(*gorm.DB) error) error {
	return p.db.Transaction(fn)
}

// AutoMigrate 自动迁移表结构
func (p *PostgreSQL) AutoMigrate(models ...interface{}) error {
	return p.db.AutoMigrate(models...)
}

// Close 关闭连接
func (p *PostgreSQL) Close() error {
	if p.sqlDB != nil {
		return p.sqlDB.Close()
	}
	return nil
}

// Health 健康检查
func (p *PostgreSQL) Health(ctx context.Context) error {
	return p.sqlDB.PingContext(ctx)
}

// Stats 获取连接池统计信息
func (p *PostgreSQL) Stats() sql.DBStats {
	return p.sqlDB.Stats()
}

// WithContext 使用上下文
func (p *PostgreSQL) WithContext(ctx context.Context) *gorm.DB {
	return p.db.WithContext(ctx)
}

// Begin 开始事务
func (p *PostgreSQL) Begin() *gorm.DB {
	return p.db.Begin()
}

// Rollback 回滚事务
func (p *PostgreSQL) Rollback(tx *gorm.DB) *gorm.DB {
	return tx.Rollback()
}

// Commit 提交事务
func (p *PostgreSQL) Commit(tx *gorm.DB) *gorm.DB {
	return tx.Commit()
}

// createDatabaseIfNotExists 创建数据库（如果不存在）
func createDatabaseIfNotExists(dsn, dbName string) error {
	// 解析DSN，移除数据库名称，连接到postgres默认数据库
	adminDSN := strings.Replace(dsn, "dbname="+dbName, "dbname=postgres", 1)

	// 使用GORM连接到PostgreSQL服务器
	adminDB, err := gorm.Open(postgres.Open(adminDSN), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent), // 静默模式避免过多日志
	})
	if err != nil {
		return fmt.Errorf("failed to connect to PostgreSQL server: %v", err)
	}

	// 获取底层sql.DB对象
	sqlDB, err := adminDB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %v", err)
	}
	defer sqlDB.Close()

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping PostgreSQL server: %v", err)
	}

	// 检查数据库是否存在
	var exists bool
	query := "SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = ?)"
	err = adminDB.Raw(query, dbName).Scan(&exists).Error
	if err != nil {
		return fmt.Errorf("failed to check if database exists: %v", err)
	}

	// 如果数据库不存在，创建它
	if !exists {
		createQuery := fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
		err = adminDB.Exec(createQuery).Error
		if err != nil {
			return fmt.Errorf("failed to create database %s: %v", dbName, err)
		}
		fmt.Printf("Database %s created successfully\n", dbName)
	} else {
		fmt.Printf("Database %s already exists\n", dbName)
	}

	return nil
}
