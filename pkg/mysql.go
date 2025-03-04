package pkg

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func LoadMysql(conf Config) (*gorm.DB, error) {
	db, err := gorm.Open(mysql.Open(conf.MysqlSlave.Database), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}
