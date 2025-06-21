package tss

import (
	"bytes"
	"context"
	crand "crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	whcommon "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/internal/testutils"
	tsscommv1 "github.com/certusone/wormhole/node/pkg/proto/tsscomm/v1"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"github.com/xlabs/multi-party-sig/protocols/frost"
	"github.com/xlabs/multi-party-sig/protocols/frost/sign"
	common "github.com/xlabs/tss-common"
	tsscommon "github.com/xlabs/tss-common"
	"github.com/xlabs/tss-lib/v2/party"
	"google.golang.org/protobuf/proto"
)

var (
	unicastRounds   = []signingRound{}
	broadcastRounds = []signingRound{
		round2Message,
		round3Message,
	}

	allRounds                     = append(unicastRounds, broadcastRounds...)
	reportableConsistancyLevel    = uint8(1)                // TODO
	nonReportableConsistancyLevel = instantConsistencyLevel // TODO
)

func parsedIntoEcho(a *assert.Assertions, t *Engine, parsed common.ParsedMessage) *IncomingMessage {
	payload, _, err := parsed.WireBytes()
	a.NoError(err)

	msg := &tsscommv1.Echo{
		Message: &tsscommv1.SignedMessage{
			Content: &tsscommv1.SignedMessage_TssContent{
				TssContent: &tsscommv1.TssContent{Payload: payload},
			},
			Sender:    uint32(t.Self.CommunicationIndex),
			Signature: nil,
		},
	}

	tmp := serializeableMessage{&tssMessageWrapper{parsed}}

	a.NoError(t.sign(tmp.getUUID(t.LoadDistributionKey), msg.Message))

	return &IncomingMessage{
		Source: t.Self,
		Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{
				Echo: msg,
			},
		},
	}
}

func (i *IncomingMessage) setSource(id *Identity) {
	i.Source = id
}

func TestBroadcast(t *testing.T) {

	// The tests here rely on n=5, threshold=2, meaning 3 guardians are needed to sign (f<=1).
	t.Run("forLeaderCreatingMessage", func(t *testing.T) {
		a := assert.New(t)
		// f = 1, n = 5
		engines := load5GuardiansSetupForBroadcastChecks(a)
		receiver := engines[4]

		e1 := engines[0]
		// make parsedMessage, and insert into e1
		// then add another one for the same round.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rnd, party.Digest{byte(j)})

			echo := parsedIntoEcho(a, e1, parsed1)

			shouldBroadcast, shouldDeliver, err := receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.NoError(err)
			a.True(shouldBroadcast)
			a.Nil(shouldDeliver)
		}
	})

	t.Run("forLeaderNotReBroadcasting", func(t *testing.T) {
		a := assert.New(t)
		// f = 1, n = 5
		engines := load5GuardiansSetupForBroadcastChecks(a)

		e1 := engines[0]
		receiver := e1
		// make parsedMessage, and insert into e1
		// then add another one for the same round.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rnd, party.Digest{byte(j)})

			echo := parsedIntoEcho(a, e1, parsed1)

			shouldBroadcast, shouldDeliver, err := receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(shouldDeliver)
		}
	})

	t.Run("OnlyOnce", func(t *testing.T) {
		a := assert.New(t)
		// f = 1, n = 5
		engines := load5GuardiansSetupForBroadcastChecks(a)
		receiver := engines[4]

		e1 := engines[0]
		// make parsedMessage, and insert into e1
		// then add another one for the same round.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rnd, party.Digest{byte(j)})

			echo := parsedIntoEcho(a, e1, parsed1)

			shouldBroadcast, shouldDeliver, err := receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.NoError(err)
			a.True(shouldBroadcast)
			a.Nil(shouldDeliver)

			shouldBroadcast, shouldDeliver, err = receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(shouldDeliver)

			shouldBroadcast, shouldDeliver, err = receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(shouldDeliver)
		}
	})

	t.Run("waitForActualValueFromLeader", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2, e3 := engines[0], engines[1], engines[2]

		receiver := engines[4]
		// two different signers on an echo, meaning it will receive from two players.
		// since f=1 and we have f+1 echos: it should broadcast at the end of this test.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rnd, party.Digest{byte(j)})

			originalValue := parsedIntoEcho(a, e1, parsed1)

			echo := makeHashEcho(e1, parsed1, originalValue)

			parsed := &parsedHashEcho{
				HashEcho: echo.toBroadcastMsg().Message.GetHashEcho(),
			}

			echo.setSource(e2.Self)

			shouldBroadcast, deliverable, err := receiver.broadcastInspection(parsed, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(deliverable)

			echo.setSource(e3.Self)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(parsed, echo)
			a.NoError(err)
			a.False(shouldBroadcast) // should broadcast only for leader.
			a.Nil(deliverable)

			echo.setSource(e1.Self)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(parsed, echo)
			a.NoError(err)
			a.False(shouldBroadcast) // should not broadcast if it hadn't seen the actual value from the leader!
			a.Nil(deliverable)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, originalValue)
			a.NoError(err)
			a.True(shouldBroadcast) // should echo when seeing the actual value from the leader.
			a.NotNil(deliverable)
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

func makeHashEcho(e *Engine, parsed common.ParsedMessage, in *IncomingMessage) *IncomingMessage {
	echocpy := proto.Clone(in.toBroadcastMsg()).(*tsscommv1.Echo)

	outgoing := &IncomingMessage{
		Source: in.Source,
		Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{
				Echo: echocpy,
			},
		}}

	tmp := serializeableMessage{&tssMessageWrapper{parsed}}

	uid := tmp.getUUID(e.LoadDistributionKey)
	dgst := hashSignedMessage(echocpy.Message)

	hshEcho := &tsscommv1.HashEcho{
		SessionUuid:          uid[:],
		OriginalContetDigest: dgst[:],
	}

	outgoing.toBroadcastMsg().Message.Content = &tsscommv1.SignedMessage_HashEcho{hshEcho}
	return outgoing

}
func TestDeliver(t *testing.T) {
	t.Run("After2fPlus1Messages", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2, e3 := engines[0], engines[1], engines[2]

		receiver := engines[4]
		// two different signers on an echo, meaning it will receive from two players.
		// since f=1 and we have f+1 echos: it should broadcast at the end of this test.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rnd, party.Digest{byte(j)})

			echo := parsedIntoEcho(a, e1, parsed1)
			hshEcho := makeHashEcho(e1, parsed1, echo)
			hshEcho.setSource(e2.Self)

			prsedHashEcho := &parsedHashEcho{hshEcho.toBroadcastMsg().Message.GetHashEcho()}
			shouldBroadcast, deliverable, err := receiver.broadcastInspection(prsedHashEcho, hshEcho)

			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(deliverable)

			hshEcho.setSource(e3.Self)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(prsedHashEcho, hshEcho)
			a.NoError(err)
			a.False(shouldBroadcast) // haven't seen the actual value from the leader yet.
			a.Nil(deliverable)

			echo.setSource(e1.Self)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.NoError(err)
			a.True(shouldBroadcast)
			a.NotNil(deliverable)
		}
	})

	t.Run("doesn'tDeliverTwice", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2, e3, e4 := engines[0], engines[1], engines[2], engines[3]

		receiver := engines[4]
		// two different signers on an echo, meaning it will receive from two players.
		// since f=1 and we have f+1 echos: it should broadcast at the end of this test.
		for j, rnd := range allRounds {
			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rnd, party.Digest{byte(j)})
			echo := parsedIntoEcho(a, e1, parsed1)
			hashecho := makeHashEcho(e1, parsed1, echo)
			hashecho.setSource(e2.Self)

			prsedHashEcho := &parsedHashEcho{hashecho.toBroadcastMsg().Message.GetHashEcho()}
			shouldBroadcast, deliverable, err := receiver.broadcastInspection(prsedHashEcho, hashecho)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(deliverable)

			hashecho.setSource(e3.Self)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(prsedHashEcho, hashecho)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(deliverable)

			echo.setSource(e1.Self)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.NoError(err)
			a.True(shouldBroadcast)
			a.NotNil(deliverable)

			// twice in a row
			shouldBroadcast, deliverable, err = receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(deliverable)

			// new hash echo, shouldn't deliver again too.
			hashecho.setSource(e4.Self)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(prsedHashEcho, hashecho)
			a.NoError(err)
			a.False(shouldBroadcast)
			a.Nil(deliverable)
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
		parsed1 := generateFakeParsedMessageWithRandomContent(e1.Self.Pid, e1.Self.Pid, rnd, trackingId)
		parsed2 := generateFakeParsedMessageWithRandomContent(e1.Self.Pid, e1.Self.Pid, rnd, trackingId)

		uid1 := parsed1.getUUID(e1.LoadDistributionKey)

		uid2 := parsed2.getUUID(e1.LoadDistributionKey)

		a.Equal(uid1, uid2)
	}
}

