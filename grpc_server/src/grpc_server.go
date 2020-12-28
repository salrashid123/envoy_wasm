package main

import (
	"crypto/tls"
	"crypto/x509"
	"echo"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	jwt "github.com/dgrijalva/jwt-go"
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
	useTLS   = flag.Bool("useTLS", false, "useMTLS")
	allowSub = flag.String("allowSub", "alice@domain.com", "Allowed Subject")
	hs       *health.Server

	conn *grpc.ClientConn
)

const (
	address string = ":50051"
)

type server struct {
}

func (s *server) SayHello(ctx context.Context, in *echo.EchoRequest) (*echo.EchoReply, error) {

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

func (s *server) SayHelloStream(in *echo.EchoRequest, stream echo.EchoServer_SayHelloStreamServer) error {

	log.Println("Got stream:  -->  ")
	stream.Send(&echo.EchoReply{Message: "Hello " + in.Name})
	stream.Send(&echo.EchoReply{Message: "Hello " + in.Name})

	return nil
}

type healthServer struct{}

func (s *healthServer) Check(ctx context.Context, in *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	log.Printf("Handling grpc Check request")
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (s *healthServer) Watch(in *healthpb.HealthCheckRequest, srv healthpb.Health_WatchServer) error {
	return status.Error(codes.Unimplemented, "Watch is not implemented")
}

func main() {

	flag.Parse()

	if *grpcport == "" {
		fmt.Fprintln(os.Stderr, "missing -grpcport flag (:50051)")
		flag.Usage()
		os.Exit(2)
	}
	sopts := []grpc.ServerOption{grpc.MaxConcurrentStreams(10)}

	if *useTLS {
		clientCaCert, err := ioutil.ReadFile("CA_crt.pem")
		clientCaCertPool := x509.NewCertPool()
		clientCaCertPool.AppendCertsFromPEM(clientCaCert)

		certificate, err := tls.LoadX509KeyPair("server.crt", "server.key")
		if err != nil {
			log.Fatalf("could not load server key pair: %s", err)
		}

		customVerify := func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			for _, rawCert := range rawCerts {

				c, _ := x509.ParseCertificate(rawCert)
				log.Printf("Conn Serial Number [%d]\n", c.SerialNumber)

			}
			return nil

		}

		tlsConfig := tls.Config{
			ClientAuth:            tls.RequireAndVerifyClientCert,
			Certificates:          []tls.Certificate{certificate},
			VerifyPeerCertificate: customVerify,
			ClientCAs:             clientCaCertPool,
		}
		creds := credentials.NewTLS(&tlsConfig)

		sopts = append(sopts, grpc.Creds(creds))
	}

	s := grpc.NewServer(sopts...)
	lis, err := net.Listen("tcp", *grpcport)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	echo.RegisterEchoServerServer(s, &server{})

	healthpb.RegisterHealthServer(s, &healthServer{})

	log.Printf("Starting gRPC Server at %s", *grpcport)
	s.Serve(lis)

}
