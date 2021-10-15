httpz
======

automatic http routes, handlers, and workflows.

### Features

- Automatically convert Go functions into HTTP Handlers
- Boot time validation of all functions, no runtime type failures
- Generate clients for any language from an *httpz.Router
- Integrated Content-Security-Policy Generator with an optional report handler
- Integration points for any monitoring or metrics framework
- Built in rate-limiter and throttler.
- Static asset serving built on `fs.FS`
- Dev asset server that can serve any build toolchain
- Automatic long running job (async) endpoint handlers 
- No external dependencies
- Native encoder/decoders for JSON, Form Encoding, HTML, and Binary Files

### LICENSE

Do What The Fuck You Want To Public License (WTFPL), see LICENSE for full details

Packages within internal/ are governed by their own licenses. See internal/*/LICENSE for details