func TestEquivocation(t *testing.T) {
	t.Run("inBroadcastLogic", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2 := engines[0], engines[1]

		receiver := engines[4]
		for i, rndType := range allRounds {

			trackingId := party.Digest{byte(i)}

			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rndType, trackingId)

			shouldBroadcast, deliverable, err := receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, parsedIntoEcho(a, e2, parsed1))
			a.NoError(err)
			a.True(shouldBroadcast) //should broadcast since e2 is the source of this message.
			a.Nil(deliverable)

			parsed2 := generateFakeMessageWithRandomContent(e1.Self.Pid, e2.Self.Pid, rndType, trackingId)

			shouldBroadcast, deliverable, err = receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed2, ""}}, parsedIntoEcho(a, e2, parsed2))
			a.ErrorContains(err, "equivication")
			a.False(shouldBroadcast)
			a.Nil(deliverable)

			equvicatingEchoerMessage := parsedIntoEcho(a, e2, parsed1)
			equvicatingEchoerMessage.
				Content.
				GetEcho().
				Message.
				Content.(*tsscommv1.SignedMessage_TssContent).
				TssContent.
				Payload[0] += 1
			// now echoer is equivicating (change content, but of some seen message):
			_, _, err = receiver.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, equvicatingEchoerMessage)
			a.ErrorContains(err, e2.Self.Hostname)
		}
	})

	t.Run("inUnicast", func(t *testing.T) {
		a := assert.New(t)
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1, e2 := engines[0], engines[1]

		receiver := engines[4]
		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cncl := context.WithCancel(supctx)
		defer cncl()

		e1.Start(ctx)
		e2.Start(ctx)

		for i, rndType := range unicastRounds {

			trackingId := party.Digest{byte(i)}

			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rndType, trackingId)
			parsed2 := generateFakeMessageWithRandomContent(e1.Self.Pid, receiver.Self.Pid, rndType, trackingId)

			bts, _, err := parsed1.WireBytes()
			a.NoError(err)

			msg := &IncomingMessage{
				Content: &tsscommv1.PropagatedMessage{
					Message: &tsscommv1.PropagatedMessage_Unicast{
						Unicast: &tsscommv1.Unicast{
							Content: &tsscommv1.Unicast_Tss{
								Tss: &tsscommv1.TssContent{
									Payload:         bts,
									MsgSerialNumber: 0,
								},
							},
						},
					},
				},
			}

			msg.setSource(e1.Self)

			receiver.handleUnicast(msg)

			bts, _, err = parsed2.WireBytes()
			a.NoError(err)

			msg.Content.Message.(*tsscommv1.PropagatedMessage_Unicast).
				Unicast.Content.(*tsscommv1.Unicast_Tss).Tss.Payload = bts
			a.ErrorIs(receiver.handleUnicast(msg), ErrEquivicatingGuardian)
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
			parsed1 := generateFakeMessageWithRandomContent(e1.Self.Pid, e1.Self.Pid, rnd, party.Digest{byte(j)})
			echo := parsedIntoEcho(a, e1, parsed1)

			echo.setSource(e1.Self)

			echo.toBroadcastMsg().Message.Signature[0] += 1
			_, _, err := e1.broadcastInspection(&deliverableMessage{&parsedTssContent{parsed1, ""}}, echo)
			a.ErrorIs(err, ErrInvalidSignature)

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

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: e2.Self})
		a.ErrorIs(err, errNeitherBroadcastNorUnicast)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source:  e2.Self,
			Content: &tsscommv1.PropagatedMessage{}})
		a.ErrorIs(err, errNeitherBroadcastNorUnicast)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source: e2.Self,
			Content: &tsscommv1.PropagatedMessage{
				Message: &tsscommv1.PropagatedMessage_Echo{},
			},
		})
		a.ErrorIs(err, ErrBroadcastIsNil)

		err = e1.handleIncomingTssMessage(&IncomingMessage{
			Source: e2.Self,
			Content: &tsscommv1.PropagatedMessage{
				Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{}},
			},
		})
		a.ErrorIs(err, ErrSignedMessageIsNil)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: e2.Self, Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
				Message: &tsscommv1.SignedMessage{
					Sender: uint32(e2.fetchIdentityFromPartyID(e2.Self.Pid).CommunicationIndex),
				},
			}}},
		})
		a.ErrorIs(err, ErrNoContent)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: e2.Self, Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
				Message: &tsscommv1.SignedMessage{
					Content: &tsscommv1.SignedMessage_TssContent{
						TssContent: &tsscommv1.TssContent{},
					},
					Sender:    uint32(e2.fetchIdentityFromPartyID(e2.Self.Pid).CommunicationIndex),
					Signature: []byte{1, 2, 3},
				},
			}}},
		})
		a.ErrorIs(err, ErrNilPayload)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: e2.Self, Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
				Message: &tsscommv1.SignedMessage{
					Content: &tsscommv1.SignedMessage_TssContent{
						TssContent: &tsscommv1.TssContent{
							Payload: []byte{1, 2, 3},
						},
					},
					Sender: uint32(e2.fetchIdentityFromPartyID(e2.Self.Pid).CommunicationIndex),
				},
			}}},
		})
		a.ErrorIs(err, errEmptySignature)

		err = e1.handleIncomingTssMessage(&IncomingMessage{Source: e2.Self, Content: &tsscommv1.PropagatedMessage{
			Message: &tsscommv1.PropagatedMessage_Echo{Echo: &tsscommv1.Echo{
				Message: &tsscommv1.SignedMessage{
					Content: &tsscommv1.SignedMessage_TssContent{
						TssContent: &tsscommv1.TssContent{
							Payload: []byte{1, 2, 3},
						},
					},
					Sender:    uint32(e2.fetchIdentityFromPartyID(e2.Self.Pid).CommunicationIndex),
					Signature: []byte{1, 2, 3},
				},
			}}},
		})
		a.ErrorContains(err, "cannot parse")
	})

	t.Run("Begin signing", func(t *testing.T) {
		var tmp *Engine = nil
		engines2 := load5GuardiansSetupForBroadcastChecks(a)

		a.ErrorIs(tmp.BeginAsyncThresholdSigningProtocol(nil, 0, reportableConsistancyLevel), errNilTssEngine)
		a.ErrorIs(e2.BeginAsyncThresholdSigningProtocol(nil, 0, reportableConsistancyLevel), errTssEngineNotStarted)

		tmp = engines2[1]
		tmp.started.Store(started)

		a.ErrorContains(e1.BeginAsyncThresholdSigningProtocol(make([]byte, 12), 0, reportableConsistancyLevel), "length is not 32 bytes")

		tmp.fp = nil
		a.ErrorContains(tmp.BeginAsyncThresholdSigningProtocol(nil, 0, reportableConsistancyLevel), "not set up correctly")
	})

	t.Run("fetch certificate", func(t *testing.T) {
		_, err := e1.fetchIdentityFromIndex(SenderIndex(e1.GuardianStorage.Guardians.Len() + 1))
		a.ErrorIs(err, ErrUnkownSender)
	})

	t.Run("handle incoming VAAs", func(t *testing.T) {
		a := assert.New(t)

		v, gs := genVaaAndGuardianSet(a)

		gst := whcommon.NewGuardianSetState(nil)
		gst.Set(gs)

		engines := load5GuardiansSetupForBroadcastChecks(a)
		engine := engines[0] // Not starting engine so it doesn't run BeginTSSSign

		engine.SetGuardianSetState(gst)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		ctx = testutils.MakeSupervisorContext(ctx)

		engine.Start(ctx)

		// bad verfication run
		v.Version = 2
		v.Nonce = 0

		bts, err := v.Marshal()
		a.NoError(err)

		engine.LeaderIdentity = PEM(engine.Self.Pid.GetID())

		t.Run("Bad Version", func(t *testing.T) {
			err = engine.handleUnicastVaaV1(&tsscommv1.Unicast_Vaav1{
				Vaav1: &tsscommv1.VaaV1Info{
					Marshaled: bts,
				},
			})

			a.ErrorContains(err, errNotVaaV1.Error())
		})

		v.Version = vaa.VaaVersion1
		bts, err = v.Marshal()
		a.NoError(err)

		t.Run("Bad Signature", func(t *testing.T) {
			err = engine.handleUnicastVaaV1(&tsscommv1.Unicast_Vaav1{
				Vaav1: &tsscommv1.VaaV1Info{
					Marshaled: bts,
				},
			})

			a.ErrorContains(err, "signature")
		})

		t.Run("Bad Marshal", func(t *testing.T) {
			err = engine.handleUnicastVaaV1(&tsscommv1.Unicast_Vaav1{
				Vaav1: &tsscommv1.VaaV1Info{
					Marshaled: []byte("BadMarshal"),
				},
			})

			a.ErrorContains(err, "unmarshal")
		})

		t.Run("nil VAA", func(t *testing.T) {
			err = engine.handleUnicastVaaV1(nil)

			a.ErrorContains(err, "nil")
		})

		t.Run("no guardian set state", func(t *testing.T) {
			engine.gst = nil

			err = engine.handleUnicastVaaV1(&tsscommv1.Unicast_Vaav1{
				Vaav1: &tsscommv1.VaaV1Info{
					Marshaled: bts,
				},
			})

			a.ErrorContains(err, "guardianSet")
		})
	})

	t.Run("witness Vaas", func(t *testing.T) {
		a := assert.New(t)

		v, gs := genVaaAndGuardianSet(a)

		gst := whcommon.NewGuardianSetState(nil)
		gst.Set(gs)

		engines := load5GuardiansSetupForBroadcastChecks(a)
		engine := engines[0] // Not starting engine so it doesn't run BeginTSSSign

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
		defer cancel()

		ctx = testutils.MakeSupervisorContext(ctx)

		a.ErrorContains(engine.WitnessNewVaa(v), errTssEngineNotStarted.Error())

		engine.Start(ctx)

		engine.isleader = true
		a.ErrorContains(engine.WitnessNewVaa(v), errNilGuardianSetState.Error())
		engine.gst = gst

		a.NoError(engine.WitnessNewVaa(v))

		a.ErrorContains(engine.WitnessNewVaa(nil), "nil")
		a.NoError(engine.WitnessNewVaa(v))

		engine.messageOutChan = nil
		a.NoError(engine.WitnessNewVaa(v)) //shouldn't output error but log.

		v.Version += 1
		a.NoError(engine.WitnessNewVaa(v))

		engine = nil
		a.ErrorContains(engine.WitnessNewVaa(v), errNilTssEngine.Error())
	})
}

