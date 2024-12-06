package tss

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/internal/testutils"
	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	tsscommon "github.com/yossigi/tss-lib/v2/common"
	"github.com/yossigi/tss-lib/v2/ecdsa/party"
	"github.com/yossigi/tss-lib/v2/ecdsa/signing"
	"github.com/yossigi/tss-lib/v2/tss"
)

var (
	unicastRounds   = []signingRound{round1Message1, round2Message}
	broadcastRounds = []signingRound{
		round1Message2,
		round3Message,
		round4Message,
		round5Message,
		round6Message,
		round7Message,
		round8Message,
		round9Message,
	}

	allRounds = append(unicastRounds, broadcastRounds...)
)

func parsedIntoEcho(a *assert.Assertions, t *Engine, parsed tss.ParsedMessage) *IncomingMessage {
	payload, _, err := parsed.WireBytes()
	a.NoError(err)

	msg := &tsscommv1.Echo{
		Message: &tsscommv1.SignedMessage{
			Content: &tsscommv1.SignedMessage_TssContent{
				TssContent: &tsscommv1.TssContent{Payload: payload},
			},
			Sender:    partyIdToProto(t.Self),
			Signature: nil,
		},
	}
	a.NoError(t.sign(msg.Message))

	return &IncomingMessage{
		Source: partyIdToProto(t.Self),
		Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{
				Echo: msg,
			},
		},
	}
}

func (i *IncomingMessage) setSource(id *tss.PartyID) {
	i.Source = partyIdToProto(id)
}

func TestBroadcast(t *testing.T) {

	// The tests here rely on n=5, threshold=2, meaning 3 guardians are needed to sign (f<=1).
	t.Run("forLeaderCreatingMessage", func(t *testing.T) {
		a := assert.New(t)
		// f = 1, n = 5
		engines := load5GuardiansSetupForBroadcastChecks(a)

		e1 := engines[0]
		// make parsedMessage, and insert into e1
		// then add another one for the same round.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self, e1.Self, rnd, party.Digest{byte(j)})

			echo := parsedIntoEcho(a, e1, parsed1)

			shouldBroadcast, shouldDeliver, err := e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.True(shouldBroadcast)
			a.False(shouldDeliver)
		}
	})

	t.Run("forEnoughEchos", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2, e3 := engines[0], engines[1], engines[2]

		// two different signers on an echo, meaning it will receive from two players.
		// since f=1 and we have f+1 echos: it should broadcast at the end of this test.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self, e1.Self, rnd, party.Digest{byte(j)})

			echo := parsedIntoEcho(a, e1, parsed1)
			echo.setSource(e2.Self)

			shouldBroadcast, shouldDeliver, err := e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.False(shouldDeliver)

			echo.setSource(e3.Self)

			shouldBroadcast, shouldDeliver, err = e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.True(shouldBroadcast)
			a.False(shouldDeliver)
		}
	})
}

func load5GuardiansSetupForBroadcastChecks(a *assert.Assertions) []*Engine {
	engines, err := loadGuardians(5, "tss5") // f=1, n=5.
	a.NoError(err)

	for _, v := range engines {
		v.GuardianStorage.Threshold = 2 // meaning 3 guardians are needed to sign.
	}

	return engines
}

func TestDeliver(t *testing.T) {
	t.Run("After2fPlus1Messages", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2, e3 := engines[0], engines[1], engines[2]

		// two different signers on an echo, meaning it will receive from two players.
		// since f=1 and we have f+1 echos: it should broadcast at the end of this test.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self, e1.Self, rnd, party.Digest{byte(j)})

			echo := parsedIntoEcho(a, e1, parsed1)
			echo.setSource(e2.Self)

			shouldBroadcast, shouldDeliver, err := e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.False(shouldDeliver)

			echo.setSource(e3.Self)

			shouldBroadcast, shouldDeliver, err = e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.True(shouldBroadcast)
			a.False(shouldDeliver)

			echo.setSource(e1.Self)

			shouldBroadcast, shouldDeliver, err = e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.True(shouldDeliver)
		}
	})

	t.Run("doesn'tDeliverTwice", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2, e3, e4 := engines[0], engines[1], engines[2], engines[3]

		// two different signers on an echo, meaning it will receive from two players.
		// since f=1 and we have f+1 echos: it should broadcast at the end of this test.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self, e1.Self, rnd, party.Digest{byte(j)})
			echo := parsedIntoEcho(a, e1, parsed1)
			echo.setSource(e2.Self)

			shouldBroadcast, shouldDeliver, err := e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.False(shouldDeliver)

			echo.setSource(e3.Self)

			shouldBroadcast, shouldDeliver, err = e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.True(shouldBroadcast)
			a.False(shouldDeliver)

			echo.setSource(e1.Self)

			shouldBroadcast, shouldDeliver, err = e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.True(shouldDeliver)

			echo.setSource(e4.Self)

			shouldBroadcast, shouldDeliver, err = e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.False(shouldDeliver)
		}
	})
}

func TestUuidNotAffectedByMessageContentChange(t *testing.T) {
	a := assert.New(t)
	engines := load5GuardiansSetupForBroadcastChecks(a)
	e1 := engines[0]
	for i, rnd := range allRounds {
		trackingId := party.Digest{byte(i)}

		// each message is generated with some random content inside.
		parsed1 := generateFakeParsedMessageWithRandomContent(e1.Self, e1.Self, rnd, trackingId)
		parsed2 := generateFakeParsedMessageWithRandomContent(e1.Self, e1.Self, rnd, trackingId)

		uid1, err := parsed1.getUUID(e1.LoadDistributionKey)
		a.NoError(err)

		uid2, err := parsed2.getUUID(e1.LoadDistributionKey)
		a.NoError(err)
		a.Equal(uid1, uid2)
	}
}

