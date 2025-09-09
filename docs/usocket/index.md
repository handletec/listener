## UDS Listener

Create a listener for **UNIX domain sockets (UDS)**. This library reduces repetitive boilerplate when building socket-based services and adds safe defaults (permissions, stale-socket handling, optional peer-credential checks).  
It relies on Go's standard `net` package.

Refer to the [UDS example](example-usocket.md) to see how to use it.  
The example `go` source code can be found at [echo](../../examples/usocket/echo/main.go).

> **Build note:** This listener targets Unix-like OSes. On Windows builds, `Start()` returns a descriptive error.

---

### Workflow

The process for instantiating a `UDS` listener is straightforward:

1. Create a new UDS listener.
2. Write a handler function (`func(net.Conn) error`) that processes each accepted connection.
3. Create a configuration instance (with sane defaults).
4. (Optional) Tweak permissions, timeouts, peer-credential allowlists, and concurrency limits.
5. Initialize the `UDS` listener and start it.

---

### New Listener <a name="usocket-listener"></a>

Initialize a new listener:

```go
udsListener := usocket.New()
```

---

### Handler <a name="usocket-handler"></a>

Handlers are per-connection functions. They run in their own goroutine and should read/write on the provided `net.Conn`.

```go
handler := func(conn net.Conn) error {
	br := bufio.NewReader(conn)
	bw := bufio.NewWriter(conn)

	// simple echo
	line, err := br.ReadString('\n')
	if err != nil { return err }
	_, _ = bw.WriteString("ECHO: " + line)
	return bw.Flush()
}
```

---

### Config <a name="usocket-config"></a>

`Config` lets you tune the listener: file permissions, timeouts, peer-credential allowlists, concurrency caps, and more.

Start with safe defaults, then override as needed:

```go
cfg := usocket.NewConfig(handler) // sets sane defaults, attaches handler

// customize values (leave unset to use defaults)
cfg.Perm         = 0o660                // rw for owner+group
cfg.MaxConns     = 64                   // cap concurrent handlers
cfg.ReadTimeout  = 30 * time.Second     // per-conn deadlines
cfg.WriteTimeout = 30 * time.Second

// Optional: restrict to current user (Linux/macOS peer credentials)
cfg.AllowUIDs = []uint32{uint32(os.Getuid())}
// or restrict to a group:
// cfg.AllowGIDs = []uint32{uint32(os.Getgid())}
```

**Defaults applied by `NewConfig`:**

- `Perm: 0660` (owner/group read+write)
- `OwnerUID, GroupGID: -1` (no chown)
- `RemoveStale: true` (safe stale-socket removal)
- `AcceptBackoff: 100ms`
- `MaxConns: 0` (unlimited)
- `ReadTimeout/WriteTimeout: 0` (no deadlines)
- `AllowUIDs/AllowGIDs: nil` (no peer-cred restriction)
- `Handler: (your func)` (required)

---

### Init and start the UDS listener

```go
err = udsListener.SetConfig(cfg)
if err != nil {
	log.Println(err)
	os.Exit(1)
}

// use slog for logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// bind to a socket path (port ignored)
err = udsListener.Init(logger, "/tmp/app.sock", 0, nil)
if err != nil {
	log.Println(err)
	os.Exit(1)
}

err = udsListener.Start()
if err != nil {
	log.Println(err)
	os.Exit(1)
}
```

---

### Testing

You can use `ncat` or `socat`:

```bash
# send one line and read response
printf 'hello\n' | ncat -U /tmp/app.sock

# interactive
ncat -U /tmp/app.sock
```
If your handler speaks JSON (NDJSON), send JSON lines instead:

```bash
printf '{"op":"ping"}\n' | ncat -U /tmp/app.sock
```