func createX509Cert(dnsName string) *x509.Certificate {
	// using random serial number
	var serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)

	serialNumber, err := crand.Int(crand.Reader, serialNumberLimit)
	if err != nil {
		panic(err)
	}

	tmpl := x509.Certificate{
		SerialNumber:          serialNumber,
		Subject:               pkix.Name{Organization: []string{"tsscomm"}},
		SignatureAlgorithm:    x509.ECDSAWithSHA256,
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 366 * 40), // valid for > 40 years used for tests...
		BasicConstraintsValid: true,

		DNSNames:    []string{"localhost", dnsName},
		IPAddresses: []net.IP{net.IPv4(127, 0, 0, 1)},
	}
	return &tmpl
}

func TestFetchPartyId(t *testing.T) {
	a := assert.New(t)
	engines := load5GuardiansSetupForBroadcastChecks(a)
	e1 := engines[0]
	id, err := e1.FetchIdentity(e1.Guardians.peerCerts[0])
	a.NoError(err)
	a.True(e1.Self.Pid.Equals(id.Pid))

	crt := createX509Cert("localhost")
	_, err = e1.FetchIdentity(crt)
	a.ErrorContains(err, "unsupported") // cert.PublicKey=nil

	crt.PublicKey = []byte{1, 2, 3}
	_, err = e1.FetchIdentity(crt)
	a.ErrorContains(err, "unknown")
}

