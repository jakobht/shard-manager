package smctl

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	cliv3 "github.com/urfave/cli/v3"

	"github.com/cadence-workflow/shard-manager/common/types"
)

func shardCommand(cf ClientFactory) *cliv3.Command {
	return &cliv3.Command{
		Name:        "shard",
		Aliases:     []string{"sh"},
		Usage:       "Inspect and manage shard-manager shards",
		Description: "Use --namespace/-n on the root command to identify the target namespace.",
		Commands: []*cliv3.Command{
			shardInspectCommand(cf),
		},
	}
}

// shardInspectCommand prints shard owner metadata by calling InspectShard API
func shardInspectCommand(cf ClientFactory) *cliv3.Command {
	return &cliv3.Command{
		Name:        "inspect",
		Aliases:     []string{"in"},
		Usage:       "Inspect the current owner of a shard from storage",
		Description: "Calls InspectShard on shard-manager and prints the response as JSON.",
		Flags: []cliv3.Flag{
			&cliv3.StringFlag{
				Name:    FlagShardKey,
				Aliases: []string{"sk"},
				Usage:   "shard key to look up",
			},
		},
		Action: func(ctx context.Context, cmd *cliv3.Command) error {
			return runInspectShard(ctx, cmd, resolveWriter(cmd), cf)
		},
	}
}

func runInspectShard(
	ctx context.Context,
	cmd *cliv3.Command,
	out io.Writer,
	cf ClientFactory,
) error {
	namespace := cmd.String(FlagNamespace)
	if namespace == "" {
		return fmt.Errorf("--%s is required", FlagNamespace)
	}

	shardKey := cmd.String(FlagShardKey)
	if shardKey == "" {
		return fmt.Errorf("--%s is required", FlagShardKey)
	}

	client, err := cf.ShardManagerClient(cmd)
	if err != nil {
		return err
	}

	callCtx, cancel := context.WithTimeout(ctx, cmd.Duration(FlagContextTimeout))
	defer cancel()

	resp, err := client.InspectShard(callCtx, &types.GetShardOwnerRequest{
		Namespace: namespace,
		ShardKey:  shardKey,
	})
	if err != nil {
		return fmt.Errorf("InspectShard: %w", err)
	}

	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(resp); err != nil {
		return fmt.Errorf("encode response: %w", err)
	}
	return nil
}
