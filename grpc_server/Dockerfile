FROM golang:1.17 as build

RUN apt-get update -y && apt-get install -y build-essential wget unzip curl


RUN curl -OL https://github.com/google/protobuf/releases/download/v3.19.1/protoc-3.19.1-linux-x86_64.zip && \
    unzip protoc-3.19.1-linux-x86_64.zip -d protoc3 && \
    mv protoc3/bin/* /usr/local/bin/ && \
    mv protoc3/include/* /usr/local/include/


ENV GO111MODULE=on
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest 
RUN go install github.com/golang/protobuf/protoc-gen-go@latest

WORKDIR /app
ADD . /app
RUN go mod download

RUN protoc --go_out=. --go_opt=paths=source_relative  --descriptor_set_out=echo/echo.proto.pb   --go-grpc_out=. --go-grpc_opt=paths=source_relative     echo/echo.proto

RUN export GOBIN=/app/bin && go install greeter_server/grpc_server.go
RUN export GOBIN=/app/bin && go install greeter_client/grpc_client.go

FROM gcr.io/distroless/base
COPY --from=build /app/grpc_server_crt.pem /
COPY --from=build /app/grpc_server_key.pem /
COPY --from=build /app/tls-ca.pem /
COPY --from=build /app/bin/grpc_server /
COPY --from=build /app/bin/grpc_client /

EXPOSE 50051

#ENTRYPOINT ["/grpc_server", "--grpcport", ":50051"]
#ENTRYPOINT ["/grpc_client", "--host",  "grpc.domain.com:50051"]
