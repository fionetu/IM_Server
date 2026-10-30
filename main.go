package main

import (
	"context"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"IM_Server/dao"
	"IM_Server/handler"
	"IM_Server/middleware"
	"IM_Server/pkg"
	"IM_Server/ws"

	"github.com/gin-gonic/gin"
)

func main() {
	// 启动顺序有依赖：配置 → 日志 → JWT → MySQL/Redis
	pkg.LoadConfig()
	pkg.InitLogger()
	pkg.InitJWT(pkg.Cfg.JWT.Secret, pkg.Cfg.JWT.ExpireHours)
	dao.InitDB()
	dao.InitRedis()

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "pong"})
	})
	r.POST("/user/register", handler.Register)
	r.POST("/user/login", handler.Login)
	r.GET("/ws", ws.Connect)

	// 需要登录态的 HTTP 接口
	auth := r.Group("", middleware.JWTAuth())
	auth.GET("/message/history", handler.MessageHistory)
	auth.GET("/message/group/history", handler.GroupMessageHistory)
	auth.POST("/message/read", handler.MarkRead)

	auth.POST("/friend/apply", handler.ApplyFriend)
	auth.POST("/friend/handle", handler.HandleFriend)
	auth.GET("/friend/list", handler.FriendList)
	auth.GET("/friend/requests", handler.FriendRequests)
	auth.GET("/friend/online", handler.OnlineFriends)

	auth.POST("/group/create", handler.CreateGroup)
	auth.POST("/group/add", handler.AddGroupMembers)
	auth.GET("/group/list", handler.GroupList)
	auth.GET("/group/members", handler.GroupMembers)

	ws.StartHub()

	// 优雅退出：收到 Ctrl+C / kill 后停止接新连接，
	// 给进行中的请求 5 秒收尾，再退出进程
	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", pkg.Cfg.Server.Port),
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			pkg.Log.Fatal("listen: ", err)
		}
	}()
	pkg.Log.Infof("server listening on :%d", pkg.Cfg.Server.Port)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		pkg.Log.Error("shutdown: ", err)
	}
	pkg.Log.Info("server stopped")
}