func TestEquivocation(t *testing.T) {
	t.Run("inBroadcastLogic", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2 := engines[0], engines[1]

		for i, rndType := range allRounds {

			trackingId := party.Digest{byte(i)}

			parsed1 := generateFakeMessageWithRandomContent(e1.Self, e2.Self, rndType, trackingId)

			shouldBroadcast, shouldDeliver, err := e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, parsedIntoEcho(a, e2, parsed1))
			a.NoError(err)
			a.True(shouldBroadcast) //should broadcast since e2 is the source of this message.
			a.False(shouldDeliver)

			parsed2 := generateFakeMessageWithRandomContent(e1.Self, e2.Self, rndType, trackingId)

			shouldBroadcast, shouldDeliver, err = e1.relbroadcastInspection(&parsedTssContent{parsed2, ""}, parsedIntoEcho(a, e2, parsed2))
			a.ErrorAs(err, &ErrEquivicatingGuardian)
			a.False(shouldBroadcast)
			a.False(shouldDeliver)

			equvicatingEchoerMessage := parsedIntoEcho(a, e2, parsed1)
			equvicatingEchoerMessage.
				Content.
				GetEcho().
				Message.
				Content.(*tsscommv1.SignedMessage_TssContent).
				TssContent.
				Payload[0] += 1
			// now echoer is equivicating (change content, but of some seen message):
			_, _, err = e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, equvicatingEchoerMessage)
			a.ErrorContains(err, e2.Self.Id)
		}
	})

	t.Run("inUnicast", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2 := engines[0], engines[1]

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cncl := context.WithCancel(supctx)
		defer cncl()

		e1.Start(ctx)
		e2.Start(ctx)

		for i, rndType := range unicastRounds {

			trackingId := party.Digest{byte(i)}

			parsed1 := generateFakeMessageWithRandomContent(e1.Self, e2.Self, rndType, trackingId)
			parsed2 := generateFakeMessageWithRandomContent(e1.Self, e2.Self, rndType, trackingId)

			bts, _, err := parsed1.WireBytes()
			a.NoError(err)

			msg := &IncomingMessage{
				Content: &tsscommv1.PropagatedMessage{
					Message: &tsscommv1.PropagatedMessage_Unicast{
						Unicast: &tsscommv1.Unicast{
							Content: &tsscommv1.TssContent{
								Payload:         bts,
								MsgSerialNumber: 0,
							},
						},
					},
				},
			}

			msg.setSource(e1.Self)

			e2.handleUnicast(msg)

			bts, _, err = parsed2.WireBytes()
			a.NoError(err)

			msg.Content.Message.(*tsscommv1.PropagatedMessage_Unicast).Unicast.Content.Payload = bts
			a.ErrorIs(e2.handleUnicast(msg), ErrEquivicatingGuardian)
		}
	})
}

func TestBadInputs(t *testing.T) {
	a := assert.New(t)
	engines := load5GuardiansSetupForBroadcastChecks(a)
	e1, e2 := engines[0], engines[1]

	supctx := testutils.MakeSupervisorContext(context.Background())
	ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
	defer cancel()

	e1.Start(ctx) // so it has a logger.

	t.Run("signature", func(t *testing.T) {
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self, e1.Self, rnd, party.Digest{byte(j)})
			echo := parsedIntoEcho(a, e1, parsed1)

			echo.setSource(e2.Self)

			echo.toEcho().Message.Signature[0] += 1
			_, _, err := e1.relbroadcastInspection(&parsedTssContent{parsed1, ""}, echo)
			a.ErrorIs(err, ErrInvalidSignature)

			if rnd == round1Message1 || rnd == round2Message {
				continue
			}

			echo.setSource(e1.Self)
			err = e1.handleIncomingTssMessage(echo)
			a.ErrorIs(err, ErrInvalidSignature)
			e1.HandleIncomingTssMessage(echo) // to ensure we go through some code path, nothing to check really.
		}
	})

	t.Run("incoming message", func(t *testing.T) {
		var tmp *Engine = nil
		// these tests ensure we don't panic on bad inputs.
		// Shouldn't fail or panic.
		tmp.HandleIncomingTssMessage(nil)
		e1.HandleIncomingTssMessage(nil)
		e2.HandleIncomingTssMessage(nil) // e2 hadn't started.

		err := tmp.handleIncomingTssMessage(nil)
		a.ErrorIs(err, errNilIncoming)

		err = e1.handleIncomingTssMessage(&IncomingMessage{})
		a.ErrorIs(err, errNilSource)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: partyIdToProto(e2.Self)})
		a.ErrorIs(err, errNeitherBroadcastNorUnicast)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source:  partyIdToProto(e2.Self),
			Content: &tsscommv1.PropagatedMessage{}})
		a.ErrorIs(err, errNeitherBroadcastNorUnicast)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source: partyIdToProto(e2.Self),
			Content: &tsscommv1.PropagatedMessage{
				Message: &tsscommv1.PropagatedMessage_Echo{},
			},
		})
		a.ErrorIs(err, ErrEchoIsNil)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source: partyIdToProto(e2.Self),
			Content: &tsscommv1.PropagatedMessage{
				Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{}},
			},
		})
		a.ErrorIs(err, ErrSignedMessageIsNil)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source: partyIdToProto(e2.Self),
			Content: &tsscommv1.PropagatedMessage{
				Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
					Message: &tsscommv1.SignedMessage{},
				}}},
		})
		a.ErrorIs(err, ErrNilPartyId)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source: partyIdToProto(e2.Self),
			Content: &tsscommv1.PropagatedMessage{
				Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
					Message: &tsscommv1.SignedMessage{
						Sender: &tsscommv1.PartyId{},
					},
				}}},
		})
		a.ErrorIs(err, ErrEmptyIDInPID)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source: partyIdToProto(e2.Self),
			Content: &tsscommv1.PropagatedMessage{
				Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
					Message: &tsscommv1.SignedMessage{
						Sender: &tsscommv1.PartyId{
							Id:  "a",
							Key: []byte{},
						},
					},
				}}},
		})
		a.ErrorIs(err, ErrEmptyKeyInPID)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: partyIdToProto(e2.Self), Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
				Message: &tsscommv1.SignedMessage{
					Sender: partyIdToProto(e2.Self),
				},
			}}},
		})
		a.ErrorIs(err, ErrNoContent)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: partyIdToProto(e2.Self), Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
				Message: &tsscommv1.SignedMessage{
					Content: &tsscommv1.SignedMessage_TssContent{
						TssContent: &tsscommv1.TssContent{},
					},
					Sender: partyIdToProto(e2.Self),
				},
			}}},
		})
		a.ErrorIs(err, ErrNilPayload)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: partyIdToProto(e2.Self), Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
				Message: &tsscommv1.SignedMessage{
					Content: &tsscommv1.SignedMessage_TssContent{
						TssContent: &tsscommv1.TssContent{
							Payload: []byte{1, 2, 3},
						},
					},
					Sender: partyIdToProto(e2.Self),
				},
			}}},
		})
		a.ErrorIs(err, ErrNoAuthenticationField)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: partyIdToProto(e2.Self), Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
				Message: &tsscommv1.SignedMessage{
					Content: &tsscommv1.SignedMessage_TssContent{
						TssContent: &tsscommv1.TssContent{
							Payload: []byte{1, 2, 3},
						},
					},
					Sender:    partyIdToProto(e2.Self),
					Signature: []byte{1, 2, 3},
				},
			}}},
		})
		a.ErrorContains(err, "cannot parse")
	})

	t.Run("Begin signing", func(t *testing.T) {
		var tmp *Engine = nil
		engines2 := load5GuardiansSetupForBroadcastChecks(a)

		a.ErrorIs(tmp.BeginAsyncThresholdSigningProtocol(nil, 0), errNilTssEngine)
		a.ErrorIs(e2.BeginAsyncThresholdSigningProtocol(nil, 0), errTssEngineNotStarted)

		tmp = engines2[1]
		tmp.started.Store(started)

		a.ErrorContains(e1.BeginAsyncThresholdSigningProtocol(make([]byte, 12), 0), "length is not 32 bytes")

		tmp.fp = nil
		a.ErrorContains(tmp.BeginAsyncThresholdSigningProtocol(nil, 0), "not set up correctly")
	})

	t.Run("fetch certificate", func(t *testing.T) {
		_, err := e1.FetchCertificate(nil)
		a.ErrorIs(err, ErrNilPartyId)

		_, err = e1.FetchCertificate(&tsscommv1.PartyId{})
		a.ErrorContains(err, "not found")
	})
}

