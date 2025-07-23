package database

import (
	"context"
	"database/sql"
	"fmt"
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
