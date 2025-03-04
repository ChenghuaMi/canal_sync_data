package pkg

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server      `mapstructure:"server"`
	MysqlMaster `mapstructure:"mysql_master"` //canal client connect
	MysqlSlave  `mapstructure:"mysql_slave"`  //slave
}
type MysqlMaster struct {
	Host        string `mapstructure:"host"`        // master host
	Port        int    `mapstructure:"port"`        //master port
	MasterUser  string `mapstructure:"master_user"` //master user
	MasterPass  string `mapstructure:"master_pass"`
	Destination string `mapstructure:"destination"` //目的地
	SoTimeOut   int    `mapstructure:"so_timeout"`
	IdleTimeOut int    `mapstructure:"idle_timeout"`
}

type MysqlSlave struct {
	Database string `mapstructure:"database"` //slave database
}
type Server struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

func Viper() Config {
	v := viper.New()
	filePath := "./etc/"

	v.SetConfigName("config") // name of config file (without extension)
	v.SetConfigType("yaml")   // REQUIRED if the config file does not have the extension in the name
	v.AddConfigPath(filePath)

	// Enable reading from environment variables
	v.AutomaticEnv()
	// v.SetEnvPrefix("ORCA") // set environment variable prefix
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	var cfg Config
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("read yaml config error %v\n", err)
	}
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("viper parse yaml file error: %v\n", err)
	}
	return cfg
}