func TestFetchPartyId(t *testing.T) {
	a := assert.New(t)
	engines := load5GuardiansSetupForBroadcastChecks(a)
	e1 := engines[0]
	pid, err := e1.FetchPartyId(e1.guardiansCerts[0])
	a.NoError(err)
	a.Equal(e1.Self.Id, pid.Id)

	crt := createX509Cert("localhost")
	_, err = e1.FetchPartyId(crt)
	a.ErrorContains(err, "unsupported") // cert.PublicKey=nil

	crt.PublicKey = []byte{1, 2, 3}
	_, err = e1.FetchPartyId(crt)
	a.ErrorContains(err, "unknown")
}

func TestCleanup(t *testing.T) {
	a := assert.New(t)
	engines := load5GuardiansSetupForBroadcastChecks(a)
	e1 := engines[0]

	uuid1 := uuid{1}
	e1.received[uuid1] = &broadcaststate{
		timeReceived: time.Now().Add(time.Minute * 10 * (-1)),
		trackingId: &tsscommon.TrackingID{
			Digest: uuid1[:],
		},
	}

	uuid2 := uuid{2}
	e1.received[uuid2] = &broadcaststate{
		timeReceived: time.Now(),
		trackingId: &tsscommon.TrackingID{
			Digest: uuid2[:],
		},
	}

	e1.cleanup(time.Minute * 5) // if more than 5 minutes passed -> delete
	a.Len(e1.received, 1)
	_, ok := e1.received[uuid{1}]
	a.False(ok)

	_, ok = e1.received[uuid{2}]
	a.True(ok)
}

type badtssMessage struct {
}

func (b *badtssMessage) GetFrom() *tss.PartyID         { panic("unimplemented") }
func (b *badtssMessage) GetTo() []*tss.PartyID         { panic("unimplemented") }
func (b *badtssMessage) IsBroadcast() bool             { panic("unimplemented") }
func (b *badtssMessage) IsToOldAndNewCommittees() bool { panic("unimplemented") }
func (b *badtssMessage) IsToOldCommittee() bool        { panic("unimplemented") }
func (b *badtssMessage) String() string                { panic("unimplemented") }
func (b *badtssMessage) Type() string                  { panic("unimplemented") }
func (b *badtssMessage) WireMsg() *tss.MessageWrapper {
	return &tss.MessageWrapper{
		TrackingID: nil,
	}
}
func (b *badtssMessage) WireBytes() ([]byte, *tss.MessageRouting, error) {
	return nil, nil, errors.New("bad message")
}

func TestRouteCheck(t *testing.T) {
	// this test is a bit of a hack.
	// To ensure we don't panic on bad inputs.
	a := assert.New(t)
	engines := load5GuardiansSetupForBroadcastChecks(a)
	e1 := engines[0]

	supctx := testutils.MakeSupervisorContext(context.Background())
	ctx, cancel := context.WithTimeout(supctx, time.Second*5)
	defer cancel()

	e1.Start(ctx)
	e1.fpOutChan <- &badtssMessage{}
	e1.fpErrChannel <- tss.NewTrackableError(errors.New("test"), "test", -1, nil, &tsscommon.TrackingID{})
	e1.fpErrChannel <- nil

	time.Sleep(time.Millisecond * 200)
}

