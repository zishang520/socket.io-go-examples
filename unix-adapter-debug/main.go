// Package main demonstrates publishing Socket.IO cluster messages over Unix
// domain sockets and decoding the complete frames on another process client.
package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/zishang520/socket.io/adapters/adapter/v3"
	"github.com/zishang520/socket.io/adapters/unix/v3"
	unixemitter "github.com/zishang520/socket.io/adapters/unix/v3/emitter"
)

const exampleTimeout = 5 * time.Second

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "unix adapter example:", err)
		os.Exit(1)
	}
}

func run() error {
	directory, err := os.MkdirTemp("", "socket.io-unix-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(directory)

	ctx, cancel := context.WithTimeout(context.Background(), exampleTimeout)
	defer cancel()

	basePath := filepath.Join(directory, "socket.io")
	receiver, err := unix.NewUnixClient(ctx, basePath)
	if err != nil {
		return err
	}
	defer receiver.Close()

	publisher, err := unix.NewUnixClient(ctx, basePath)
	if err != nil {
		return err
	}
	defer publisher.Close()

	if err := receiver.Listen(basePath + ".receiver"); err != nil {
		return err
	}

	emitter := unixemitter.NewEmitter(publisher)
	if err := emitter.Emit("greeting", "hello over Unix sockets"); err != nil {
		return err
	}
	if err := receiveEvent(receiver, "greeting", "JSON", "hello over Unix sockets", nil); err != nil {
		return err
	}

	binary := []byte{0x00, 0x01, 0x7f, 0xff}
	if err := emitter.Emit("binary", binary); err != nil {
		return err
	}
	return receiveEvent(receiver, "binary", "MessagePack", "", binary)
}

func receiveEvent(
	receiver *unix.UnixClient,
	wantEvent string,
	wantEncoding string,
	wantText string,
	wantBinary []byte,
) error {
	payload, _, err := receiver.ReadMessage()
	if err != nil {
		return err
	}

	message, err := adapter.DecodeClusterMessage(payload)
	if err != nil {
		return fmt.Errorf("decode %s event: %w", wantEvent, err)
	}
	if message.Type != adapter.BROADCAST {
		return fmt.Errorf("%s event: unexpected message type %d", wantEvent, message.Type)
	}
	broadcast, ok := message.Data.(*adapter.BroadcastMessage)
	if !ok || broadcast.Packet == nil {
		return fmt.Errorf("%s event: missing broadcast packet", wantEvent)
	}
	args, ok := broadcast.Packet.Data.([]any)
	if !ok || len(args) != 2 || args[0] != wantEvent {
		return fmt.Errorf("%s event: unexpected packet data %#v", wantEvent, broadcast.Packet.Data)
	}

	encoding := "MessagePack"
	if len(payload) > 0 && payload[0] == '{' {
		encoding = "JSON"
	}
	if encoding != wantEncoding {
		return fmt.Errorf("%s event: got %s encoding, want %s", wantEvent, encoding, wantEncoding)
	}

	if wantBinary != nil {
		got, ok := args[1].([]byte)
		if !ok || !bytes.Equal(got, wantBinary) {
			return fmt.Errorf("%s event: unexpected binary payload %#v", wantEvent, args[1])
		}
	} else if args[1] != wantText {
		return fmt.Errorf("%s event: unexpected text payload %#v", wantEvent, args[1])
	}

	fmt.Printf("received %-8s event via %-11s namespace=%q payload=%#v\n",
		wantEvent, encoding, message.Nsp, args[1])
	return nil
}
