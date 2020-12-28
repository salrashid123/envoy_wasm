module main

go 1.15

require (
	echo v0.0.0
	github.com/dgrijalva/jwt-go v3.2.0+incompatible // indirect
	golang.org/x/net v0.0.0-20201110031124-69a78807bb2b // indirect
	google.golang.org/grpc v1.33.2 // indirect
)

replace echo => ./src/echo