func TestE2E(t *testing.T) {
	// Setting up all engines (not just 5), each with a different guardian storage.
	// all will attempt to sign a single message, while outputing messages to each other,
	// and reliably broadcasting them.

	t.Run("with correct metrics", func(t *testing.T) {
		inProgressSigs.Set(0) // reseting the gauge.

		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		dgst := party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		fmt.Println("engines started, requesting sigs")

		m := dto.Metric{}
		inProgressSigs.Write(&m)
		a.Equal(0, int(m.Gauge.GetValue()))

		// all engines are started, now we can begin the protocol.
		for _, engine := range engines {
			tmp := make([]byte, 32)
			copy(tmp, dgst[:])
			engine.BeginAsyncThresholdSigningProtocol(tmp, 0)
		}

		inProgressSigs.Write(&m)
		a.Equal(engines[0].Threshold+1, int(m.Gauge.GetValue()))

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}

		time.Sleep(time.Millisecond * 500) // ensuring all other engines have finished and not just one of them.
		inProgressSigs.Write(&m)
		a.Equal(0, int(m.Gauge.GetValue())) // ensuring nothing is in progress.

		sigProducedCntr.Write(&m)
		a.Equal(engines[0].Threshold+1, int(m.Counter.GetValue()))

		sentMsgCntr.Write(&m)
		committeeSize := engines[0].Threshold + 1
		numBroadcastRounds := 8
		numUnicastRounds := 2
		numUnicastSendRequestsPerGuardian := engines[0].Threshold * numUnicastRounds
		a.Equal(committeeSize*(numBroadcastRounds+numUnicastSendRequestsPerGuardian), int(m.Counter.GetValue()))

		receivedMsgCntr.Write(&m)
		// n^2 * (numBroadcastRounds + numUnicastRounds)
		a.Greater(int(m.Counter.GetValue()), committeeSize*committeeSize*(numBroadcastRounds+numUnicastRounds))

		deliveredMsgCntr.Write(&m)
		// messages from committeeSize are delivered numBroadcastRounds times by each guardian.
		a.Equal(committeeSize*numBroadcastRounds*len(engines), int(m.Counter.GetValue()))
	})

	t.Run("with 5 sigs", func(t *testing.T) {
		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		digests := make([]party.Digest, 5)
		for i := 0; i < 5; i++ {
			digests[i] = party.Digest{byte(i)}
		}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, len(digests))

		fmt.Println("engines started, requesting sigs")

		// all engines are started, now we can begin the protocol.
		for _, d := range digests {

			for _, engine := range engines {
				tmp := make([]byte, 32)
				copy(tmp, d[:])

				engine.BeginAsyncThresholdSigningProtocol(tmp, 1)
			}
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})
}

func ctxExpiredFirst(ctx context.Context, ch chan struct{}) bool {
	select {
	case <-ctx.Done():
		return true
	case <-ch:
		return false
	}
}

// func TestFTLoop(t *testing.T) {
// 	for i := 0; i < 5; i++ {
// 		t.Run("looping", TestFT)
// 	}

// }

