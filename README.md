# Envoy WASM with external gRPC server

Sample for envoy with WASM filter where the filter will invoke an external GRPC service.

THe full flow is like this:

```
client -> 

    (jwt_header) -> 
      [ 
        envoy.filters.network.http_connection_manager ->
        envoy.filters.http.jwt_authn ->
        envoy.filters.http.wasm ->
      ]
      -> (api_req) -> (jwt_header) gRPC server -> (api_resp)
                                                            -> [
                                                                 envoy.filters.http.router
                                                               ] 
                                                                 -> upstream_server
```
Basically, the client transmits a jwt bearer authorization token to envoy.
Envoy will first validate the JWT header using its native `jwt_authn` filter
Once validated, the decoded JWT claims are emitted as metadata to a wasm filter
The wasm filter will extract the `sub` field metadata and use that in an rpc call to an external gRPC server.
The external grpcServer will respond back `isAdmin: true` if the `sub` field is _Alice_, otherwise the value is false.
Envoy will ultimately send the `isAdmin` header to the upstream server.
The upstream server is httpbin.org which will just display the headers it received.

This flow is very similar to how external authorization servers can be configured (shown below).  However, this repo is just a sample
which demonstrates how to configure/develop an filter.

I've done building the wasm filter the hard way...you should consider just taking a look at [wasme](https://github.com/solo-io/wasm)

The sample filter is also just a copy of the `wasm-cc` sandbox filter

References:

- [External Authorization](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/ext_authz_filter#config-http-filters-ext-authz)
- [Envoy External Authorization server (envoy.ext_authz) with OPA HelloWorld](https://github.com/salrashid123/envoy_external_authz)
- [JWT Authentication](https://www.envoyproxy.io/docs/envoy/latest/configuration/http/http_filters/jwt_authn_filter#config-http-filters-jwt-authn)

- [Wasm C++ filter](https://www.envoyproxy.io/docs/envoy/latest/start/sandboxes/wasm-cc)
- [Envoy Sandbox](https://github.com/envoyproxy/envoy/tree/master/examples)

- [Redefining extensibility in proxies - introducing WebAssembly to Envoy and Istio](https://istio.io/latest/blog/2020/wasm-announce/)
- [wasm c++ grpcCallHandler() not being called for istio 1.7.2](https://github.com/istio/istio/issues/27918)

To use this sample, you'll need:

* [bazel](https://bazel.build/)
* envoy `1.17`

```bash
docker cp `docker create envoyproxy/envoy-dev:latest`:/usr/local/bin/envoy /tmp/

/tmp/envoy --version
   version: 27c507ee0ae51713dbdf66a24cb9a47f46700b78/1.20.0-dev/Clean/RELEASE/BoringSSL
```

* golang 1.17
* optional `protoc`

---

### Setup

#### Build wasm filter

First clone envoy and build the filter

```bash
git clone https://github.com/envoyproxy/envoy.git
rm -rf envoy/examples/wasm-cc/
cp -R wasm-cc envoy/examples
```

Now build the modified filter

```bash
cd envoy 
bazel build //examples/wasm-cc:envoy_filter_http_wasm_example.wasm
```

#### Host override

Add to `/etc/hosts`

```bash
127.0.0.1	grpc.domain.com
```

This is the address for the grpc server (this is just for convenience to make the SNI match 

(you don't need to do this but i got lazy with the envoy config...TODO: configure envoy better)

#### Run Envoy

```bash
/tmp/envoy -c envoy-wasm.yaml -l debug
```

#### Run gRPC Server 

```bash
cd grpc_server/

# (optional) recompile protos
# /usr/local/bin/protoc --go_out=. --go_opt=paths=source_relative  --descriptor_set_out=echo/echo.proto.pb   --go-grpc_out=. --go-grpc_opt=paths=source_relative     echo/echo.proto

# run server
go run greeter_server/grpc_server.go --tlsCert grpc_server_crt.pem --tlsKey grpc_server_key.pem --grpcport :50051

## test client
# go run greeter_client/grpc_client.go  --host localhost:50051 --servername grpc.domain.com --cacert ../certs/tls-ca.crt
```

#### Run CLient

We're going to use curl to emit two different pregenerated JWTs 


Alice's JWT includes her name in the `sub` field

```json
{
  "alg": "RS256",
  "kid": "DHFbpoIUqrY8t2zpA2qXfCmr5VO5ZEr4RzHU_-envvQ",
  "typ": "JWT"
}.
{
  "exp": 1609408793,
  "iat": 1609108793,
  "iss": "new-issuer@secure.istio.io",
  "sub": "alice@domain.com"
}
```

And bob includes his
```json
{
  "alg": "RS256",
  "kid": "DHFbpoIUqrY8t2zpA2qXfCmr5VO5ZEr4RzHU_-envvQ",
  "typ": "JWT"
}.
{
  "exp": 1609408787,
  "iat": 1609108787,
  "iss": "new-issuer@secure.istio.io",
  "sub": "bob@domain.com"
}
```

Now use their names to invoke 


>> You can generate your own JWTs using istio's handy scripts here:

```bash
wget --no-verbose https://raw.githubusercontent.com/istio/istio/release-1.10/security/tools/jwt/samples/gen-jwt.py
wget --no-verbose https://raw.githubusercontent.com/istio/istio/release-1.10/security/tools/jwt/samples/key.pem
python3 gen-jwt.py -iss foo.bar -aud sal -sub alice@domain.com -expire 10000 key.pem
JWK URI = "https://raw.githubusercontent.com/istio/istio/release-1.10/security/tools/jwt/samples/jwks.json";
```

- Alice

```bash
curl -v -H "host: http.domain.com"  --resolve  http.domain.com:8080:127.0.0.1 \
  -H "Authorization: Bearer `cat jwts/alice.txt`" \
  -H "User: sal" http://http.domain.com:8080/get


> GET /get HTTP/1.1
> Host: http.domain.com
> User-Agent: curl/7.72.0
> Accept: */*
> Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjE2MDk0MDg3OTMsImlhdCI6MTYwOTEwODc5MywiaXNzIjoibmV3LWlzc3VlckBzZWN1cmUuaXN0aW8uaW8iLCJzdWIiOiJhbGljZUBkb21haW4uY29tIn0.WeRcHxVsKZAKD1uu-1efYhUwH9K5cWr6-Doo-CVulAhPol8oXazmZ-6wMUnqtOcWh5YOevVzUhIF8jUDibIHgsvksSprXrZf8BAkC68ctb1O0eDTlhKw0fdS41PedmBWnTESkBYFgEAKDeS4Re3bIN2irPVfSTldxqXepkl8K6R_R_Gnuyqxaie16JmIADMJ1unRbd4rcW3grXdYF4Dc7EvCpinQuQJQOdaNn1mQ2JrckTnrr8R6xf6pLpEDjAKGqeNKQdRjAAUdHSZqIylHMwIgcWAVrTFWz9TmUrmQmSqReJRa4SdAGBaTCKL9UeBiqyGYpEZn1wCfMj-ukwNQvA
> User: sal

< HTTP/1.1 200 OK
< date: Mon, 28 Dec 2020 00:55:39 GMT
< server: envoy
< access-control-allow-origin: *
< access-control-allow-credentials: true
< x-envoy-upstream-service-time: 34
< x-wasm-custom: FOO
< content-type: text/plain; charset=utf-8
< transfer-encoding: chunked

{
  "args": {}, 
  "headers": {
    "Accept": "*/*", 
    "Content-Length": "0", 
    "Host": "http.domain.com", 
    "Isadmin": "true", 
    "User": "sal", 
    "User-Agent": "curl/7.72.0", 
    "X-Amzn-Trace-Id": "Root=1-5fe92d0b-079e277b0d79542f3c3e7af6", 
    "X-Envoy-Expected-Rq-Timeout-Ms": "15000"
  }, 
  "origin": "69.250.44.79", 
  "url": "http://http.domain.com/get"
}

Hello, world
```

The authorization bearer token is the client token Alice sends.  

- `"Isadmin": "true"`: the auth header is decoded by envoy.  The `sub` field is given to the gRPC server.  If the sub=Alice, then the gRPC server adds this header back in the response.  The wasm filter will append `isAdmin:true` to the upstram.
- `x-wasm-custom: FOO`: this is a header value the wasm filter returns back to the client.
- 

- Bob

If bob tries to use his jwt token in the same way, the header he sees is `isAdmin: false`

```bash
curl -v -H "host: http.domain.com"  --resolve  http.domain.com:8080:127.0.0.1 \
  -H "Authorization: Bearer `cat jwts/bob.txt`" \
  -H "User: sal" http://http.domain.com:8080/get


> GET /get HTTP/1.1
> Host: http.domain.com
> User-Agent: curl/7.72.0
> Accept: */*
> Authorization: Bearer eyJhbGciOiJSUzI1NiIsImtpZCI6IkRIRmJwb0lVcXJZOHQyenBBMnFYZkNtcjVWTzVaRXI0UnpIVV8tZW52dlEiLCJ0eXAiOiJKV1QifQ.eyJleHAiOjE2MDk0MDg3ODcsImlhdCI6MTYwOTEwODc4NywiaXNzIjoibmV3LWlzc3VlckBzZWN1cmUuaXN0aW8uaW8iLCJzdWIiOiJib2JAZG9tYWluLmNvbSJ9.Q3QPnOkqhQN_BrDDmmSpugLRVbcyoXrXgl7NqtlUrZeef2tMQh7ycJhg4z73J6iw49v7ye2CgMrjScHTUVaGgPItItYAVfTwGXC-VBekqnrhCRhZ57ou3vJHjT7xADL9qvwahBDKjpGji8uzsvHsHZXBgiVxVh_5lYBLt6PcoVgqHAgn_uNTnE0EJJgV7Vs39k73wtxqYkuvpZdMdaWw1gLOmFhxSu2yqLHNtfLZIPyVZxyrK1KtAw9yFIDmsIEtLOpjdIqKIJ5Nh48OeN5LNhz0r2Alrj7nM_d11FYc-0k9R58vRE7SgIJNvzUKlcptkjHb0K23DoIw8QnhFFHGfg
> User: sal

< HTTP/1.1 200 OK
< date: Mon, 28 Dec 2020 01:01:31 GMT
< server: envoy
< access-control-allow-origin: *
< access-control-allow-credentials: true
< x-envoy-upstream-service-time: 41
< x-wasm-custom: FOO
< content-type: text/plain; charset=utf-8
< transfer-encoding: chunked

{
  "args": {}, 
  "headers": {
    "Accept": "*/*", 
    "Content-Length": "0", 
    "Host": "http.domain.com", 
    "Isadmin": "false", 
    "User": "sal", 
    "User-Agent": "curl/7.72.0", 
    "X-Amzn-Trace-Id": "Root=1-5fe92e6b-47819b1739fd76e15353398b", 
    "X-Envoy-Expected-Rq-Timeout-Ms": "15000"
  }, 
  "origin": "69.250.44.79", 
  "url": "http://http.domain.com/get"
}

Hello, world
```

---

A couple of notes about the flow:

- `envoy.filters.network.http_connection_manager` 
   will remove the `isAdmin` header if its sent in unilaterally by the client.
   see `internal_only_headers:  - isadmin` setting

- `envoy.filters.http.jwt_authn`
   will validate the inboud jwt and emit the claims as dynamic metadata
   see `payload_in_metadata: "my_payload"`

- `envoy.filters.http.wasm`
   will read the config file for the gRPC cluster name:
   ```json
                     configuration:
                    "@type": "type.googleapis.com/google.protobuf.StringValue"
                    value: |
                      {
                       "clustername": "grpc.domain.com",
                      } 
    ```  

    the configuration file is actually defined as a proto struct here:
    `wasm-cc/echo/echo.proto`:

    ```proto
          // this proto represents configuration for the example filter
          message Config {
            string clustername = 1;
          }
    ```

  The proto messages the wasm filter uses to make the outbound call is also defined and compiled
  with the wasm filter.  see   `wasm-cc/echo/echo.proto`