func TestCleanup(t *testing.T) {
	a := assert.New(t)
	engines := load5GuardiansSetupForBroadcastChecks(a)
	e1 := engines[0]

	uuid1 := uuid{1}
	e1.received[uuid1] = &broadcaststate{
		timeReceived: time.Now().Add(time.Minute * 10 * (-1)),
	}

	uuid2 := uuid{2}
	e1.received[uuid2] = &broadcaststate{
		timeReceived: time.Now(),
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

func (b *badtssMessage) ValidateBasic() bool            { return true }
func (b *badtssMessage) GetRound() int                  { return 2 }
func (b *badtssMessage) Content() common.MessageContent { return nil }
func (b *badtssMessage) GetFrom() *common.PartyID       { panic("unimplemented") }
func (b *badtssMessage) GetTo() []*common.PartyID       { panic("unimplemented") }
func (b *badtssMessage) IsBroadcast() bool              { panic("unimplemented") }
func (b *badtssMessage) IsToOldAndNewCommittees() bool  { panic("unimplemented") }
func (b *badtssMessage) IsToOldCommittee() bool         { panic("unimplemented") }
func (b *badtssMessage) String() string                 { panic("unimplemented") }
func (b *badtssMessage) Type() string                   { panic("unimplemented") }
func (b *badtssMessage) WireMsg() *common.MessageWrapper {
	return &common.MessageWrapper{
		TrackingID: nil,
	}
}
func (b *badtssMessage) WireBytes() ([]byte, *common.MessageRouting, error) {
	return nil, nil, errors.New("bad message")
}
func (b *badtssMessage) GetProtocol() common.ProtocolType {
	return common.ProtocolFROST
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
	e1.fpErrChannel <- common.NewTrackableError(errors.New("test"), "test", -1, nil, &tsscommon.TrackingID{})
	e1.fpErrChannel <- nil

	time.Sleep(time.Millisecond * 200)
}

func TestDefaultSameLeader(t *testing.T) {
	a := assert.New(t)

	engines := load5GuardiansSetupForBroadcastChecks(a)

	leader := engines[0].LeaderIdentity
	a.NotNil(leader)

	for _, e := range engines {
		a.Equal(e.LeaderIdentity, leader)

		if bytes.Equal(PEM(e.Self.Pid.GetID()), leader) {
			a.True(e.isleader)
		} else {
			a.False(e.isleader)
		}
	}
}

func TestNoFaultsFlow(t *testing.T) {
	// checking metrics first since this is a bit flakey.
	t.Run("with correct metrics", func(t *testing.T) {
		sigProducedCntr.Reset()
		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		dgst := party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Second*20)
		defer cancel()

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		fmt.Println("engines started, requesting sigs")

		m := dto.Metric{}

		cID := vaa.ChainID(1)
		// all engines are started, now we can begin the protocol.
		for _, engine := range engines {
			tmp := make([]byte, 32)
			copy(tmp, dgst[:])
			engine.BeginAsyncThresholdSigningProtocol(tmp, cID, reportableConsistancyLevel)
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("context expired")
		}

		time.Sleep(time.Millisecond * 500) // ensuring all other engines have finished and not just one of them.

		sigProducedCntr.WithLabelValues(cID.String()).Write(&m)
		a.Equal(engines[0].Threshold+1, int(m.Counter.GetValue()))
	})

	// Setting up all engines (not just 5), each with a different guardian storage.
	// all will attempt to sign a single message, while outputing messages to each other,
	// and reliably broadcasting them.
	t.Run("Call multiple to sign the same digest", func(t *testing.T) {
		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		dgst := party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Second*10)
		defer cancel()

		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		dnchn := msgHandler(ctx, engines, 1)

		cID := vaa.ChainID(1)

		// demand signing multiple times.
		for range 10 {
			for _, engine := range engines {
				tmp := make([]byte, 32)
				copy(tmp, dgst[:])
				engine.BeginAsyncThresholdSigningProtocol(tmp, cID, reportableConsistancyLevel)
			}
			fmt.Println()
		}

		time.Sleep(time.Millisecond * 500)
		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("context expired")
		}
	})

	t.Run("19 signers", func(t *testing.T) {
		t.SkipNow() // No tss19 engines available at the moment.
		a := assert.New(t)
		engines, err := loadGuardians(19, "tss19")
		a.NoError(err)

		dgst := party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		dnchn := msgHandler(ctx, engines, 1)

		cID := vaa.ChainID(1)

		for _, engine := range engines {
			tmp := make([]byte, 32)
			copy(tmp, dgst[:])
			engine.BeginAsyncThresholdSigningProtocol(tmp, cID, reportableConsistancyLevel)
		}

		time.Sleep(time.Millisecond * 500)
		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("context expired")
		}
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

				engine.BeginAsyncThresholdSigningProtocol(tmp, 1, reportableConsistancyLevel)
			}
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("context expired")
		}
	})

	t.Run("with nonreportable consistency level", func(t *testing.T) {
		// test will check thatno FT is triggered when the consistency level is non-reportable.
		// does so by starting signing for 2 out of 3 guardians and then wait for timeout.
		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		dgst := party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Second*10)
		defer cancel()

		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		dnchn := msgHandler(ctx, engines, 1)

		cID := vaa.ChainID(1)

		e := getSigningGuardian(a, engines, party.SigningTask{
			Digest:       dgst,
			Faulties:     []*common.PartyID{},
			AuxilaryData: chainIDToBytes(cID),
		})

		for _, engine := range engines {
			if e.Self.Pid.Equals(engine.Self.Pid) {
				continue
			}

			tmp := make([]byte, 32)
			copy(tmp, dgst[:])
			engine.BeginAsyncThresholdSigningProtocol(tmp, cID, nonReportableConsistancyLevel)
		}

		if !ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("signature shouldn't have been created")
		}
	})

	t.Run("TSS sign after VAA seen by leader", func(t *testing.T) {
		a := assert.New(t)

		nvaa, gs := genVaaAndGuardianSet(a)

		// ensuring valid vaa.
		a.NoError(nvaa.Verify(gs.Keys))

		gst := whcommon.NewGuardianSetState(nil)
		gst.Set(gs)

		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*30)
		defer cancel()

		engines[0].isleader = true
		for _, engine := range engines {
			engine.LeaderIdentity = PEM(engines[0].Self.Pid.GetID())
			engine.SetGuardianSetState(gst)
			a.NoError(engine.Start(ctx))
		}

		dnchn := msgHandler(ctx, engines, 1)

		engines[0].WitnessNewVaa(nvaa)
		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("context expired without signature")
		}
	})
}

