package hermes

import (
	"fmt"

	pb "github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func NewClient(grpcAddr string) (pb.ScraperServiceClient, *grpc.ClientConn, error) {
	retrypolicy := `{
		"methodConfig": [{
			"name": [{}],
			"retryPolicy": {
				"maxAttempts": 4,
				"initialBackoff": ".01s",
				"maxBackoff": "1s",
				"backoffMultiplier": 2,
				"retryableStatusCodes": [ "UNAVAILABLE" ]
			}
		}]
	}`

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultServiceConfig(retrypolicy),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create grpc client: %w", err)
	}

	return pb.NewScraperServiceClient(conn), conn, nil
}