func TestFT(t *testing.T) {

	t.Run("avoid report problem if in config", func(t *testing.T) {
		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		n := 1
		chainId := vaa.ChainID(1)
		digests := make([]party.SigningTask, n)
		for i := 0; i < n; i++ {
			digests[i] = party.SigningTask{
				Digest:       [32]byte{byte(i)},
				Faulties:     nil,
				AuxilaryData: chainIDToBytes(chainId),
			}
		}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute/6)
		defer cancel()

		fmt.Println("starting engines.")
		for _, engine := range engines {
			engine.Configurations.maxJitter = time.Nanosecond
			engine.Configurations.DelayGraceTime = time.Second
			engine.Configurations.ChainsWithNoSelfReport = append(engine.Configurations.ChainsWithNoSelfReport, uint16(chainId))
			a.NoError(engine.Start(ctx))
		}

		e := getSigningGuardian(a, engines, digests...)
		a.NotNil(e)

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, len(digests)) // ensure we wait

		for _, d := range digests {
			d := d

			for _, engine := range engines {
				// someone who is needed to sign will not join here, and will not let anyone replace it either.
				if equalPartyIds(e.Self, engine.Self) {
					continue
				}
				engine.BeginAsyncThresholdSigningProtocol(d.Digest[:], chainId)
			}
		}

		if !ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("Should've expired first")
		}
	})
	t.Run("multiple-callls-in-parallel", func(t *testing.T) {
		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		n := 100
		digests := make([]party.Digest, n)
		for i := range n {
			digests[i] = party.Digest{byte(i)}
		}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*8)
		defer cancel()

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, len(digests))

		fmt.Println("engines started, requesting sigs")

		wg := sync.WaitGroup{}
		barrier := make(chan struct{})
		// all engines are started, now we can begin the protocol.
		for _, d := range digests {
			for _, engine := range engines {
				engine := engine
				d := d

				wg.Add(1)
				go func() {
					defer wg.Done()
					<-barrier
					tmp := make([]byte, 32)
					copy(tmp, d[:])

					engine.BeginAsyncThresholdSigningProtocol(tmp, 1)
				}()
			}
		}

		time.Sleep(time.Millisecond * 500)
		close(barrier)

		time.Sleep(time.Millisecond * 500)
		engines[0].reportProblem(1)
		time.Sleep(time.Millisecond * 500)
		engines[1].reportProblem(1)
		wg.Wait()
		fmt.Println("=========Done with all goroutines=========")

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})
	t.Run("single server crashes", func(t *testing.T) {
		a := assert.New(t)

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		fmt.Println("starting engines.")
		for _, engine := range engines {
			engine.GuardianStorage.DelayGraceTime = time.Second * 3
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		fmt.Println("engines started, requesting sigs")

		cID := vaa.ChainID(1)
		singingTask := party.SigningTask{
			Digest:       party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9},
			Faulties:     []*tss.PartyID{},
			AuxilaryData: chainIDToBytes(cID),
		}
		e := getSigningGuardian(a, engines, singingTask)

		enginesWithoutE := make([]*Engine, 0, len(engines)-1)
		eSelf := partyIdToString(e.Self)
		for i := range engines {
			if partyIdToString(engines[i].Self) == eSelf {
				continue
			}

			enginesWithoutE = append(enginesWithoutE, engines[i])
		}

		// all engines are started, now we can begin the protocol.
		for _, engine := range enginesWithoutE {
			tmp := make([]byte, len(singingTask.Digest))
			copy(tmp, singingTask.Digest[:])

			engine.BeginAsyncThresholdSigningProtocol(tmp, cID)
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})

	t.Run("down server returns during overlap time and signs with original committee", func(t *testing.T) {
		a := assert.New(t)
		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		cID := vaa.ChainID(1)
		tsk := party.SigningTask{
			Digest:       party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9},
			Faulties:     []*tss.PartyID{},
			AuxilaryData: chainIDToBytes(cID),
		}

		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		signers := getSigningGuardians(a, engines, tsk)

		fmt.Println("starting engines.")
		for _, engine := range signers { // start only original committee!
			// should wake a little before the synchronsingInterval.
			engine.GuardianStorage.Configurations.guardianDownTime = synchronsingInterval
			engine.GuardianStorage.maxJitter = time.Microsecond
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		fmt.Println("engines started, requesting sigs")

		signers[0].reportProblem(cID) // using chainid==0.

		time.Sleep(synchronsingInterval / 2)

		// Only engines from original comittee are allowed to sign.
		for _, engine := range signers {
			tmp := make([]byte, len(tsk.Digest))
			copy(tmp, tsk.Digest[:])

			engine.BeginAsyncThresholdSigningProtocol(tmp, cID)
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})

	t.Run("down server returns and signs on original committee different return times", func(t *testing.T) {
		// changing the guardianDownTime parameter with different value for each guardian
		// let us simulate a situation where each guardian received the "problem" message at a different time.
		//
		// one of the guardian revival time will be so long that it'll have to restart the guardian using
		// a timer it set up, and not due to the overlapping interval.
		a := assert.New(t)
		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		cID := vaa.ChainID(1)
		signingTask := party.SigningTask{
			Digest:       party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9},
			AuxilaryData: chainIDToBytes(cID),
		}

		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		signers := getSigningGuardians(a, engines, signingTask)
		a.Len(signers, 3)

		fmt.Println("starting engines.")
		// start only original committee!
		for i, engine := range signers {
			// set each guardian with a different downtime.
			// ensure the protocol generates a signature.
			engine.GuardianStorage.Configurations.guardianDownTime = time.Second * 4 * time.Duration(i+1)
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		fmt.Println("engines started, requesting sigs")

		signers[0].reportProblem(0) // using chainid==0.

		time.Sleep(synchronsingInterval + time.Second)

		// Only engines from original comittee are allowed to sign.
		for _, engine := range signers {
			tmp := make([]byte, len(signingTask.Digest))
			copy(tmp, signingTask.Digest[:])

			engine.BeginAsyncThresholdSigningProtocol(tmp, cID)
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})

	t.Run("server crashes during signing multiple digests", func(t *testing.T) {
		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		n := 3
		chainId := vaa.ChainID(1)
		digests := make([]party.SigningTask, n)
		for i := 0; i < n; i++ {
			digests[i] = party.SigningTask{
				Digest:       [32]byte{byte(i)},
				Faulties:     nil,
				AuxilaryData: chainIDToBytes(chainId),
			}
		}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute/4)
		defer cancel()

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		e := getSigningGuardian(a, engines, digests...)
		a.NotNil(e)

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, len(digests))

		fmt.Println("engines started, requesting sigs")

		go func() {
			time.Sleep(time.Second / 4) // enough time for the engines to start the signing protocol.
			e.reportProblem(chainId)    // telling the server to report to everyone it has an issue.
			fmt.Printf("========\n %v Issued problem now!\n=======\n=", e.Self.Id)
		}()

		for _, d := range digests {
			d := d

			for _, engine := range engines {
				engine.BeginAsyncThresholdSigningProtocol(d.Digest[:], chainId)
			}
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})

	t.Run("cant recover after f faults", func(t *testing.T) {
		a := assert.New(t)

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Second*20)
		defer cancel()

		cid := vaa.ChainID(0)
		dgst := party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9}

		// 7 guardians, 3 faults, need 5 signers.
		engines, err := loadGuardians(7, "tss7")
		a.NoError(err)

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}
		f := engines[0].getMaxExpectedFaults()

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		fmt.Println("engines started, requesting sigs")

		for i := 0; i < f+1; i++ {
			engines[i].reportProblem(cid) // TODO: need to happen on a specific chain.
		}

		time.Sleep(time.Second * 2) // waiting for the f issues to be reported.

		// letting the other engines run
		for i := f + 1; i < len(engines); i++ {
			engine := engines[i]

			tmp := make([]byte, 32)
			copy(tmp, dgst[:])

			engine.BeginAsyncThresholdSigningProtocol(tmp, cid)
		}

		// expecting the time to run out.
		if !ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("received sig", "received sig")
		}
	})

	t.Run("3 sig f faults", func(t *testing.T) {
		a := assert.New(t)

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		engines, err := loadGuardians(7, "tss7")
		a.NoError(err)

		a.Len(engines, 7)

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		fmt.Println("engines started, requesting sigs")

		cID := vaa.ChainID(1)
		tsks := make([]party.SigningTask, 3)
		for i := range tsks {
			tsks[i] = party.SigningTask{
				Digest:       party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9},
				Faulties:     []*tss.PartyID{},
				AuxilaryData: chainIDToBytes(cID),
			}
		}
		signers := getSigningGuardians(a, engines, tsks...)
		a.GreaterOrEqual(len(signers), 3)

		removed := signers[:engines[0].getMaxExpectedFaults()]
		for _, r := range removed {
			r.DelayGraceTime /= 2 // reducing test time.
		}

		for _, engine := range engines {

			// skipping the engines that are removed.
			if contains(removed, engine) {
				continue
			}

			for _, tsk := range tsks {
				tmp := make([]byte, len(tsk.Digest))
				copy(tmp, tsk.Digest[:])

				engine.BeginAsyncThresholdSigningProtocol(tmp, cID)
			}
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})

	t.Run("recover given a missing heartbeat", func(t *testing.T) {
		t.Skip()
	})

	t.Run("server crashes on a single chain, shouldn't affect signatures on other chain", func(t *testing.T) {
		a := assert.New(t)

		ctx, cancel := context.WithTimeout(context.Background(), time.Minute*1)
		defer cancel()

		ctx = testutils.MakeSupervisorContext(ctx)

		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")

		fmt.Println("engines started, requesting sigs")

		tsks := make([]party.SigningTask, 2)
		for i := range tsks {
			tsks = append(tsks, party.SigningTask{
				Digest:       party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9},
				Faulties:     []*tss.PartyID{},
				AuxilaryData: chainIDToBytes(vaa.ChainID(i)),
			})
		}

		e := getSigningGuardian(a, engines, tsks...)
		a.NotNil(e) // duo to quorum size of 3 out of 5 there must be one guardian that is needed for both tasks.

		dnchn := msgHandler(ctx, engines, len(tsks)) // expecting 2 messages.
		e.reportProblem(0)                           // on the chain of the first task only.

		// all engines are started, now we can begin the protocol.
		for i := 0; i < len(tsks); i++ {
			for _, engine := range engines {
				dgst := party.Digest{}
				copy(dgst[:], tsks[i].Digest[:])

				engine.BeginAsyncThresholdSigningProtocol(dgst[:], vaa.ChainID(i))
			}
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})

	t.Run("2 server delayed on one chain rejoin signing after their downtime ends", func(t *testing.T) {
		a := assert.New(t)
		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		cID := vaa.ChainID(1)
		tsk := party.SigningTask{
			Digest:       party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9},
			Faulties:     []*tss.PartyID{},
			AuxilaryData: chainIDToBytes(cID),
		}

		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		signers := getSigningGuardians(a, engines, tsk)

		fmt.Println("starting engines.")
		for _, engine := range signers {
			engine.GuardianStorage.Configurations.guardianDownTime = synchronsingInterval
			engine.GuardianStorage.maxJitter = time.Microsecond
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		fmt.Println("engines started, requesting sigs")

		signers[0].reportProblem(cID)
		signers[1].reportProblem(cID)

		time.Sleep(synchronsingInterval / 2)

		// Only engines from original comittee are allowed to sign.

		// the signing guardian should use a committee with the other guardians (which we haven't started on purpose),
		// since it received by now the problem message. (This is mainly to ensure: the guardian WILL allow signing this message)
		signers[2].BeginAsyncThresholdSigningProtocol(tsk.Digest[:], cID)

		time.Sleep(signers[0].guardianDownTime + time.Second) // waiting for the downtime to end.

		fmt.Println("###rejoining###")
		for i := 0; i < 2; i++ { // letting them sign again.
			signers[i].BeginAsyncThresholdSigningProtocol(tsk.Digest[:], cID)
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNowf("context expired", "context expired")
		}
	})

}

