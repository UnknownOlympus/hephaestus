package hermes

import (
	"fmt"

	pb "github.com/UnknownOlympus/olympus-protos/gen/go/scraper/olympus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// NewClient creates a new gRPC client for the ScraperService.
// It takes a gRPC address as input and returns the ScraperServiceClient,
// the gRPC connection, and an error if any occurs during the creation process.
// The function configures a retry policy for the gRPC client with a maximum of
// 4 attempts and exponential backoff strategy for retryable status codes.
//
// Parameters:
//   - grpcAddr: The address of the gRPC server to connect to.
//
// Returns:
//   - pb.ScraperServiceClient: The gRPC client for the ScraperService.
//   - *grpc.ClientConn: The gRPC connection object.
//   - error: An error if the client creation fails.
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
