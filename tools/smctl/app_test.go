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

func TestBuildCommand_metadata(t *testing.T) {
	t.Parallel()
	cmd := BuildCommand()
	if cmd.Name != "smctl" {
		t.Fatalf("Name: got %q want smctl", cmd.Name)
	}
	if cmd.Usage == "" {
		t.Fatal("Usage should be set")
	}
	if cmd.Version == "" {
		t.Fatal("Version should be set")
	}
}

func TestBuildCommand_help_listsRootFlagsAndCommands(t *testing.T) {
	t.Parallel()
	cmd := BuildCommand()
	buf := new(bytes.Buffer)
	cmd.Writer = buf

	if err := cmd.Run(context.Background(), []string{"smctl", "--help"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := buf.String()
	for _, want := range []string{
		"smctl",
		"namespace",
		"--" + FlagNamespace,
		"--" + FlagAddress,
		"--" + FlagTransport,
		"--" + FlagTLSCertPath,
		"--" + FlagContextTimeout,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("root help output should contain %q:\n%s", want, out)
		}
	}
}

func TestNamespaceCommand_help_listsSubcommands(t *testing.T) {
	t.Parallel()
	cmd := BuildCommand()
	buf := new(bytes.Buffer)
	cmd.Writer = buf

	if err := cmd.Run(context.Background(), []string{"smctl", "namespace", "--help"}); err != nil {
		t.Fatalf("Run: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "state") {
		t.Errorf("namespace help should list 'state' subcommand:\n%s", out)
	}
}

func TestGetNamespaceState(t *testing.T) {
	heartbeatAt := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)

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
			name: "success: --namespace and --address propagate from root through nested subcommands",
			args: []string{
				"smctl",
				"--" + FlagAddress, "host.example.com:7943",
				"--" + FlagNamespace, "ns-1",
				"namespace", "state",
			},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					GetNamespaceState(gomock.Any(), &types.GetNamespaceStateRequest{Namespace: "ns-1"}).
					Return(&types.GetNamespaceStateResponse{
						Namespace: "ns-1",
						Executors: []*types.NamespaceExecutorState{
							{
								ExecutorID:    "executor-a",
								Status:        types.ExecutorStatusACTIVE,
								LastHeartbeat: heartbeatAt,
								Metadata:      map[string]string{"zone": "dca1"},
								AssignedShards: []*types.ExecutorAssignedShardState{
									{
										ShardKey:                 "shard-1",
										AssignmentStatus:         types.AssignmentStatusREADY,
										AssignedStateModRevision: 7,
									},
								},
							},
						},
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
					},
				}
			},
			check: func(t *testing.T, stdout string) {
				var resp types.GetNamespaceStateResponse
				if err := json.Unmarshal([]byte(stdout), &resp); err != nil {
					t.Fatalf("output is not valid JSON: %v\nout: %s", err, stdout)
				}
				if resp.Namespace != "ns-1" {
					t.Errorf("namespace: got %q want %q", resp.Namespace, "ns-1")
				}
				if !strings.Contains(stdout, "ExecutorStatusACTIVE") {
					t.Errorf("status should be marshalled as enum string, got: %s", stdout)
				}
			},
		},
		{
			name:    "missing --namespace fails with required-flag error",
			args:    []string{"smctl", "namespace", "state"},
			setup:   func(t *testing.T, ctrl *gomock.Controller) setupResult { return setupResult{} },
			wantErr: `--` + FlagNamespace + ` is required`,
		},
		{
			name: "API error surfaces NamespaceNotFound",
			args: []string{"smctl", "-n", "missing", "namespace", "state"},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					GetNamespaceState(gomock.Any(), &types.GetNamespaceStateRequest{Namespace: "missing"}).
					Return(nil, &types.NamespaceNotFoundError{Namespace: "missing"})
				return setupResult{client: mc}
			},
			wantErr: "namespace not found missing",
		},
		{
			name: "factory error is propagated",
			args: []string{"smctl", "-n", "ns-1", "namespace", "state"},
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
				"namespace", "state",
			},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					GetNamespaceState(gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ *types.GetNamespaceStateRequest, _ ...any) (*types.GetNamespaceStateResponse, error) {
						deadline, ok := ctx.Deadline()
						if !ok {
							t.Errorf("ctx should have deadline from --%s", FlagContextTimeout)
							return &types.GetNamespaceStateResponse{}, nil
						}
						if remaining := time.Until(deadline); remaining > 50*time.Millisecond {
							t.Errorf("--%s=1ms should produce a sub-50ms deadline, got %v", FlagContextTimeout, remaining)
						}
						return &types.GetNamespaceStateResponse{Namespace: "ns-1"}, nil
					})
				return setupResult{client: mc}
			},
		},
		{
			name: "namespace alias 'ns' and state alias 'st' work",
			args: []string{"smctl", "-n", "ns-1", "ns", "st"},
			setup: func(t *testing.T, ctrl *gomock.Controller) setupResult {
				mc := sharddistributor.NewMockClient(ctrl)
				mc.EXPECT().
					GetNamespaceState(gomock.Any(), &types.GetNamespaceStateRequest{Namespace: "ns-1"}).
					Return(&types.GetNamespaceStateResponse{Namespace: "ns-1"}, nil)
				return setupResult{client: mc}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			res := tt.setup(t, ctrl)

			cf := NewMockClientFactory(ctrl)
			// Close is invoked through the After hook on every Run, error or not.
			cf.EXPECT().Close().Return(nil).Times(1)
			// ShardManagerClient is only invoked when the action runs (i.e. flag
			// parsing succeeded) and is the first thing the action does.
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
