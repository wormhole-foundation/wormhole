package p2p

import (
	"context"
	"crypto/ecdsa"
	"errors"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/governor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/libp2p/go-libp2p/core/crypto"
)

type (
	// RunParams is used to pass parameters into `p2p.Run()`. It allows applications to specify only what they need.
	RunParams struct {
		// These parameters are always required.
		bootstrapPeers string
		networkID      string
		priv           crypto.PrivKey
		gst            *common.GuardianSetState
		rootCtxCancel  context.CancelFunc

		// obsvC is optional and can be set with `WithSignedObservationListener`.
		obsvC chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservation]

		// obsvReqC is optional and can be set with `WithObservationRequestListener`.
		obsvReqC chan<- *gossipv1.ObservationRequest

		// signedInC is optional and can be set with `WithSignedVAAListener`.
		signedInC chan<- *gossipv1.SignedVAAWithQuorum

		// signedGovCfg is optional and can be set with `WithChainGovernorConfigListener`.
		signedGovCfg chan *gossipv1.SignedChainGovernorConfig

		// WithChainGovernorStatusListener is optional and can be set with `WithChainGovernorStatusListener`.
		signedGovSt chan *gossipv1.SignedChainGovernorStatus

		// disableHeartbeatVerify is optional and can be set with `WithDisableHeartbeatVerify` or `WithGuardianOptions`.
		disableHeartbeatVerify bool

		// The following options are guardian specific. Set with `WithGuardianOptions`.
		nodeName              string
		gk                    *ecdsa.PrivateKey
		gossipSendC           chan []byte
		obsvReqSendC          <-chan *gossipv1.ObservationRequest
		acct                  *accountant.Accountant
		gov                   *governor.ChainGovernor
		components            *Components
		ibcFeaturesFunc       func() string
		gatewayRelayerEnabled bool
		ccqEnabled            bool
		signedQueryReqC       chan<- *gossipv1.SignedQueryRequest
		queryResponseReadC    <-chan *query.QueryResponsePublication
		ccqBootstrapPeers     string
		ccqPort               uint
		ccqAllowedPeers       string

		// This is junk:
		gossipControlSendC     chan []byte
		gossipAttestationSendC chan []byte
		gossipVaaSendC         chan []byte
	}

	// RunOpt is used to specify optional parameters.
	RunOpt func(p *RunParams) error
)

// NewRunParams is used to create the `RunParams` which gets passed into `p2p.Run()`. It takes the required parameters,
// plus any desired optional ones, which can be set using the various `With` functions defined below.
func NewRunParams(
	bootstrapPeers string,
	networkID string,
	priv crypto.PrivKey,
	gst *common.GuardianSetState,
	rootCtxCancel context.CancelFunc,
	opts ...RunOpt,
) (*RunParams, error) {
	p := &RunParams{
		bootstrapPeers: bootstrapPeers,
		networkID:      networkID,
		priv:           priv,
		gst:            gst,
		rootCtxCancel:  rootCtxCancel,
	}

	for _, opt := range opts {
		err := opt(p)
		if err != nil {
			return nil, err
		}
	}

	if err := p.verify(); err != nil {
		return nil, err
	}

	return p, nil
}

// WithSignedObservationListener is used to set the channel to receive `SignedObservation“ messages.
func WithSignedObservationListener(obsvC chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservation]) RunOpt {
	return func(p *RunParams) error {
		p.obsvC = obsvC
		return nil
	}
}

// WithSignedVAAListener is used to set the channel to receive `SignedVAAWithQuorum messages.
func WithSignedVAAListener(signedInC chan<- *gossipv1.SignedVAAWithQuorum) RunOpt {
	return func(p *RunParams) error {
		p.signedInC = signedInC
		return nil
	}
}

