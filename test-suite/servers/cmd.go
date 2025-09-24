package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/zishang520/socket.io/servers/socket/v3"
	"github.com/zishang520/socket.io/v3/pkg/log"
	"github.com/zishang520/socket.io/v3/pkg/types"
)

func Socket(addr string) *socket.Server {
	config := socket.DefaultServerOptions()
	config.SetPingInterval(300 * time.Millisecond)
	config.SetPingTimeout(200 * time.Millisecond)
	config.SetMaxHttpBufferSize(1000000)
	config.SetConnectTimeout(1000 * time.Millisecond)
	config.SetCors(&types.Cors{
		Origin:      "*",
		Credentials: true,
	})

	httpServer := types.NewWebServer(nil)
	io := socket.NewServer(httpServer, config)

	httpServer.Listen(addr, nil)

	return io
}

func main() {
	log.DEBUG = true

	io := Socket(":3000")

	io.On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)

		defer client.Emit("auth", client.Handshake().Auth)

		client.On("message", func(args ...any) {
			client.Emit("message-back", args...)
		})

		client.On("message-with-ack", func(args ...any) {
			if len(args) > 0 {
				if ack, ok := args[len(args)-1].(socket.Ack); ok {
					ack(args[:len(args)-1], nil)
				}
			}
		})
	})

	io.Of("/custom", nil).On("connection", func(clients ...any) {
		client := clients[0].(*socket.Socket)
		defer client.Emit("auth", client.Handshake().Auth)
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	defer stop()
	<-ctx.Done()
	io.Close(nil)
	os.Exit(0)
}
