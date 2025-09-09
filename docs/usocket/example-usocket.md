## UDS Listener Example (Echo)

Below is a sample code to show how the `UDS` listener library can be used.

```golang
//go:build unix

package main

import (
    "bufio"
    "errors"
    "flag"
    "io"
    "log"
    "log/slog"
    "net"
    "os"
    "time"

    "github.com/handletec/listener/usocket"
)

func main() {
    socketPath := flag.String("sock", "/tmp/demo.sock", "unix socket path")
    flag.Parse()

    // 1) Create listener
    l := usocket.New()

    // 2) Build config with sane defaults + our handler
    cfg := usocket.NewConfig(func(conn net.Conn) error {
        br := bufio.NewReader(conn)
        bw := bufio.NewWriter(conn)

        // Optional greeting so clients immediately see output
        if _, err := bw.WriteString("READY\n"); err != nil { return err }
        if err := bw.Flush(); err != nil { return err }

        // Echo loop: read a line, echo it back; exit on client close
        for {
            line, err := br.ReadString('\n')
            if err != nil {
                if errors.Is(err, io.EOF) { return nil }
                return err
            }
            if _, err := bw.WriteString("ECHO: " + line); err != nil {
                return err
            }
            if err := bw.Flush(); err != nil {
                return err
            }
        }
    })

    // 3) Optional tuning
    cfg.Perm         = 0o660
    cfg.MaxConns     = 64
    cfg.ReadTimeout  = 30 * time.Second
    cfg.WriteTimeout = 30 * time.Second
    // cfg.AllowUIDs = []uint32{uint32(os.Getuid())} // restrict to current user

    // 4) Apply config
    if err := l.SetConfig(cfg); err != nil {
        log.Fatal(err)
    }

    // 5) Init + Start
    logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
    if err := l.Init(logger, *socketPath, 0, nil); err != nil {
        log.Fatal(err)
    }
    if err := l.Start(); err != nil {
        log.Fatal(err)
    }
    defer l.Close()

    // Keep running
    select {}
}
```
