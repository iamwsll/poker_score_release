package models

import (
	"log"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase 初始化数据库连接
func InitDatabase(dbPath string, maxIdleConns, maxOpenConns int, connMaxLifetime time.Duration) error {
	var err error
	
	// 打开数据库连接
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info), // 设置日志级别
	})
	if err != nil {
		return err
	}

	// 获取底层的sql.DB以配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(maxIdleConns)
	sqlDB.SetMaxOpenConns(maxOpenConns)
	sqlDB.SetConnMaxLifetime(connMaxLifetime)

	log.Println("数据库连接成功")

	// 自动迁移数据库表
	err = autoMigrate()
	if err != nil {
		return err
	}

	log.Println("数据库表迁移成功")

	// 初始化默认管理员账户
	err = initAdminUser()
	if err != nil {
		return err
	}

	log.Println("数据库初始化完成")
	return nil
}

// autoMigrate 自动迁移所有表
func autoMigrate() error {
	return DB.AutoMigrate(
		&User{},
		&Session{},
		&Room{},
		&RoomMember{},
		&UserBalance{},
		&RoomOperation{},
		&Settlement{},
		&BetRecord{},
	)
}

// initAdminUser 初始化默认管理员账户
func initAdminUser() error {
	// 检查管理员账户是否已存在
	var count int64
	err := DB.Model(&User{}).Where("role = ?", "admin").Count(&count).Error
	if err != nil {
		return err
	}

	// 如果已存在管理员，则不创建
	if count > 0 {
		log.Println("管理员账户已存在，跳过创建")
		return nil
	}

	// 导入密码加密工具
	// 这里直接使用bcrypt加密，避免循环依赖
	passwordHash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy" // admin123
	
	// 创建默认管理员账户
	admin := User{
		Phone:        "13800138000",
		Nickname:     "系统管理员",
		PasswordHash: passwordHash,
		Role:         "admin",
	}

	err = DB.Create(&admin).Error
	if err != nil {
		return err
	}

	log.Println("默认管理员账户创建成功: 手机号 13800138000, 密码 admin123")
	return nil
}

// CloseDatabase 关闭数据库连接
func CloseDatabase() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