// WithObservationRequestListener is used to set the channel to receive `ObservationRequest messages.
func WithObservationRequestListener(obsvReqC chan<- *gossipv1.ObservationRequest) RunOpt {
	return func(p *RunParams) error {
		p.obsvReqC = obsvReqC
		return nil
	}
}

// WithChainGovernorConfigListener is used to set the channel to receive `SignedChainGovernorConfig messages.
func WithChainGovernorConfigListener(signedGovCfg chan *gossipv1.SignedChainGovernorConfig) RunOpt {
	return func(p *RunParams) error {
		p.signedGovCfg = signedGovCfg
		return nil
	}
}

// WithChainGovernorStatusListener is used to set the channel to receive `SignedChainGovernorStatus messages.
func WithChainGovernorStatusListener(signedGovSt chan *gossipv1.SignedChainGovernorStatus) RunOpt {
	return func(p *RunParams) error {
		p.signedGovSt = signedGovSt
		return nil
	}
}

// WithDisableHeartbeatVerify is used to set disableHeartbeatVerify.
func WithDisableHeartbeatVerify(disableHeartbeatVerify bool) RunOpt {
	return func(p *RunParams) error {
		p.disableHeartbeatVerify = disableHeartbeatVerify
		return nil
	}
}

// WithGuardianOptions is used to set options that are only meaningful to the guardian.
func WithGuardianOptions(
	nodeName string,
	gk *ecdsa.PrivateKey,
	obsvC chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservation],
	signedInC chan<- *gossipv1.SignedVAAWithQuorum,
	obsvReqC chan<- *gossipv1.ObservationRequest,
	gossipSendC chan []byte,
	obsvReqSendC <-chan *gossipv1.ObservationRequest,
	acct *accountant.Accountant,
	gov *governor.ChainGovernor,
	disableHeartbeatVerify bool,
	components *Components,
	ibcFeaturesFunc func() string,
	gatewayRelayerEnabled bool,
	ccqEnabled bool,
	signedQueryReqC chan<- *gossipv1.SignedQueryRequest,
	queryResponseReadC <-chan *query.QueryResponsePublication,
	ccqBootstrapPeers string,
	ccqPort uint,
	ccqAllowedPeers string,
) RunOpt {
	return func(p *RunParams) error {
		p.nodeName = nodeName
		p.gk = gk
		p.obsvC = obsvC
		p.signedInC = signedInC
		p.obsvReqC = obsvReqC
		p.gossipSendC = gossipSendC
		p.obsvReqSendC = obsvReqSendC
		p.acct = acct
		p.gov = gov
		p.disableHeartbeatVerify = disableHeartbeatVerify
		p.components = components
		p.ibcFeaturesFunc = ibcFeaturesFunc
		p.gatewayRelayerEnabled = gatewayRelayerEnabled
		p.ccqEnabled = ccqEnabled
		p.signedQueryReqC = signedQueryReqC
		p.queryResponseReadC = queryResponseReadC
		p.ccqBootstrapPeers = ccqBootstrapPeers
		p.ccqPort = ccqPort
		p.ccqAllowedPeers = ccqAllowedPeers
		return nil
	}
}

// verify is used to verify the RunParams object.
func (p *RunParams) verify() error {
	if p.bootstrapPeers == "" {
		return errors.New("bootstrapPeers may not be nil")
	}
	if p.networkID == "" {
		return errors.New("networkID may not be nil")
	}
	if p.priv == nil {
		return errors.New("priv may not be nil")
	}
	if p.gst == nil {
		return errors.New("gst may not be nil")
	}
	if p.rootCtxCancel == nil {
		return errors.New("rootCtxCancel may not be nil")
	}
	if p.nodeName != "" { // Heartbeating is enabled.
		if p.gk == nil {
			return errors.New("if heart beating is enabled, gk may not be nil")
		}
	}
	if p.obsvReqSendC != nil {
		if p.gk == nil {
			return errors.New("if obsvReqSendC is not nil, gk may not be nil")
		}
	}
	return nil
}