func genVaaAndGuardianSet(a *assert.Assertions) (*vaa.VAA, *whcommon.GuardianSet) {
	gss := whcommon.NewGuardianSetState(nil)
	_ = gss

	nvaa := &vaa.VAA{
		Version:          vaa.VaaVersion1,
		GuardianSetIndex: 0,
		Signatures:       nil,
		Timestamp:        time.Now(),
		Nonce:            12345,
		EmitterChain:     vaa.ChainIDPythNet,
		EmitterAddress:   vaa.Address{1, 2, 3, 4, 54, 56, 67},
		Payload:          []byte("hello world"),
		Sequence:         5578,
		ConsistencyLevel: pythnetFinalizedConsistencyLevel,
	}

	addrss := []ethcommon.Address{}
	sigs := []*vaa.Signature{}
	for i := range 5 {
		guardianSigner, err := guardiansigner.GenerateSignerWithPrivatekeyUnsafe(nil)
		a.NoError(err)

		dgst := nvaa.SigningDigest()

		tmp, err := guardianSigner.Sign(context.Background(), dgst[:])
		a.NoError(err)

		sig := &vaa.Signature{Index: uint8(i)}
		copy(sig.Signature[:], tmp)

		sigs = append(sigs, sig)

		addrss = append(addrss, crypto.PubkeyToAddress(guardianSigner.PublicKey(context.Background())))
	}

	gs := whcommon.NewGuardianSet(addrss, 0)

	nvaa.Signatures = sigs
	return nvaa, gs
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
	// t.Skip("Skipping these test until we decide about anouncing mechanism.")

	t.Run("single server crashes", func(t *testing.T) {
		t.Skip("TODO: handle server crashes")
	})

	t.Run("server crashes during signing multiple digests", func(t *testing.T) { t.Skip("TODO: handle server crashes") })

	t.Run("cant sign after f faults", func(t *testing.T) { t.Skip("TODO: handle server crashes") })

	t.Run("metric cleanup", func(t *testing.T) {
		// run for a few signatures, and ensure the metrics are cleaned up.
		a := assert.New(t)
		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		n := 3
		chainId := vaa.ChainID(1)
		digests := make([]party.SigningTask, n)
		for i := 0; i < n; i++ {
			digests[i] = party.SigningTask{
				Digest:       [32]byte{byte(i + 1)},
				Faulties:     nil,
				AuxilaryData: chainIDToBytes(chainId),
			}
		}

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute*4)
		defer cancel()

		fmt.Println("starting engines.")
		for _, engine := range engines {
			engine.Configurations.MaxSignerTTL = time.Second * 4
			a.NoError(engine.Start(ctx))
		}

		e := getSigningGuardian(a, engines, digests...)
		a.NotNil(e)

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, len(digests))

		fmt.Println("engines started, requesting sigs")

		for _, d := range digests {
			d := d

			for _, engine := range engines {
				engine.BeginAsyncThresholdSigningProtocol(d.Digest[:], chainId, reportableConsistancyLevel)
			}
		}

		timer := time.After(engines[0].maxSignerTTL() * 4)

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("context expired")
		}

		<-timer

		for _, e := range engines {
			e.SignatureMetrics.Range(func(k, v interface{}) bool {
				fmt.Println(k, v)
				a.Fail("metrics not cleaned up")
				return false
			})
		}
	})

	t.Run("Two quorums only one guardian in conjunction", func(t *testing.T) {
		t.Skip("TODO: Make one of the signers of VAAv1 send the VAAv1 (similar to leader mechanism), so the others will also sign.")

		// This test simulates an error we've seen while testing with real data:
		// 3 servers manage to generate VAA but not VAAv2 (TSS signatuer).
		// That is, 3 servers saw the same digest, but only one of them was part of the tss-committee.
		// As a result, the VAA was generated, but the VAAv2 was not (since the others in the committee didn't f+1 messages that started signing).
		a := assert.New(t)

		supctx := testutils.MakeSupervisorContext(context.Background())
		ctx, cancel := context.WithTimeout(supctx, time.Minute)
		defer cancel()

		cID := vaa.ChainID(1)
		tsk := party.SigningTask{
			Digest:       party.Digest{1, 2, 3, 4, 5, 6, 7, 8, 9},
			Faulties:     []*common.PartyID{},
			AuxilaryData: chainIDToBytes(cID),
		}

		engines, err := loadGuardians(5, "tss5")
		a.NoError(err)

		fmt.Println("starting engines.")
		for _, engine := range engines {
			a.NoError(engine.Start(ctx))
		}

		signers := getSigningGuardians(a, engines, tsk)
		a.Len(signers, 3)

		fmt.Println("msgHandler settup:")
		dnchn := msgHandler(ctx, engines, 1)

		nonSigners := make([]*Engine, 0, 2)
		for _, engine := range engines {
			if !contains(signers, engine) {
				nonSigners = append(nonSigners, engine)
			}
		}

		// starting 3 signers where two aren't in the committee and one is.
		for _, engine := range append(nonSigners, signers[0]) {
			tmp := make([]byte, 32)
			copy(tmp, tsk.Digest[:])

			engine.BeginAsyncThresholdSigningProtocol(tmp, cID, reportableConsistancyLevel)
		}

		if ctxExpiredFirst(ctx, dnchn) {
			a.FailNow("context expired")
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
			parsed := generateFakeMessageWithRandomContent(from.Pid, to.Pid, rnd, msgDigest)
			bts, _, err := parsed.WireBytes()
			a.NoError(err)

			m := &IncomingMessage{
				Source: from,
				Content: &tsscommv1.PropagatedMessage{Message: &tsscommv1.PropagatedMessage_Unicast{
					Unicast: &tsscommv1.Unicast{
						Content: &tsscommv1.Unicast_Tss{
							Tss: &tsscommv1.TssContent{Payload: bts},
						},
					},
				}},
			}

			err = e2.handleUnicast(m)
			a.ErrorContains(err, "received broadcast type message in unicast")
		}
	})

	t.Run("Echo", func(t *testing.T) {
		t.Skip("TODO: Right now there are no 'bad' rounds for echoes (since we've switched to frost), ecdsa might have those. so we might need to include a mechanism to review protocol type in each message.")

		msgDigest := party.Digest{2}
		for _, rnd := range unicastRounds {
			parsed := generateFakeMessageWithRandomContent(from.Pid, to.Pid, rnd, msgDigest)
			bts, _, err := parsed.WireBytes()
			a.NoError(err)

			m := &IncomingMessage{
				Source: from,
				Content: &tsscommv1.PropagatedMessage{Message: &tsscommv1.PropagatedMessage_Echo{
					Echo: &tsscommv1.Echo{
						Message: &tsscommv1.SignedMessage{
							Content: &tsscommv1.SignedMessage_TssContent{
								TssContent: &tsscommv1.TssContent{Payload: bts},
							},
							Sender:    uint32(from.CommunicationIndex),
							Signature: nil,
						},
					},
				}},
			}
			a.NoError(e1.sign(uuid{}, m.Content.GetEcho().Message))

			err = e2.handleBroadcast(m)
			// a.ErrorIs(err, errBadRoundsInBroadcast)
		}
	})
}

