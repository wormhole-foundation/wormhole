package p2p

import (
	"context"
	"errors"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
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

		// batchObsvRecvC is optional and can be set with `WithSignedObservationBatchListener`.
		batchObsvRecvC chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch]

		// obsvReqRecvC is optional and can be set with `WithObservationRequestListener`.
		obsvReqRecvC chan<- *gossipv1.ObservationRequest

		// signedIncomingVaaRecvC is optional and can be set with `WithSignedVAAListener`.
		signedIncomingVaaRecvC chan<- *gossipv1.SignedVAAWithQuorum

		// signedGovCfgRecvC is optional and can be set with `WithChainGovernorConfigListener`.
		signedGovCfgRecvC chan *gossipv1.SignedChainGovernorConfig

		// signedGovStatusRecvC is optional and can be set with `WithChainGovernorStatusListener`.
		signedGovStatusRecvC chan *gossipv1.SignedChainGovernorStatus

		// disableHeartbeatVerify is optional and can be set with `WithDisableHeartbeatVerify` or `WithGuardianOptions`.
		disableHeartbeatVerify bool

		// The following options are guardian specific. Set with `WithGuardianOptions`.
		nodeName               string
		guardianSigner         guardiansigner.GuardianSigner
		gossipControlSendC     chan []byte
		gossipAttestationSendC chan []byte
		gossipVaaSendC         chan []byte
		obsvReqSendC           <-chan *gossipv1.ObservationRequest
		acct                   *accountant.Accountant
		gov                    *governor.ChainGovernor
		components             *Components
		ibcFeaturesFunc        func() string
		processorFeaturesFunc  func() string
		gatewayRelayerEnabled  bool
		ccqEnabled             bool
		signedQueryReqC        chan<- *gossipv1.SignedQueryRequest
		queryResponseReadC     <-chan *query.QueryResponsePublication
		ccqBootstrapPeers      string
		ccqPort                uint
		ccqAllowedPeers        string
		protectedPeers         []string
		ccqProtectedPeers      []string
		featureFlags           []string
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

// WithComponents is used to set the components if you need something other than the defaults.
func WithComponents(components *Components) RunOpt {
	return func(p *RunParams) error {
		p.components = components
		return nil
	}
}

// WithProcessorFeaturesFunc is used to set the processor features function.
func WithProcessorFeaturesFunc(processorFeaturesFunc func() string) RunOpt {
	return func(p *RunParams) error {
		p.processorFeaturesFunc = processorFeaturesFunc
		return nil
	}
}

// WithSignedObservationBatchListener is used to set the channel to receive `SignedObservationBatch` messages.
func WithSignedObservationBatchListener(batchObsvC chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch]) RunOpt {
	return func(p *RunParams) error {
		p.batchObsvRecvC = batchObsvC
		return nil
	}
}

// WithSignedVAAListener is used to set the channel to receive `SignedVAAWithQuorum messages`.
func WithSignedVAAListener(signedIncomingVaaRecvC chan<- *gossipv1.SignedVAAWithQuorum) RunOpt {
	return func(p *RunParams) error {
		p.signedIncomingVaaRecvC = signedIncomingVaaRecvC
		return nil
	}
}

// WithObservationRequestListener is used to set the channel to receive `ObservationRequest` messages.
func WithObservationRequestListener(obsvReqRecvC chan<- *gossipv1.ObservationRequest) RunOpt {
	return func(p *RunParams) error {
		p.obsvReqRecvC = obsvReqRecvC
		return nil
	}
}

// WithChainGovernorConfigListener is used to set the channel to receive `SignedChainGovernorConfig` messages.
func WithChainGovernorConfigListener(signedGovCfgRecvC chan *gossipv1.SignedChainGovernorConfig) RunOpt {
	return func(p *RunParams) error {
		p.signedGovCfgRecvC = signedGovCfgRecvC
		return nil
	}
}

// WithChainGovernorStatusListener is used to set the channel to receive `SignedChainGovernorStatus` messages.
func WithChainGovernorStatusListener(signedGovStatusRecvC chan *gossipv1.SignedChainGovernorStatus) RunOpt {
	return func(p *RunParams) error {
		p.signedGovStatusRecvC = signedGovStatusRecvC
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

// WithProtectedPeers is used to set the protected peers.
func WithProtectedPeers(protectedPeers []string) RunOpt {
	return func(p *RunParams) error {
		p.protectedPeers = protectedPeers
		return nil
	}
}

// WithCcqProtectedPeers is used to set the protected peers for CCQ.
func WithCcqProtectedPeers(ccqProtectedPeers []string) RunOpt {
	return func(p *RunParams) error {
		p.ccqProtectedPeers = ccqProtectedPeers
		return nil
	}
}

// WithGuardianOptions is used to set options that are only meaningful to the guardian.
func WithGuardianOptions(
	nodeName string,
	guardianSigner guardiansigner.GuardianSigner,
	batchObsvRecvC chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservationBatch],
	signedIncomingVaaRecvC chan<- *gossipv1.SignedVAAWithQuorum,
	obsvReqRecvC chan<- *gossipv1.ObservationRequest,
	gossipControlSendC chan []byte,
	gossipAttestationSendC chan []byte,
	gossipVaaSendC chan []byte,
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
	protectedPeers []string,
	ccqProtectedPeers []string,
	featureFlags []string,
) RunOpt {
	return func(p *RunParams) error {
		p.nodeName = nodeName
		p.guardianSigner = guardianSigner
		p.batchObsvRecvC = batchObsvRecvC
		p.signedIncomingVaaRecvC = signedIncomingVaaRecvC
		p.obsvReqRecvC = obsvReqRecvC
		p.gossipControlSendC = gossipControlSendC
		p.gossipAttestationSendC = gossipAttestationSendC
		p.gossipVaaSendC = gossipVaaSendC
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
		p.protectedPeers = protectedPeers
		p.ccqProtectedPeers = ccqProtectedPeers
		p.featureFlags = featureFlags
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
		if p.guardianSigner == nil {
			return errors.New("if heart beating is enabled, guardianSigner may not be nil")
		}
	}
	if p.obsvReqSendC != nil {
		if p.guardianSigner == nil {
			return errors.New("if obsvReqSendC is not nil, vs may not be nil")
		}
	}
	return nil
}
