
```
go run greeter_server/grpc_server.go --grpcport :50051 --tlsCert grpc_server_crt.pem --tlsKey grpc_server_key.pem

go run greeter_client/grpc_client.go --host localhost:50051 --cacert tls-ca.pem --servername grpc.domain.com -skipHealthCheck


docker run --net=host --add-host grpc.domain.com:127.0.0.1    -t salrashid123/grpc_app /grpc_client  \
    --host=grpc.domain.com:50051 --cacert /tls-ca.pem  \
    --servername grpc.domain.com


docker run -p 50051:50051  -t salrashid123/grpc_app /grpc_server  \
    --grpcport :50051
    --tlsCert /grpc_server_crt.pem  \
    --tlsKey /grpc_server_key.pem

```