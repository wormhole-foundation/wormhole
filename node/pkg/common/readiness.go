package common

import "github.com/certusone/wormhole/node/pkg/readiness"

const (
	ReadinessEthSyncing    readiness.Component = "ethSyncing"
	ReadinessSolanaSyncing readiness.Component = "solanaSyncing"
	ReadinessTerraSyncing  readiness.Component = "terraSyncing"
	ReadinessBSCSyncing    readiness.Component = "bscSyncing"
)