func generateFakeParsedMessageWithRandomContent(from, to *common.PartyID, rnd signingRound, digest party.Digest) broadcastMessage {
	fake := generateFakeMessageWithRandomContent(from, to, rnd, digest)
	return &deliverableMessage{&parsedTssContent{fake, ""}}
}

// if to == nil it's a broadcast message.
func generateFakeMessageWithRandomContent(from, to *common.PartyID, rnd signingRound, digest party.Digest) common.ParsedMessage {
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
		meta    = common.MessageRouting{From: from, IsBroadcast: true}
		content common.MessageContent
	)

	switch rnd {
	case round2Message:
		// TODO: change to frost message!
		content = &sign.Broadcast2{
			Di: rndmBigNumber.Bytes(),
			Ei: rndmBigNumber.Bytes(),
		}
	case round3Message:
		if to == nil {
			panic("not a broadcast message")
		}
		meta = common.MessageRouting{From: from, To: []*common.PartyID{to}, IsBroadcast: false}

		content = &sign.Broadcast3{
			Zi: rndmBigNumber.Bytes(),
		}
	default:
		panic("unknown round")
	}

	return common.NewMessage(meta, content, common.NewMessageWrapper(meta, content, trackingId))
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
	Sender *Identity
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
			chns[en.Self.Pid.GetID()] = make(chan msgg, 10000)
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

					case msg := <-chns[engine.Self.Pid.GetID()]:
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
						sg, err := frost.Secp256k1SignatureTranslate(sig)
						if err != nil {
							panic("failed to translate signature:" + err.Error())
						}

						pk := engine.GetPublicKey()
						if err := sg.Verify(pk, sig.M); err != nil {
							panic("failed to verify signature:" + err.Error())
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
	for _, id := range pids {
		feedChn := chns[id.Pid.GetID()]
		feedChn <- msgg{
			Sender:   engine.Self,
			Sendable: m.cloneSelf(),
		}
	}
}

