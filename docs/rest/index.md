## REST Listener

Create a listener for REST operations. This library is intended to reduce repetitive boilerplate when building REST services.  
It supports creating both **HTTP** and **HTTPS** listeners.

It uses [chi](https://github.com/go-chi/chi) as the underlying router.

Refer to the [HTTP example](example-http.md) or [HTTPS example](example-https.md) documentation to see how to use it.  
The example `go` source code can be found at [http](../../examples/rest/http/main.go) or [https](../../examples/rest/https/main.go).

To add TLS support to the application, follow the steps in the [TLS](#rest-tls) section.

---

### Workflow

The process for instantiating a `REST` listener is straightforward:

1. Create a new REST listener.
2. Create a new router.
3. Set handlers for given methods, paths, functions, and optional middlewares.
   - Middlewares are optional for routers, groups, and handlers.
   - Only **handler-level** middlewares can access named URL parameters (because router/group-level middlewares run before params are parsed).
4. (Optional) Create groups for related endpoints and add them to the router.
5. Create a configuration instance.
6. Set configuration options and the router on the listener.
7. Initialize the `REST` listener and start it.

---

### Methods

Supported HTTP methods are:

| Method Name | Constant |
|-------------|----------|
| GET         | `MethodGet` |
| POST        | `MethodPost` |
| PUT         | `MethodPut` |
| DELETE      | `MethodDelete` |
| HEAD        | `MethodHead` |
| OPTIONS     | `MethodOptions` |
| CONNECT     | `MethodConnect` |
| PATCH       | `MethodPatch` |

---

### New Listener <a name="rest-listener"></a>

Initialize a new listener:

```go
restListener := rest.New()
```

---

### Router <a name="rest-router"></a>

Create a new router under a base path:

```go
restRouter := rest.NewRouter("/api", routerMiddleWare)

// middlewares must have the following structure
func routerMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log.Println("calling router middleware")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

Replace the `/api` base path with one that matches your application preference.  
All REST requests registered on this router will be nested under the base path.

---

### Handlers <a name="rest-handler"></a>

#### Creating and setting handlers

With a [Router](#rest-router) defined, you can add handlers as needed.  
Handlers are functions that run per request, bound to an HTTP method and endpoint. Different methods can be bound to the same path.

```go
restHandler := rest.NewNewHandler() // instantiate a new handler

// set the method, endpoint and function for handling requests
_ = restHandler.Set(rest.MethodGet, "/server/list", serverList, serverListMiddleWare)
_ = restHandler.Set(rest.MethodGet, "/server/{type}/{id}", serverID, serverIDMiddleWare)

restRouter.SetHandler(restHandler) // mount handler under the base path

func serverList(w http.ResponseWriter, r *http.Request) {
	log.Println("servers list called")
}

func serverID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := ctx.Value(ctxInt)
	srvType := ctx.Value(ctxStr)
	log.Println("server id called with id", id, "for server type", srvType)
}

// handler-level middleware example
func serverListMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log.Println("calling server list middleware")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

> **Note:** Only **handler-level** middlewares can access URL parameters (`chi.URLParam`). Router- and group-level middlewares run before URL params are parsed.

---

### Defining groups

You can group related endpoints for clarity and apply middlewares to the entire group.

```go
grpUser := rest.NewGroup("/user", userGroupMiddleWare)
_ = grpUser.Set(rest.MethodGet, "/list", userList, userListMiddleWare)
_ = grpUser.Set(rest.MethodGet, "/{id}", userID, userIDMiddleWare)

restRouter.AddGoup(grpUser)

func userList(w http.ResponseWriter, r *http.Request) {
	log.Println("user list called")
}

func userID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := ctx.Value(ctxInt)
	log.Println("user id called with id", id)
}

// middleware example
func userGroupMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		log.Println("calling user group middleware")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
```

---

### Config <a name="rest-config"></a>

`Config` lets you tune the listener: max requests per second, timeouts, CORS, compression, router, etc.

The default config has safe defaults, but you can override them:

```go
restConfig := rest.NewConfig()

// customize values (leave unset to use defaults)
restConfig.RPS     = 4096
restConfig.Timeout = 15 * time.Second

// configure CORS for your app
restConfig.CORS.SetOrigins([]string{"https://*", "http://*"})
restConfig.CORS.SetHeaders([]string{"Authorization", "Content-Type", "Access-Control-Request-Method", "Access-Control-Request-Headers"})
restConfig.CORS.SetMethods([]string{"OPTIONS", "GET", "POST", "PUT", "DELETE", "HEAD", "CONNECT", "PATCH"})
// Optional extras:
// restConfig.CORS.MaxAge = 600
// restConfig.CORS.AllowCredentials = true
// restConfig.CORS.Debug = true

restConfig.EnableCompress(true) // enable output compression

restConfig.SetRouter(restRouter) // attach configured routes
```

---

### Init and start the REST listener

```go
err = restListener.SetConfig(restConfig)
if err != nil {
	log.Println(err)
	os.Exit(1)
}

// use slog for logging
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// listen on all interfaces (IPv4/IPv6) on port 8081 with no TLS configuration
err = restListener.Init(logger, "[::]", 8081, nil)
if err != nil {
	log.Println(err)
	os.Exit(1)
}

err = restListener.Start()
if err != nil {
	log.Println(err)
	os.Exit(1)
}
```

---

### TLS support <a name="rest-tls"></a>

If you need HTTPS, you can build a TLS config. Certificates/keys can be reloaded without restarting. You can also add a custom CA.

```go
// Initialize a new TLS config builder with OS CA pool.
listenerTLS, err := listener.NewTLSConfigBuilder(true)
if err != nil {
	return fmt.Errorf("error initializing TLS builder -> %w", err)
}

// Set server certificate + key (these files can be rotated without restart).
err = listenerTLS.SetCertKeyFile("/path/to/app.crt", "/path/to/app.key")
if err != nil {
	return fmt.Errorf("TLS cert/key error -> %w", err)
}

// Optionally configure client authentication policy.
switch clientAuthTypeStr {
case "none":
	listenerTLS.SetClientAuthType(listener.TLSClientAuthNone)
case "request":
	listenerTLS.SetClientAuthType(listener.TLSClientAuthRequest)
case "require":
	listenerTLS.SetClientAuthType(listener.TLSClientAuthRequire)
case "verify":
	listenerTLS.SetClientAuthType(listener.TLSClientAuthVerify)
case "requireverify":
	listenerTLS.SetClientAuthType(listener.TLSClientAuthRequireVerify)
default:
	return fmt.Errorf("unsupported TLS_CLIENT_AUTH_TYPE %s", clientAuthTypeStr)
}

// Optionally add custom CA certificates if not in system trust store.
err = listenerTLS.AddCADir("/path/to/ca/directory")
if err != nil {
	return fmt.Errorf("client CA directory error -> %w", err)
}

err = listenerTLS.AddCAFile("/path/to/ca/ca.pem")
if err != nil {
	return fmt.Errorf("client CA file error -> %w", err)
}

// Pass TLS config into Init
tlsCfg, err := listenerTLS.ForServer()
if err != nil {
	return fmt.Errorf("TLS server config error -> %w", err)
}
restListener.Init(logger, "[::]", 8443, tlsCfg)
```