func TestMessagesWithBadRounds(t *testing.T) {
	a := assert.New(t)
	gs := load5GuardiansSetupForBroadcastChecks(a)
	e1, e2 := gs[0], gs[1]
	from := e1.Self
	to := e2.Self

	t.Run("Unicast", func(t *testing.T) {
		msgDigest := party.Digest{1}
		for _, rnd := range broadcastRounds {
			parsed := generateFakeMessageWithRandomContent(from, to, rnd, msgDigest)
			bts, _, err := parsed.WireBytes()
			a.NoError(err)

			m := &IncomingMessage{
				Source: partyIdToProto(from),
				Content: &tsscommv1.PropagatedMessage{Message: &tsscommv1.PropagatedMessage_Unicast{
					Unicast: &tsscommv1.Unicast{
						Content: &tsscommv1.TssContent{Payload: bts},
					},
				}},
			}
			err = e2.handleUnicast(m)
			a.ErrorIs(err, errUnicastBadRound)
		}
	})

	t.Run("Echo", func(t *testing.T) {
		msgDigest := party.Digest{2}
		for _, rnd := range unicastRounds {
			parsed := generateFakeMessageWithRandomContent(from, to, rnd, msgDigest)
			bts, _, err := parsed.WireBytes()
			a.NoError(err)

			m := &IncomingMessage{
				Source: partyIdToProto(from),
				Content: &tsscommv1.PropagatedMessage{Message: &tsscommv1.PropagatedMessage_Echo{
					Echo: &tsscommv1.Echo{
						Message: &tsscommv1.SignedMessage{
							Content: &tsscommv1.SignedMessage_TssContent{
								TssContent: &tsscommv1.TssContent{Payload: bts},
							},
							Sender:    partyIdToProto(from),
							Signature: nil,
						},
					},
				}},
			}
			a.NoError(e1.sign(m.Content.GetEcho().Message))

			_, err = e2.handleEcho(m)
			a.ErrorIs(err, errBadRoundsInEcho)
		}
	})
}

func generateFakeParsedMessageWithRandomContent(from, to *tss.PartyID, rnd signingRound, digest party.Digest) processedMessage {
	fake := generateFakeMessageWithRandomContent(from, to, rnd, digest)
	return &parsedTssContent{fake, ""}
}

// if to == nil it's a broadcast message.
func generateFakeMessageWithRandomContent(from, to *tss.PartyID, rnd signingRound, digest party.Digest) tss.ParsedMessage {
	partiesState := make([]byte, maxParties)
	for i := 0; i < maxParties; i++ {
		partiesState[i] = 255
	}

	trackingId := &tsscommon.TrackingID{
		Digest:       digest[:],
		PartiesState: partiesState,
		AuxilaryData: []byte{},
	}

	rndmBigNumber := &big.Int{}
	buf := make([]byte, 16)
	rand.Read(buf)
	rndmBigNumber.SetBytes(buf)

	var (
		meta    = tss.MessageRouting{From: from, IsBroadcast: true}
		content tss.MessageContent
	)

	switch rnd {
	case round1Message1:
		if to == nil {
			panic("not a broadcast message")
		}
		meta = tss.MessageRouting{From: from, To: []*tss.PartyID{to}, IsBroadcast: false}
		content = &signing.SignRound1Message1{C: rndmBigNumber.Bytes()}
	case round1Message2:
		content = &signing.SignRound1Message2{Commitment: rndmBigNumber.Bytes()}
	case round2Message:
		if to == nil {
			panic("not a broadcast message")
		}
		meta = tss.MessageRouting{From: from, To: []*tss.PartyID{to}, IsBroadcast: false}
		content = &signing.SignRound2Message{C1: rndmBigNumber.Bytes()}
	case round3Message:
		content = &signing.SignRound3Message{Theta: rndmBigNumber.Bytes()}
	case round4Message:
		content = &signing.SignRound4Message{ProofAlphaX: rndmBigNumber.Bytes()}
	case round5Message:
		content = &signing.SignRound5Message{Commitment: rndmBigNumber.Bytes()}
	case round6Message:
		content = &signing.SignRound6Message{ProofAlphaX: rndmBigNumber.Bytes()}
	case round7Message:
		content = &signing.SignRound7Message{Commitment: rndmBigNumber.Bytes()}
	case round8Message:
		content = &signing.SignRound8Message{DeCommitment: [][]byte{rndmBigNumber.Bytes()}}
	case round9Message:
		content = &signing.SignRound9Message{S: rndmBigNumber.Bytes()}
	default:
		panic("unknown round")
	}

	return tss.NewMessage(meta, content, tss.NewMessageWrapper(meta, content, trackingId))
}

