package hermes_test

import (
	"testing"

	"github.com/UnknownOlympus/hephaestus/internal/client/hermes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		client, conn, err := hermes.NewClient("bufnet")

		require.NoError(t, err)
		assert.NotNil(t, client)
		assert.NotNil(t, conn)
	})

	t.Run("error - failed to create client", func(t *testing.T) {
		t.Parallel()
		client, conn, err := hermes.NewClient("Segment%%2815197306101420000%29.ts")

		require.Error(t, err)
		require.ErrorContains(t, err, "failed to create grpc client")
		assert.Nil(t, client)
		assert.Nil(t, conn)
	})
}
