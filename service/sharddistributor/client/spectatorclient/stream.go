package spectatorclient

import (
	"context"
	"time"

	"github.com/cadence-workflow/shard-manager/client/sharddistributor"
	"github.com/cadence-workflow/shard-manager/common/clock"
	"github.com/cadence-workflow/shard-manager/common/types"
)

const defaultStreamTimeout = 10 * time.Minute

// spectatorStream holds the state for an active GRPC stream.
// A timer cancels the stream context after a timeout to force reconnection,
// avoiding network infrastructure silently killing long-lived connections.
type spectatorStream struct {
	sharddistributor.WatchNamespaceStateClient
	ctx    context.Context
	cancel context.CancelFunc
	timer  clock.Timer
}

func newSpectatorStream(
	ctx context.Context,
	client sharddistributor.Client,
	timeSource clock.TimeSource,
	namespace string,
) (*spectatorStream, error) {
	streamCtx, streamCancel := context.WithCancel(ctx)
	timer := timeSource.JitteredAfterFunc(defaultStreamTimeout, streamCancel)

	stream, err := client.WatchNamespaceState(streamCtx, &types.WatchNamespaceStateRequest{
		Namespace: namespace,
	})
	if err != nil {
		timer.Stop()
		streamCancel()
		return nil, err
	}

	return &spectatorStream{
		WatchNamespaceStateClient: stream,
		ctx:                       streamCtx,
		cancel:                    streamCancel,
		timer:                     timer,
	}, nil
}

func (ss *spectatorStream) Close() {
	ss.timer.Stop()
	ss.cancel()
}