func loadMockGuardianStorage(gstorageIndex int, from string) *GuardianStorage {
	path, err := testutils.GetMockGuardianTssStorage(gstorageIndex, from)
	if err != nil {
		panic(err)
	}

	st, err := NewGuardianStorageFromFile(path)
	if err != nil {
		panic(err)
	}
	return st
}

func loadGuardians(numParticipants int, from string) ([]*Engine, error) {
	engines := make([]*Engine, numParticipants)

	for i := 0; i < numParticipants; i++ {
		e, err := NewReliableTSS(loadMockGuardianStorage(i, from))
		if err != nil {
			return nil, err
		}
		en, ok := e.(*Engine)
		if !ok {
			return nil, errors.New("not an engine")
		}
		engines[i] = en
	}

	return engines, nil
}

type msgg struct {
	Sender *tsscommv1.PartyId
	Sendable
}

func msgHandler(ctx context.Context, engines []*Engine, numDiffSigsExpected int) chan struct{} {
	signalSuccess := make(chan struct{})
	once := sync.Once{}

	nmsigs := map[string]struct{}{}
	lck := sync.Mutex{}

	go func() {
		wg := sync.WaitGroup{}
		wg.Add(len(engines) * 2)

		chns := make(map[string]chan msgg, len(engines))
		for _, en := range engines {
			chns[en.Self.Id] = make(chan msgg, 10000)
		}

		for _, e := range engines {
			engine := e

			// need a separate goroutine for handling engine output and engine input.
			// simulating network stream incoming and network stream outgoing.

			// incoming
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return

					case msg := <-chns[engine.Self.Id]:
						engine.HandleIncomingTssMessage(&IncomingMessage{
							Source:  msg.Sender,
							Content: msg.GetNetworkMessage(),
						})
					}
				}
			}()

			//  Listener, responsible to receive output of engine, and direct it to the other engines.
			go func() {
				defer wg.Done()
				for {
					select {
					case <-ctx.Done():
						return

					case m := <-engine.ProducedOutputMessages():
						if m.IsBroadcast() {
							broadcast(chns, engine, m)
							continue
						}
						unicast(m, chns, engine)
					case sig := <-engine.ProducedSignature():
						signature := append(sig.Signature, sig.SignatureRecovery...)
						address := engine.GetEthAddress()

						pubKey, err := crypto.Ecrecover(sig.M, signature)
						if err != nil {
							panic("failed to do ecrecover:" + err.Error())
						}
						addr := common.BytesToAddress(crypto.Keccak256(pubKey[1:])[12:])

						// check that the recovered address equals the provided address
						if addr != address {
							panic("recovered address does not match provided address")
						}

						lck.Lock()
						nmsigs[sig.TrackingId.ToString()] = struct{}{}
						ln := len(nmsigs)
						lck.Unlock()

						fmt.Println("received signature", ln)
						if ln < numDiffSigsExpected {
							continue
						}

						fmt.Printf("/////////\nreceived all signatures (%v)\n/////////\n", numDiffSigsExpected)
						once.Do(func() {
							close(signalSuccess)
						})
					}
				}
			}()
		}

		wg.Wait()
	}()

	return signalSuccess
}

func unicast(m Sendable, chns map[string]chan msgg, engine *Engine) {
	pids := m.GetDestinations()
	for _, pid := range pids {
		feedChn := chns[pid.Id]
		feedChn <- msgg{
			Sender:   partyIdToProto(engine.Self),
			Sendable: m.cloneSelf(),
		}
	}
}

func broadcast(chns map[string]chan msgg, engine *Engine, m Sendable) {
	for _, feedChn := range chns {
		feedChn <- msgg{
			Sender:   partyIdToProto(engine.Self),
			Sendable: m.cloneSelf(),
		}
	}
}

// strictly for the tests.
func (c *activeSigCounter) digestToGuardiansLen() int {
	c.mtx.RLock()
	defer c.mtx.RUnlock()

	return len(c.digestToGuardians)
}