func broadcast(chns map[string]chan msgg, engine *Engine, m Sendable) {
	for _, feedChn := range chns {
		feedChn <- msgg{
			Sender:   engine.Self,
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

// Used to receive all messages for some engine, then feed them all at once, and collect the result.
// on error returns err.
// simulates echoes for each message too!
type echoFeed struct {
	eng      *Engine
	peers    []*Engine
	messages []IncomingMessage
}

func (b *echoFeed) addMessage(src *Engine, m Sendable) {
	inc := IncomingMessage{
		Source:  src.Self,
		Content: m.GetNetworkMessage(),
	}

	b.messages = append(b.messages, inc)
}

func (b *echoFeed) feedWithEchoes() error {
	defer func() {
		b.messages = nil // clear messages after feeding.
	}()

	if b.messages == nil {
		return nil
	}

	for _, msg := range b.messages {
		if err := b.eng.handleIncomingTssMessage(&msg); err != nil {
			return err
		}

		echo := b.genEcho(msg)
		// for each message: create fictional echo, making the guardian think that all other guardians have echoed it.
		for _, v := range b.peers {
			Incoming := &IncomingMessage{
				Source:  v.Self,
				Content: echo.GetNetworkMessage(),
			}

			if err := b.eng.handleIncomingTssMessage(Incoming); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *echoFeed) genEcho(msg IncomingMessage) *Echo {
	// src := msg.GetSource()
	// sig := msg.Content.GetEcho().Message.Signature

	parsed, err := b.eng.parseBroadcast(&msg)
	if err != nil {
		panic(err) // shouldn't happen in the test.
	}

	return b.eng.makeEcho(&msg, parsed)
}

func TestSigCounter(t *testing.T) {
	a := assert.New(t)

	supctx := testutils.MakeSupervisorContext(context.Background())

	t.Run("MaxCountBlockAdditionalUpdates", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		// t.Skip("TODO: implement this test, fails since we've moved to broadcast only messages!")

		cID := vaa.ChainID(0)
		tsks := []party.SigningTask{
			party.SigningTask{party.Digest{1}, []*common.PartyID{}, chainIDToBytes(cID)},
			party.SigningTask{party.Digest{2}, []*common.PartyID{}, chainIDToBytes(cID)},
		}
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1 := getSigningGuardian(a, engines, tsks...)

		e1.maxSimultaneousSignatures = 1
		feeder := &echoFeed{
			eng:   e1,
			peers: engines,
		}

		signersTask0 := getSigningGuardians(a, engines, tsks[0])
		signersTask1 := getSigningGuardians(a, engines, tsks[1])

		for taskNum, committee := range [][]*Engine{signersTask0, signersTask1} {
			for _, e := range committee {
				e.Start(ctx)

				msg := beginSigningAndGrabMessage(e, tsks[taskNum].Digest[:], cID)
				feeder.addMessage(e, msg)
				err := feeder.feedWithEchoes()
				if err != nil {
					a.ErrorContains(err, "maximum number of simultaneous")

					return
				}
			}
		}

		t.FailNow() // expected feeding to fail due to maxSimultaneousSignatures.
	})

	t.Run("ErrorReduceCount", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		// Tests might fail due to change of the GuardianStorage files
		cID := vaa.ChainID(0)
		tsks := []party.SigningTask{
			party.SigningTask{party.Digest{1}, []*common.PartyID{}, chainIDToBytes(cID)},
		}
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1 := getSigningGuardian(a, engines, tsks...)
		e1.maxSimultaneousSignatures = 1

		e1.Start(ctx)

		msg := beginSigningAndGrabMessage(e1, tsks[0].Digest[:], cID)

		feeder := &echoFeed{
			eng:   e1,
			peers: engines,
		}

		feeder.addMessage(e1, msg)
		a.NoError(feeder.feedWithEchoes())

		incoming := &IncomingMessage{
			Source:  e1.Self,
			Content: msg.GetNetworkMessage(),
		}

		parsed, err := e1.parseTssContent(incoming.toBroadcastMsg().Message.GetTssContent(), incoming.GetSource())
		a.NoError(err)

		tid := parsed.getTrackingID()
		// test:
		a.Equal(e1.sigCounter.digestToGuardiansLen(), 1)
		select {
		case e1.fpErrChannel <- common.NewTrackableError(fmt.Errorf("dummyerr"), "de", -1, e1.Self.Pid, tid):
		case <-time.After(time.Second * 1):
			t.FailNow()
			return
		}
		time.Sleep(time.Millisecond * 500)

		a.Equal(e1.sigCounter.digestToGuardiansLen(), 0)
	})

	t.Run("sigDoneReduceCount", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		// Tests might fail due to change of the GuardianStorage files
		cID := vaa.ChainID(0)
		tsks := []party.SigningTask{
			party.SigningTask{party.Digest{1}, []*common.PartyID{}, chainIDToBytes(cID)},
		}
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1 := getSigningGuardian(a, engines, tsks...)
		e1.maxSimultaneousSignatures = 1

		e1.Start(ctx)

		msg := beginSigningAndGrabMessage(e1, tsks[0].Digest[:], cID)

		feeder := &echoFeed{
			eng:   e1,
			peers: engines,
		}

		feeder.addMessage(e1, msg)
		a.NoError(feeder.feedWithEchoes())

		incoming := &IncomingMessage{
			Source:  e1.Self,
			Content: msg.GetNetworkMessage(),
		}

		parsed, err := e1.parseTssContent(incoming.toBroadcastMsg().Message.GetTssContent(), incoming.GetSource())
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
		<-e1.sigOutChan
		time.Sleep(time.Second * 1)
		a.Equal(e1.sigCounter.digestToGuardiansLen(), 0)
	})

	t.Run("CanHaveSimulSigners", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(supctx, time.Minute*1)
		defer cancel()

		// t.Skip("TODO: implement this test, fails since we've moved to broadcast only messages!")

		cID := vaa.ChainID(0)
		tsks := []party.SigningTask{
			party.SigningTask{party.Digest{1}, []*common.PartyID{}, chainIDToBytes(cID)},
			party.SigningTask{party.Digest{2}, []*common.PartyID{}, chainIDToBytes(cID)},
		}
		engines := load5GuardiansSetupForBroadcastChecks(a)
		e1 := getSigningGuardian(a, engines, tsks...)

		e1.maxSimultaneousSignatures = 2
		feeder := &echoFeed{
			eng:   e1,
			peers: engines,
		}

		signersTask0 := getSigningGuardians(a, engines, tsks[0])
		signersTask1 := getSigningGuardians(a, engines, tsks[1])

		for taskNum, committee := range [][]*Engine{signersTask0, signersTask1} {
			for _, e := range committee {
				e.Start(ctx)

				msg := beginSigningAndGrabMessage(e, tsks[taskNum].Digest[:], cID)
				feeder.addMessage(e, msg)
				err := feeder.feedWithEchoes()
				a.NoError(err)
			}
		}
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
	go e1.BeginAsyncThresholdSigningProtocol(dgst, cid, reportableConsistancyLevel)

	var msg Sendable
	for { // cleaning the channel, and taking one of the messages.
		select {
		case tmp := <-e1.ProducedOutputMessages():
			msg = tmp
			parsed, err := e1.parseBroadcast(&IncomingMessage{
				Source:  e1.Self,
				Content: tmp.GetNetworkMessage(),
			})
			if err != nil {
				panic("failed to parse broadcast message: " + err.Error())
			}

			if _, ok := parsed.(*deliverableMessage); !ok {
				continue
			}

			return msg

		case <-time.After(time.Second * 5):
			// This means the signer wasn't one of the signing committees. (did the Guardian storage change?)
			// if it did, just make sure this engine is expected to sign, else use the right engine in the test.
			panic("timeout!")
		}
	}
	return msg
}

func round1NumberOfMessages(e1 *Engine) int {
	// we only send one since this is a broadcast message...
	return 1
}

func contains(lst []*Engine, e *Engine) bool {
	for _, l := range lst {
		if l.Self.Pid.Equals(e.Self.Pid) {
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
