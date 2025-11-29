package config

import (
	"payment-service/src/pkg/databases/mysql"
	"payment-service/src/pkg/log"

	"github.com/spf13/viper"
)

func NewDatabase(viper *viper.Viper, log log.Log) mysql.DBInterface {
	db, err := mysql.InitConnection(viper, log)
	if err != nil {
		log.Error("database init", err.Error(), "config", "")

	}

	return db
}
