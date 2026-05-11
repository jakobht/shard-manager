package smctl

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	cliv3 "github.com/urfave/cli/v3"
	"go.uber.org/mock/gomock"

	"github.com/cadence-workflow/shard-manager/client/sharddistributor"
	"github.com/cadence-workflow/shard-manager/common/types"
)

func TestShardCommand_help_listsSubcommands(t *testing.T) {
	t.Parallel()
	cmd := BuildCommand()
	buf := new(bytes.Buffer)
	cmd.Writer = buf

	if err := cmd.Run(context.Background(), []string{"smctl", "shard", "--help"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "inspect") {
		t.Errorf("shard help should list 'inspect' subcommand:\n%s", out)
	}
}

func TestInspectShard(t *testing.T) {
	type setupResult struct {
		client     sharddistributor.Client
		clientErr  error
		expectArgs func(t *testing.T, cmd *cliv3.Command)
	}

	tests := []struct {
		name    string
		args    []string
		setup   func(t *testing.T, ctrl *gomock.Controller) setupResult
		wantErr string
		check   func(t *testing.T, stdout string)
	}{
		{
			name: "success: root flags propagate to shard inspect",
			args: []string{
				"smctl",
				"--" + FlagAddress, "host.example.com:7943",
				"--" + FlagNamespace, "ns-1",
				"shard", "inspect",
				"--" + FlagShardKey, "shard-1",
			},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					InspectShard(gomock.Any(), &types.GetShardOwnerRequest{
						Namespace: "ns-1",
						ShardKey:  "shard-1",
					}).
					Return(&types.GetShardOwnerResponse{
						Owner:     "executor-a",
						Namespace: "ns-1",
						Metadata:  map[string]string{"zone": "dca1"},
					}, nil)
				return setupResult{
					client: mc,
					expectArgs: func(t *testing.T, cmd *cliv3.Command) {
						if got := cmd.String(FlagAddress); got != "host.example.com:7943" {
							t.Errorf("--%s on cmd: got %q want %q", FlagAddress, got, "host.example.com:7943")
						}
						if got := cmd.String(FlagNamespace); got != "ns-1" {
							t.Errorf("--%s on cmd: got %q want %q", FlagNamespace, got, "ns-1")
						}
						if got := cmd.String(FlagShardKey); got != "shard-1" {
							t.Errorf("--%s on cmd: got %q want %q", FlagShardKey, got, "shard-1")
						}
					},
				}
			},
			check: func(t *testing.T, stdout string) {
				var resp types.GetShardOwnerResponse
				if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
					t.Fatalf("output is not valid JSON: %v\nout: %s", err, stdout)
				}
				if resp.Owner != "executor-a" {
					t.Errorf("Owner: got %q want %q", resp.Owner, "executor-a")
				}
				if resp.Namespace != "ns-1" {
					t.Errorf("Namespace: got %q want %q", resp.Namespace, "ns-1")
				}
				if got, want := resp.Metadata["zone"], "dca1"; got != want {
					t.Errorf("Metadata[zone]: got %q want %q", got, want)
				}
			},
		},
		{
			name:    "missing --namespace fails with required-flag error",
			args:    []string{"smctl", "shard", "inspect", "--" + FlagShardKey, "shard-1"},
			setup:   func(t *testing.T, ctrl *gomock.Controller) setupResult { return setupResult{} },
			wantErr: `--` + FlagNamespace + ` is required`,
		},
		{
			name:    "missing --shard-key fails with required-flag error",
			args:    []string{"smctl", "-n", "ns-1", "shard", "inspect"},
			setup:   func(t *testing.T, ctrl *gomock.Controller) setupResult { return setupResult{} },
			wantErr: `--` + FlagShardKey + ` is required`,
		},
		{
			name: "API error surfaces NamespaceNotFound",
			args: []string{"smctl", "-n", "missing", "shard", "inspect", "--" + FlagShardKey, "sk-1"},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					InspectShard(gomock.Any(), &types.GetShardOwnerRequest{
						Namespace: "missing",
						ShardKey:  "sk-1",
					}).
					Return(nil, &types.NamespaceNotFoundError{Namespace: "missing"})
				return setupResult{client: mc}
			},
			wantErr: "namespace not found missing",
		},
		{
			name: "API error surfaces ShardNotFound",
			args: []string{"smctl", "-n", "ns-1", "shard", "inspect", "--" + FlagShardKey, "missing"},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					InspectShard(gomock.Any(), &types.GetShardOwnerRequest{
						Namespace: "ns-1",
						ShardKey:  "missing",
					}).
					Return(nil, &types.ShardNotFoundError{Namespace: "ns-1", ShardKey: "missing"})
				return setupResult{client: mc}
			},
			wantErr: "shard not found ns-1:missing",
		},
		{
			name: "factory error is propagated",
			args: []string{"smctl", "-n", "ns-1", "shard", "inspect", "--" + FlagShardKey, "sk-1"},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				return setupResult{clientErr: errors.New("boom")}
			},
			wantErr: "boom",
		},
		{
			name: "context-timeout flag is honored",
			args: []string{
				"smctl",
				"--" + FlagContextTimeout, "1ms",
				"-n", "ns-1",
				"shard", "inspect",
				"--" + FlagShardKey, "sk-1",
			},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					InspectShard(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ *types.GetShardOwnerRequest, _ ...any) (*types.GetShardOwnerResponse, error) {
						deadline, ok := ctx.Deadline()
						if !ok {
							t.Errorf("ctx should have deadline from --%s", FlagContextTimeout)
							return &types.GetShardOwnerResponse{}, nil
						}
						if remaining := time.Until(deadline); remaining > 50*time.Millisecond {
							t.Errorf("--%s=1ms should produce a sub-50ms deadline, got %v", FlagContextTimeout, remaining)
						}
						return &types.GetShardOwnerResponse{
							Owner:     "executor-a",
							Namespace: "ns-1",
						}, nil
					})
				return setupResult{client: mc}
			},
		},
		{
			name: "shard alias 'sh', inspect alias 'in', and shard-key alias 'sk' work",
			args: []string{"smctl", "-n", "ns-1", "sh", "in", "-sk", "sk-1"},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					InspectShard(gomock.Any(), &types.GetShardOwnerRequest{
						Namespace: "ns-1",
						ShardKey:  "sk-1",
					}).
					Return(&types.GetShardOwnerResponse{
						Owner:     "executor-b",
						Namespace: "ns-1",
					}, nil)
				return setupResult{client: mc}
			},
			check: func(t *testing.T, stdout string) {
				var resp types.GetShardOwnerResponse
				if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
					t.Fatalf("output is not valid JSON: %v\nout: %s", err, stdout)
				}
				if resp.Owner != "executor-b" {
					t.Errorf("Owner: got %q want %q", resp.Owner, "executor-b")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			res := tt.setup(t, ctrl)

			cf := NewMockClientFactory(ctrl)
			cf.EXPECT().Close().Return(nil).Times(1)
			if tt.wantErr == "" || res.client != nil || res.clientErr != nil {
				exp := cf.EXPECT().ShardManagerClient(gomock.Any())
				if res.expectArgs != nil {
					exp = exp.Do(func(cmd *cliv3.Command) { res.expectArgs(t, cmd) })
				}
				exp.Return(res.client, res.clientErr).MaxTimes(1)
			}

			cmd := BuildCommandWithFactory(cf)
			buf := new(bytes.Buffer)
			cmd.Writer = buf
			cmd.ErrWriter = buf

			err := cmd.Run(context.Background(), tt.args)

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil. out=%s", tt.wantErr, buf.String())
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error: got %q want substring %q", err.Error(), tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("Run: %v", err)
			}
			if tt.check != nil {
				tt.check(t, buf.String())
			}
		})
	}
}
