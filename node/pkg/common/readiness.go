package common

import "github.com/certusone/wormhole/node/pkg/readiness"

const (
	ReadinessEthSyncing        readiness.Component = "ethSyncing"
	ReadinessSolanaSyncing     readiness.Component = "solanaSyncing"
	ReadinessTerraSyncing      readiness.Component = "terraSyncing"
	ReadinessAlgorandSyncing   readiness.Component = "algorandSyncing"
	ReadinessBSCSyncing        readiness.Component = "bscSyncing"
	ReadinessPolygonSyncing    readiness.Component = "polygonSyncing"
	ReadinessEthRopstenSyncing readiness.Component = "ethRopstenSyncing"
	ReadinessAvalancheSyncing  readiness.Component = "avalancheSyncing"
	ReadinessOasisSyncing      readiness.Component = "oasisSyncing"
	ReadinessFantomSyncing     readiness.Component = "fantomSyncing"
)
