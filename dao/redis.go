package dao

import (
	"IM_Server/pkg"
	"context"

	"github.com/redis/go-redis/v9"
)

var RDB *redis.Client

// Ctx Redis 操作用的背景 context（HTTP 场景可换成请求级 ctx 以便超时/取消传递）
var Ctx = context.Background()

func InitRedis() {
	RDB = redis.NewClient(&redis.Options{
		Addr:     pkg.Cfg.Redis.Addr,
		Password: pkg.Cfg.Redis.Password,
		DB:       pkg.Cfg.Redis.DB,
	})
	if err := RDB.Ping(Ctx).Err(); err != nil {
		pkg.Log.Fatal("connect redis: ", err)
	}
	pkg.Log.Info("redis connected: ", pkg.Cfg.Redis.Addr)
}
