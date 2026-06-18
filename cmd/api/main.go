package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "storeready_ai/gen/swagger" // swag init 生成的 docs
	"storeready_ai/internal/app"
	infrolog "storeready_ai/internal/infra/log"
)

func main() {

	a, err := app.NewWithPath("")

	if err != nil {
		infrolog.L().Fatal("应用初始化失败", infrolog.Any("err", err))
	}

	// 后台启动服务（主协程等待信号）
	go func() {
		if err := a.Start(); err != nil {
			infrolog.L().Fatal("应用启动失败", infrolog.Any("err", err))
		}
	}()

	infrolog.L().Info("应用已启动", infrolog.String("listen", a.Cfg.Server.Listen))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	infrolog.L().Info("收到退出信号")

	// 优雅退出：给 Stop 一个固定超时时间（可按需改为读取 cfg.Server.GracefulTimeout）
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	if err := a.Stop(ctx); err != nil {
		infrolog.L().Error("应用停止失败", infrolog.Any("err", err))
	}

	infrolog.L().Info("退出完成")
}
