module main

go 1.17

require (
	github.com/google/uuid v1.1.2
	github.com/salrashid123/envoy_wasm/echo v0.0.0
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b
	google.golang.org/grpc v1.33.2
)

require (
	github.com/golang-jwt/jwt/v4 v4.2.0 // indirect
	golang.org/x/sys v0.0.0-20200930185726-fdedc70b468f // indirect
	golang.org/x/text v0.3.3 // indirect
	google.golang.org/genproto v0.0.0-20200526211855-cb27e3aa2013 // indirect
	google.golang.org/protobuf v1.25.0 // indirect
)

replace github.com/salrashid123/envoy_wasm/echo => ./echo
