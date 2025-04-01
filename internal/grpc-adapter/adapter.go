package geyserAdapter

import (
	"crypto/x509"
	"fmt"
	"net/url"
	"time"

	pb "github.com/rpcpool/yellowstone-grpc/examples/golang/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type GeyserUtils struct {
}

func NewGeyserAdapter() GeyserUtils {
	return GeyserUtils{}
}

func (x GeyserUtils) CreateSubscriptionRequest() *pb.SubscribeRequest {
	f := false
	return &pb.SubscribeRequest{
		Transactions: map[string]*pb.SubscribeRequestFilterTransactions{
			"": {
				Failed:         &f,
				AccountInclude: []string{"6EF8rrecthR5Dkzon8Nwu78hRvfCKubJ14M5uBEwF6P"},
			},
		},
	}
}

func (x GeyserUtils) CreateGRPCConnection(endpoint string) (*grpc.ClientConn, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid endpoint URL: %w", err)
	}

	port := u.Port()
	if port == "" {
		port = "80"
	}

	opts := []grpc.DialOption{

		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                1000 * time.Second,
			Timeout:             time.Second,
			PermitWithoutStream: true,
		}),
	}

	if u.Scheme == "https" {
		pool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to get system cert pool: %w", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(pool, "")))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	return grpc.Dial(fmt.Sprintf("%s:%s", u.Hostname(), port), opts...)
}
