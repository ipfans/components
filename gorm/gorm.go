package gorm

import (
	"database/sql"
	"time"

	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Handler func(gorm *gorm.DB)
type DBHandler func(db *sql.DB)

type Config struct {
	Opts      []gorm.Option
	Handler   Handler
	DBHandler DBHandler
}

func NewGORM(confs ...Config) func(v *viper.Viper) (*gorm.DB, error) {
	return func(v *viper.Viper) (db *gorm.DB, err error) {
		var c Config
		if len(confs) > 0 {
			c = confs[0]
		}
		if db, err = gorm.Open(mysql.Open(v.GetString("mysql.dsn")), c.Opts...); err != nil {
			return
		}
		if c.Handler != nil {
			c.Handler(db)
		}
		var sqlDB *sql.DB
		if sqlDB, err = db.DB(); err != nil {
			return
		}
		sqlDB.SetConnMaxIdleTime(time.Hour)
		sqlDB.SetConnMaxLifetime(24 * time.Hour)
		sqlDB.SetMaxIdleConns(10)
		sqlDB.SetMaxOpenConns(100)
		if c.DBHandler != nil {
			c.DBHandler(sqlDB)
		}
		return
	}
}
