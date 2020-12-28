package main

import (
	"crypto/tls"
	"crypto/x509"
	"echo"
	"flag"
	"io/ioutil"
	"log"
	"time"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const ()

var (
	conn *grpc.ClientConn
)

func main() {

	address := flag.String("host", "localhost:50051", "host:port of gRPC server")
	cacert := flag.String("cacert", "CA_crt.pem", "CACert for server")
	sub := flag.String("sub", "alice@domain.com", "Subject field to send")
	serverName := flag.String("servername", "server.domain.com", "CACert for server")
	useTLS := flag.Bool("useTLS", false, "useMTLS")
	flag.Parse()

	var err error

	if *useTLS {
		caCert, err := ioutil.ReadFile(*cacert)
		if err != nil {
			log.Fatalf("did not read cacert: %v", err)
		}
		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		clientCerts, err := tls.LoadX509KeyPair(
			"client.crt",
			"client.key",
		)

		tlsConfig := tls.Config{
			ServerName:   *serverName,
			Certificates: []tls.Certificate{clientCerts},
			RootCAs:      caCertPool,
		}

		creds := credentials.NewTLS(&tlsConfig)

		conn, err = grpc.Dial(*address, grpc.WithTransportCredentials(creds))
	} else {
		conn, err = grpc.Dial(*address, grpc.WithInsecure())
	}
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()

	c := echo.NewEchoServerClient(conn)
	ctx := context.Background()

	r, err := c.SayHello(ctx, &echo.EchoRequest{Name: *sub})
	if err != nil {
		log.Fatalf("could not greet: %v", err)
	}
	time.Sleep(1 * time.Second)
	log.Printf("RPC Response: %v ", r)

	// stream, err := c.SayHelloStream(ctx, &echo.EchoRequest{Name: "Stream RPC msg"})
	// if err != nil {
	// 	log.Fatalf("SayHelloStream(_) = _, %v", err)
	// }
	// for {
	// 	m, err := stream.Recv()
	// 	if err == io.EOF {
	// 		break
	// 	}
	// 	if err != nil {
	// 		log.Fatalf("SayHelloStream(_) = _, %v", err)
	// 	}
	// 	log.Printf("Message: %s", m.Message)
	// }

}
