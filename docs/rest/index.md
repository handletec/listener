## REST Listener

Create a listener for REST operations, this library is intended to reduce repetitve writing of code for REST. IT supports creating both HTTP and HTTPS listeners.

It uses `chi` as the underlying library to set up the listener and its related handlers.

Refer to the [example-http](example-http.md) or [example-https](example-https.md) documentation to view examples on how to use it. The example `go` source code can be found at [http](../../examples/rest/http/main.go) or [https](../../examples/rest/https/main.go)

To add TLS support to the application, do the following as usual, then refer to the [TLS](#rest-tls) section to add the TLS support.


#### Workflow

The process for instantiating a `REST` listener is relatively straightforward.

1. Create a new REST listener.
2. Create a new router.
3. Set handlers for given methods, path, function to handle the request and optional middlewares.
	- Middlewares are purely optional for routers, groups and handlers. They are great for performing tasks that are common for specific patterns, but can be left blank if it is not needed.
	- middlewares that call named parameters from the url won't work in router and group level because the named parameter doesn't exist in the context where it was called. This means only handler level middlewares are able to call named parameters.
4. (Optional) set handlers as above but for a group.
5. Create a configuration instance.
6. Set the necessary config and the configured routers.
7. Initalize the `REST` listener and start it.


#### Methods
Supported methods for the listener are 

| Method Name | Method Var | 
| :--: | :--: | 
| GET | `MethodGet` |
| POST | `MethodPost` |
| PUT | `MethodPut` |
| DELETE | `MethodDelete` |
| HEAD | `MethodHead` |
| OPTIONS | `MethodOptions` |
| CONNECT | `MethodConnect` |

#### New Listener <a name="rest-listener"></a>

Initalize a new listener as follows

```golang
restListener := rest.New()
```


#### Router <a name="rest-router"></a>

Create a new instance of router as follows

```golang
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

Replace the `/api` base path with one that matches your application preference. This makes all HTTP REST requests available under the base path.


#### Handlers <a name="rest-handler"></a>

##### Creating and setting handlers

With a [Router](#rest-handler) defined, we can add handlers to it as needed. Handlers are functions that get run based on a given method and endpoint configuration. Different methods can be used for the same endpoint if needed. Once instantiated, the handler can be configured with the methods and endpoints.

A handler needs to be instantiated before being allowed to be used. Middlewares can be defined for each handler individually, allowing checks such as authorization, parameter checks, etc to be done before the actual function is called.

Once all handlers have been defined, we proceed to mount them under the base path. A little note, if using groups **ONLY**, there is no need to call this mount function as the groups are already mounted when calling the `AddGroup` function.

```golang
restHandler := rest.NewNewHandler() // instantiate a new handler

// set the method, endpoint and function for handling requests
restHandler.Set(rest.MethodGet, "/server/list", serverList, serverListMiddleWare)
restHandler.Set(rest.MethodGet, "/server/{type}/{id}", serverID, serverIDMiddleWare)

restRouter.SetHandler(restHandler) // set the remaining handlers configured for the router


func serverList(w http.ResponseWriter, r *http.Request) {
	log.Println("servers list called")
}

func serverID(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	id := ctx.Value(ctxInt)
	srvType := ctx.Value(ctxStr)
	log.Println("server id called with id", id, "for server type", srvType)
}

// middlewares must have the following structure
func serverListMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		log.Println("calling server list middleware")

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func serverIDMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		id := chi.URLParam(r, "id")
		srvType := chi.URLParam(r, "type")

		ctx = context.WithValue(ctx, ctxStr, srvType)
		ctx = context.WithValue(ctx, ctxInt, id)

		log.Println("calling server id middleware with id", id, "for server type", srvType)

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
```

##### (Optional) Defining groups

An alternative to writing handlers is grouping them together, where related tasks can be grouped together, making it easier to set middlewares and such. Writing a group requires instantiaing a group before defining the handlers, before adding the group to the previously defined router.

```golang
grpUser := rest.NewGroup("/user")
grpUser.Set(rest.MethodGet, "/list", userList, userListMiddleWare)
grpUser.Set(rest.MethodGet, "/{id}", userID, userIDMiddleWare)

restRouter.AddGoup(grpUser)

func userList(w http.ResponseWriter, r *http.Request) {
	log.Println("user list called")
}

func userID(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id := ctx.Value(ctxInt)
	log.Println("user id called with id", id)
}

// middlewares must have the following structure
func userListMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		log.Println("calling user list middleware")

		next.ServeHTTP(w, r.WithContext(ctx))

	})
}

func userIDMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		id := chi.URLParam(r, "id")

		ctx = context.WithValue(ctx, ctxInt, id)

		log.Println("calling user id middleware with id", id)
		next.ServeHTTP(w, r.WithContext(ctx))

	})
}
```


#### Config <a name="rest-config"></a>

Config allows users to define basic rules for REST, such as maximum requests per second, timeout for requests, CORS and [Router](#rest-router).

The default config provides safe defaults to be used, however it can be customized to the users liking as needed. The only compulsory field to be given is the handler, which is unique to each application.

Instantiate a new config and set the parameters to your liking.

```golang
restConfig := rest.NewConfig()

// customize values (leave this unset to use the default values)
restConfig.RPS = 4096 
restConfig.Timeout = time.Duration(15 * time.Second)

// for CORS, customize the values to match the applications needs, typically setting the origin, headers and methods should be sufficient
restConfig.CORS.SetOrigins([]string{"https://*", "http://*"}) // set allowed origins for the application.
restConfig.CORS.SetHeaders([]string{"Authorization", "Content-type", "Access-Control-Request-Method", "Access-Control-Request-Headers"}) // restrict the headers allowed by the application
restConfig.CORS.SetMethods([]string{"OPTIONS", "GET", "POST", "PUT", "DELETE", "HEAD", "CONNECT"}) // restrict supported methods by the application

// the following have default values, so they can be ignored
restConfig.CORS.MaxAge = 600            // number of seconds this result can be cached for (defualt 0)
restConfig.CORS.AllowCredentials = true // allow response to cross-origin requests (those that don't match the Origins)
restConfig.CORS.Debug = true            // print the CORS detail for application development purposes
// end CORS

restConfig.EnableCompress(true) // enable output compression, useful for large amounts of data

// end customize values

restConfig.SetRouter(restRouter) // sets the configured routes for this application

```


#### Init and start the `REST` listener

Once we have done the needful, we can initialize this listener and start it for our application

```golang
err = restListener.SetConfig(restConfig)
if nil != err {
	log.Println(err)
	os.Exit(1)
}

// the library uses the new `slog` library for logging, which makes it easier to write custom logs
logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

// listen on all interfaces (ipv4 and ipv6) on port 8081 with no TLS configuration
restListener.Init(logger, "[::]", 8081, nil)
err = restListener.Start()
if nil != err {
	log.Println(err)
	os.Exit(1)
}
```

#### TLS support <a name="rest-tls"></a>

If the application you are writing is going to need TLS support, with or without a reverse proxy, this library supports adding TLS to the application with custom CA and dynamic loading of certicates and keys.

Dynamic loading of certificates and keys allows both to be rotated or replaced without needing to restart the application.

In some cases, the application you're writing will be specific to an organization, which may have its own Certificate Authority (CA), whereby this library provides support to add the root CA to the application.

```golang
listenerTLS, err = listener.NewTLS(true) // initalize a new TLS instance and read the default OS defined CAs
if nil != err {
	return fmt.Errorf("error initializing listener TLS -> %w", err)
}

err = listenerTLS.SetCertFile("/path/to/app.crt") // set the path to the certificate
if nil != err {
	return fmt.Errorf("TLS certificate file error -> %w", err)
}

err = listenerTLS.SetKeyFile("/path/to/app.key") // set the path to the key
if nil != err {
	return fmt.Errorf("TLS private key file error -> %w", err)

}
```

##### Client TLS authentication

This library also provides support for defining client level TLS authentication. To use this feature, initalize the [TLS](#rest-tls) as above then do the following

```golang

switch clientAuthTypeStr {
case "none":
	// there is no need to verify clients
	listenerTLS.SetClientAuthType(listener.TLSClientAuthNone)
case "request":
	// server may request client cert but clients are not obligated to send it
	listenerTLS.SetClientAuthType(listener.TLSClientAuthRequest)
case "require":
	// clients should send a certificate however the cert does not need to be valid
	listenerTLS.SetClientAuthType(listener.TLSClientAuthRequire)
case "verify":
	// server may request client cert and if client responds, the cert must be valid
	listenerTLS.SetClientAuthType(listener.TLSClientAuthVerify)
case "requireverify":
	// server requests client cert and the client **MUST** respond with a valid certificate
	listenerTLS.SetClientAuthType(listener.TLSClientAuthRequireVerify)
default:
	return fmt.Errorf("unsupported TLS_CLIENT_AUTH_TYPE %s, acccepted values are 'none','request','require','verify','requireverify'", clientAuthTypeStr)
}

// the following lines is only needed if a custom CA is used that is not part of the OS CA list.

// add path to folder container custom CA certificates, use either this or specific custom CA file
err = listenerTLS.AddCADir("/path/to/ca/directory")
if nil != err {
	return fmt.Errorf("client CA directory error -> %w", err)
}

// add path to file container custom CA certificates, use either this or specific custom CA folder
err = listenerTLS.AddCAFile("/path/to/ca/ca.pem")
if nil != err {
	return fmt.Errorf("client CA file error -> %w", err)
}

// get `tls.Config` instance to be passed to the REST listener initialization
tlsConfig := listenerTLS.GetTLSconfig()

```
