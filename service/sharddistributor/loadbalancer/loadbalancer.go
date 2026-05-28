package loadbalancer

import (
	"fmt"
	"time"

	"github.com/cadence-workflow/shard-manager/common/log"
	"github.com/cadence-workflow/shard-manager/common/metrics"
	"github.com/cadence-workflow/shard-manager/common/types"
	"github.com/cadence-workflow/shard-manager/service/sharddistributor/config"
	"github.com/cadence-workflow/shard-manager/service/sharddistributor/loadbalancer/plan"
	"github.com/cadence-workflow/shard-manager/service/sharddistributor/loadbalancer/strategy/greedy"
	"github.com/cadence-workflow/shard-manager/service/sharddistributor/loadbalancer/strategy/naive"
	"github.com/cadence-workflow/shard-manager/service/sharddistributor/store"
)

// This module provides simulated dynamic dispatch based on the value of the
// dynamic config, which can change at any time. A regular strategy pattern
// would still need to switch implementations dynamically based on the current
// value of the dynamic config.

// PlanInitialPlacement returns planned placements for a batch of unassigned shards.
func PlanInitialPlacement(
	cfg *config.Config,
	namespace string,
	state *store.NamespaceState,
	shardIDs []string,
) ([]plan.Placement, error) {
	mode := cfg.GetLoadBalancingMode(namespace)
	switch mode {
	case types.LoadBalancingModeNAIVE:
		return naive.PlanInitialPlacement(state, shardIDs)
	case types.LoadBalancingModeGREEDY:
		return greedy.PlanInitialPlacement(state, shardIDs)
	default:
		return nil, fmt.Errorf("unsupported load balancing mode: %s", mode)
	}
}

// PlanRebalance returns planned shard moves for the current assignment state.
func PlanRebalance(
	cfg *config.Config,
	namespace string,
	state *store.NamespaceState,
	currentAssignments map[string][]string,
	now time.Time,
	logger log.Logger,
	metricsScope metrics.Scope,
) ([]plan.Move, error) {
	mode := cfg.GetLoadBalancingMode(namespace)
	switch mode {
	case types.LoadBalancingModeNAIVE:
		return naive.PlanRebalance(cfg.LoadBalancingNaive, namespace, state, currentAssignments, logger, metricsScope)
	case types.LoadBalancingModeGREEDY:
		return greedy.PlanRebalance(cfg.LoadBalancingGreedy, namespace, state, currentAssignments, now, logger, metricsScope)
	default:
		return nil, fmt.Errorf("unsupported load balancing mode: %s", mode)
	}
}
