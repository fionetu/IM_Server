package pkg

import "go.uber.org/zap"

// Log 全局结构化日志，main 里 InitLogger 之后可用
var Log *zap.SugaredLogger

func InitLogger() {
	// 开发期用人类可读的 console 输出；上线换 NewProduction（JSON）
	l, err := zap.NewDevelopment()
	if err != nil {
		panic("init logger: " + err.Error())
	}
	Log = l.Sugar()
}
