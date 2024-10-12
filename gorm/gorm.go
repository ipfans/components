package gorm

import (
	"database/sql"
	"time"

	"github.com/ipfans/components/utils"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Config struct {
	DSN             string        `koanf:"dsn"`                // Required, Example: root:123456@tcp(127.0.0.1:3306)/test?charset=utf8&parseTime=True&loc=Local
	ConnMaxIdleTime time.Duration `koanf:"conn_max_idle_time"` // Optional, Default: 1 hour
	ConnMaxLifetime time.Duration `koanf:"conn_max_lifetime"`  // Optional, Default: 24 hours
	MaxIdleConns    int           `koanf:"max_idle_conns"`     // Optional, Default: 10
	MaxOpenConns    int           `koanf:"max_open_conns"`     // Optional, Default: 100
}

func New(conf Config) (db *gorm.DB, err error) {
	if db, err = gorm.Open(mysql.Open(conf.DSN)); err != nil {
		return
	}
	var sqlDB *sql.DB
	sqlDB, err = db.DB()
	if err != nil {
		return
	}
	sqlDB.SetConnMaxIdleTime(utils.DefaultValue(conf.ConnMaxIdleTime, time.Hour))
	sqlDB.SetConnMaxLifetime(utils.DefaultValue(conf.ConnMaxLifetime, 24*time.Hour))
	sqlDB.SetMaxIdleConns(utils.DefaultValue(conf.MaxIdleConns, 10))
	sqlDB.SetMaxOpenConns(utils.DefaultValue(conf.MaxOpenConns, 100))
	return
}
