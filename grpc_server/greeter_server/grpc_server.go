package main

import (
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/salrashid123/envoy_wasm/echo"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	jwt "github.com/golang-jwt/jwt/v4"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/metadata"
)

type MyCustomClaims struct {
	Uid string   `json:"uid"`
	Aud []string `json:"aud"` // https://github.com/dgrijalva/jwt-go/pull/308
	jwt.StandardClaims
}

var (
	grpcport = flag.String("grpcport", ":50051", "grpcport")
	tlsCert  = flag.String("tlsCert", "grpc_server_crt.pem", "tls Certificate")
	tlsKey   = flag.String("tlsKey", "grpc_server_key.pem", "tls Key")
	insecure = flag.Bool("insecure", false, "startup without TLS")
	allowSub = flag.String("allowSub", "alice@domain.com", "Allowed Subject")
	hs       *health.Server

	conn *grpc.ClientConn
)

const (
	address string = ":50051"
)

// server is used to implement echo.EchoServer.
type server struct {
	// Embed the unimplemented server
	echo.UnimplementedEchoServerServer
}
type healthServer struct{}

func (s *healthServer) Check(ctx context.Context, in *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	log.Printf("Handling grpc Check request: " + in.Service)
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (s *healthServer) Watch(in *healthpb.HealthCheckRequest, srv healthpb.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "Watch is not implemented")
}

func (s *server) SayHelloUnary(ctx context.Context, in *echo.EchoRequest) (*echo.EchoReply, error) {

	log.Println("Got rpc: --> ", in.Name)
	log.Println("Request ctx %v", ctx)

	headers, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, grpc.Errorf(codes.PermissionDenied, fmt.Sprintf("Could not read inbound metadata"))
	}
	log.Println("Metadata Headers %v", headers)

	if in.Name == *allowSub {
		return &echo.EchoReply{Message: "true"}, nil
	} else {
		return &echo.EchoReply{Message: "false"}, nil

	}
}

func (s *server) SayHelloServerStream(in *echo.EchoRequest, stream echo.EchoServer_SayHelloServerStreamServer) error {

	log.Println("Got stream:  -->  ")
	stream.Send(&echo.EchoReply{Message: "Hello " + in.Name})
	stream.Send(&echo.EchoReply{Message: "Hello " + in.Name})

	return nil
}

func main() {
	flag.Parse()
	if *grpcport == "" {
		flag.Usage()
		log.Fatalf("missing -grpcport flag (:50051)")
	}

	lis, err := net.Listen("tcp", *grpcport)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	sopts := []grpc.ServerOption{}
	if *insecure == false {
		if *tlsCert == "" || *tlsKey == "" {
			log.Fatalf("Must set --tlsCert and tlsKey if --insecure flags is not set")
		}
		ce, err := credentials.NewServerTLSFromFile(*tlsCert, *tlsKey)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		sopts = append(sopts, grpc.Creds(ce))
	}

	s := grpc.NewServer(sopts...)
	echo.RegisterEchoServerServer(s, &server{})

	healthpb.RegisterHealthServer(s, &healthServer{})

	log.Printf("Starting server...")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}