func TestSigCounter(t *testing.T) {
	a := assert.New(t)

	supctx := testutils.MakeSupervisorContext(context.Background())
	ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
	defer cancel()

	t.Run("MaxCountBlockAdditionalUpdates", func(t *testing.T) {
		// Tests might fail due to change of the GuardianStorage files
		cID := vaa.ChainID(0)
		tsks := []party.SigningTask{
			party.SigningTask{party.Digest{1}, []*tss.PartyID{}, chainIDToBytes(cID)},
			party.SigningTask{party.Digest{2}, []*tss.PartyID{}, chainIDToBytes(cID)},
		}
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1 := getSigningGuardian(a, engines, tsks...)

		e1.maxSimultaneousSignatures = 1
		e1.Start(ctx)

		msg := beginSigningAndGrabMessage(e1, tsks[0].Digest[:], cID)

		a.NoError(e1.handleIncomingTssMessage(&IncomingMessage{
			Source:  partyIdToProto(e1.Self),
			Content: msg.GetNetworkMessage(),
		}))

		// trying to handle a new message for a different signature.
		msg = beginSigningAndGrabMessage(e1, tsks[1].Digest[:], cID)

		a.ErrorContains(e1.handleIncomingTssMessage(&IncomingMessage{
			Source:  partyIdToProto(e1.Self),
			Content: msg.GetNetworkMessage(),
		}), "reached the maximum number of simultaneous signatures")
	})

	t.Run("ErrorReduceCount", func(t *testing.T) {
		// Tests might fail due to change of the GuardianStorage files
		cID := vaa.ChainID(0)
		tsks := []party.SigningTask{
			party.SigningTask{party.Digest{1}, []*tss.PartyID{}, chainIDToBytes(cID)},
		}
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1 := getSigningGuardian(a, engines, tsks...)
		e1.maxSimultaneousSignatures = 1

		e1.Start(ctx)

		msg := beginSigningAndGrabMessage(e1, tsks[0].Digest[:], cID)

		incoming := &IncomingMessage{
			Source:  partyIdToProto(e1.Self),
			Content: msg.GetNetworkMessage(),
		}

		a.NoError(e1.handleIncomingTssMessage(incoming))

		parsed, err := e1.parseUnicast(incoming)
		a.NoError(err)

		// test:
		a.Equal(e1.sigCounter.digestToGuardiansLen(), 1)
		select {
		case e1.fpErrChannel <- tss.NewTrackableError(fmt.Errorf("dummyerr"), "de", -1, e1.Self, parsed.getTrackingID()):
		case <-time.After(time.Second * 1):
			t.FailNow()
			return
		}

		time.Sleep(time.Millisecond * 500)

		a.Equal(e1.sigCounter.digestToGuardiansLen(), 0)
	})

	t.Run("sigDoneReduceCount", func(t *testing.T) {
		// Tests might fail due to change of the GuardianStorage files
		cID := vaa.ChainID(0)
		tsks := []party.SigningTask{
			party.SigningTask{party.Digest{1}, []*tss.PartyID{}, chainIDToBytes(cID)},
		}
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1 := getSigningGuardian(a, engines, tsks...)
		e1.maxSimultaneousSignatures = 1

		e1.Start(ctx)

		msg := beginSigningAndGrabMessage(e1, tsks[0].Digest[:], cID)

		incoming := &IncomingMessage{
			Source:  partyIdToProto(e1.Self),
			Content: msg.GetNetworkMessage(),
		}

		a.NoError(e1.handleIncomingTssMessage(incoming))

		parsed, err := e1.parseUnicast(incoming)
		a.NoError(err)

		// test:
		a.Equal(e1.sigCounter.digestToGuardiansLen(), 1)
		e1.fpSigOutChan <- &tsscommon.SignatureData{
			Signature:         []byte{},
			SignatureRecovery: []byte{},
			R:                 []byte{},
			S:                 []byte{},
			M:                 []byte{},
			TrackingId:        parsed.getTrackingID(),
		}
		time.Sleep(time.Millisecond * 500)
		a.Equal(e1.sigCounter.digestToGuardiansLen(), 0)
	})

	t.Run("CanHaveSimulSigners", func(t *testing.T) {
		cID := vaa.ChainID(0)
		tsks := []party.SigningTask{
			party.SigningTask{party.Digest{1}, []*tss.PartyID{}, chainIDToBytes(cID)},
			party.SigningTask{party.Digest{2}, []*tss.PartyID{}, chainIDToBytes(cID)},
		}

		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1 := getSigningGuardian(a, engines, tsks...)
		e1.maxSimultaneousSignatures = 2

		e1.Start(ctx)

		msg := beginSigningAndGrabMessage(e1, tsks[0].Digest[:], cID)

		a.NoError(e1.handleIncomingTssMessage(&IncomingMessage{
			Source:  partyIdToProto(e1.Self),
			Content: msg.GetNetworkMessage(),
		}))

		a.NoError(e1.handleIncomingTssMessage(&IncomingMessage{
			Source:  partyIdToProto(e1.Self),
			Content: beginSigningAndGrabMessage(e1, tsks[1].Digest[:], cID).GetNetworkMessage(),
		}))

	})
}
func getSigningGuardian(a *assert.Assertions, engines []*Engine, tsks ...party.SigningTask) *Engine {
	return getSigningGuardians(a, engines, tsks...)[0]
}

func getSigningGuardians(a *assert.Assertions, engines []*Engine, tsks ...party.SigningTask) []*Engine {
	a.GreaterOrEqual(len(tsks), 1) // at least one

	guardians := make([]*Engine, 0, len(engines))
mainloop:
	for _, e := range engines {

		for _, tsk := range tsks {
			info1, err := e.fp.GetSigningInfo(tsk)
			a.NoError(err)

			if !info1.IsSigner {
				continue mainloop
			}
		}

		guardians = append(guardians, e)
	}

	return guardians
}

func beginSigningAndGrabMessage(e1 *Engine, dgst []byte, cid vaa.ChainID) Sendable {
	go e1.BeginAsyncThresholdSigningProtocol(dgst, cid)

	var msg Sendable
	for i := 0; i < round1NumberOfMessages(e1); i++ { // cleaning the channel, and taking one of the messages.
		select {
		case tmp := <-e1.ProducedOutputMessages():
			if !tmp.IsBroadcast() {
				msg = tmp
			}

		case <-time.After(time.Second * 5):
			// This means the signer wasn't one of the signing committees. (did the Guardian storage change?)
			// if it did, just make sure this engine is expected to sign, else use the right engine in the test.
			panic("timeout!")
		}
	}
	return msg
}

func round1NumberOfMessages(e1 *Engine) int {
	// although threshold is non-inclusive, we only send e1.Threshold since one doesn't includes itself in the unicasts.
	// the +1 is for the additional broadcast message.
	return e1.Threshold + 1
}

func contains(lst []*Engine, e *Engine) bool {
	for _, l := range lst {
		if l.Self.Id == e.Self.Id {
			return true
		}
	}

	return false
}

func TestTrackingIDSizeIsOkay(t *testing.T) {
	dgst := party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9}
	tid := tsscommon.TrackingID{
		Digest:       dgst[:],
		PartiesState: make([]byte, (maxParties+7)/8),
		AuxilaryData: chainIDToBytes(vaa.ChainID(5)),
	}

	tidstr := tid.ToString()
	assert.Equal(t, len(tidstr), trackingIDHexStrSize)
}
