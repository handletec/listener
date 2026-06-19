# Changelog

## Unreleased

### Breaking Changes

#### `NewCORS()` default `AllowedOrigins` changed from wildcard to deny-all

**Before:** `NewCORS()` defaulted `AllowedOrigins` to `["https://*", "http://*"]`, permitting all HTTPS and HTTP origins.

**After:** `AllowedOrigins` defaults to `nil`. All cross-origin requests are denied unless callers explicitly call `SetOrigins`.

**Migration:** Call `cors.SetOrigins([]string{"https://example.com"})` with the specific origins your service accepts. Never combine wildcard origins (`"*"` or `"https://*"`) with `AllowCredentials = true`.

#### `ForClient` now returns `(*tls.Config, error)`

**Before:**

```go
tlsConfig := builder.ForClient()
```

**After:**

```go
tlsConfig, err := builder.ForClient()
if err != nil {
    return err
}
```

This change is required because `ForClient` now loads client certificate and key files
configured via `SetCertKeyFile` if the key pair has not already been loaded. File I/O
can fail, so an error return is necessary.

### Security Fixes

- `ForClient` and `ForServer` now return a snapshot clone of the internal CA pool.
  Subsequent mutations to the builder (e.g. `AddCABytes`, `AddCAFile`) no longer affect
  already-returned `tls.Config` values.
- `SetInsecureSkipVerify` and `SetClientAuth` now acquire the internal mutex before
  writing, eliminating a data race with concurrent `ForClient`/`ForServer` calls.
- `startWatcher` now uses `sync.Once` to prevent a data race when `ForServer` is called
  from multiple goroutines.
- `Close` now holds the mutex while signalling shutdown and clearing the watcher
  reference, making it fully idempotent and race-safe.

### Bug Fixes

- `AddCABytes` now returns an error when the input is empty or contains no `CERTIFICATE`
  PEM blocks. Previously it returned nil silently.
- `VerifyCertTrusted` now PEM-decodes its input before parsing certificates. The
  previous implementation passed raw PEM bytes directly to `x509.ParseCertificates`,
  which expects DER. The function now treats the first certificate in the chain as the
  leaf and any remaining certificates as intermediates, verified with
  `ExtKeyUsageServerAuth`.
- `ForClient` now loads a client certificate from files configured via `SetCertKeyFile`
  when `SetCertKeyFromBytes` has not been called. Previously it only injected a
  certificate that was already loaded in memory.
