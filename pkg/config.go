package pkg

import (
	"github.com/spf13/viper"
)

// Config 全局配置，启动时从 config.yaml 加载
type Config struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`
	MySQL struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"mysql"`
	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`
	JWT struct {
		Secret      string `mapstructure:"secret"`
		ExpireHours int    `mapstructure:"expire_hours"`
	} `mapstructure:"jwt"`
}

var Cfg Config

// LoadConfig 读取 config.yaml 到 Cfg，失败直接退出（配置错了没有降级空间）
func LoadConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		panic("load config: " + err.Error())
	}
	if err := viper.Unmarshal(&Cfg); err != nil {
		panic("parse config: " + err.Error())
	}
}
