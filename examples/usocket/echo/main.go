//go:build unix

package main

import (
	"bufio"
	"errors"
	"flag"
	"io"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/handletec/listener/usocket"
)

func main() {
	sock := flag.String("sock", "/tmp/demo.sock", "unix socket path")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	l := usocket.New()
	if err := l.Init(logger, *sock, 0, nil); err != nil {
		panic(err)
	}

	cfg := usocket.NewConfig(func(conn net.Conn) error {
		connLog := logger.With(
			"component", "usocket-handler",
			"local", conn.LocalAddr().String(),
			"remote", conn.RemoteAddr().String(), // often empty for UDS
		)
		start := time.Now()
		connLog.Info("accepted")

		br := bufio.NewReader(conn)
		bw := bufio.NewWriter(conn)

		if _, err := bw.WriteString("READY\n"); err != nil {
			connLog.Error("write greeting failed", "err", err)
			return err
		}
		if err := bw.Flush(); err != nil {
			connLog.Error("flush greeting failed", "err", err)
			return err
		}
		connLog.Debug("greeted client")

		for {
			line, err := br.ReadString('\n')
			if err != nil {
				if errors.Is(err, io.EOF) {
					// If client closed without newline, log any partial bytes we got
					if len(line) > 0 {
						connLog.Info("recv (partial before EOF)", "line", strings.TrimRight(line, "\r\n"))
						_, _ = bw.WriteString("ECHO: " + strings.TrimRight(line, "\r\n") + "\n")
						_ = bw.Flush()
					}
					connLog.Info("closed", "dur", time.Since(start))
					return nil
				}
				connLog.Error("read failed", "err", err)
				return err
			}

			trimmed := strings.TrimRight(line, "\r\n")
			connLog.Info("recv", "line", trimmed)

			if _, err := bw.WriteString("ECHO: " + trimmed + "\n"); err != nil {
				connLog.Error("write failed", "err", err)
				return err
			}
			if err := bw.Flush(); err != nil {
				connLog.Error("flush failed", "err", err)
				return err
			}
			connLog.Debug("sent echo")
		}
	})

	// Helpful timeouts for interactive tests
	cfg.ReadTimeout = 30 * time.Second
	cfg.WriteTimeout = 30 * time.Second

	if err := l.SetConfig(cfg); err != nil {
		panic(err)
	}
	if err := l.Start(); err != nil {
		panic(err)
	}
	defer l.Close() // graceful cleanup (also removes the socket file)

	// Graceful shutdown on Ctrl+C / SIGTERM
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Info("server stopped")
}
