package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"AIM/internal/tui"
)

func main() {
	var opts tui.Options
	flag.StringVar(&opts.Server, "server", "", "AIM 服务地址，例如 http://127.0.0.1:8080")
	flag.StringVar(&opts.Username, "username", "", "登录账号")
	flag.StringVar(&opts.Password, "password", "", "登录密码")
	flag.IntVar(&opts.PageSize, "history", 30, "每次加载的历史消息条数")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	app, err := tui.NewApp(os.Stdin, os.Stdout, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := app.Run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
