package tss

import (
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/yossigi/tss-lib/v2/ecdsa/keygen"
	"github.com/yossigi/tss-lib/v2/tss"
)

const (
	Participants = 5
	Threshold    = 2 //  12 means 12 + 1 to  produce signature.
)

type dkgSetupPlayer struct {
	SecretKey []byte
	*tss.PartyID
	*tss.PeerContext
	*tss.Parameters
	IdToPIDmapping map[string]*tss.PartyID

	LocalParty tss.Party

	// communication channels
	Out               <-chan tss.Message
	ProtocolEndOutput <-chan *keygen.LocalPartySaveData
}

func TestGuardianStorageUnmarshal(t *testing.T) {
	var st GuardianStorage
	err := st.load(testutils.MustGetMockGuardianTssStorage())
	if err != nil {
		t.Error(err)
	}
}

func TestMarshalSecretKey(t *testing.T) {
	a := assert.New(t)
	sk, err := ecdsa.GenerateKey(tss.S256(), rand.Reader)
	a.NoError(err)

	bz := marshalEcdsaSecretkey(sk)
	unmarshaled := unmarshalEcdsaSecretKey(bz)
	a.True(sk.PublicKey.Equal(&unmarshaled.PublicKey))
	a.Equal(sk.D, unmarshaled.D)
}

func TestMarshalPK(t *testing.T) {
	a := assert.New(t)
	sk, err := ecdsa.GenerateKey(tss.S256(), rand.Reader)
	a.NoError(err)

	bz, _ := marshalEcdsaPublickey(&sk.PublicKey)
	unmarshaled, err := unmarshalEcdsaPublickey(tss.S256(), bz)
	a.NoError(err)

	a.True(sk.PublicKey.Equal(unmarshaled))
}

func TestSetUpGroup(t *testing.T) {
	a := assert.New(t)

	all := setupPlayers(a)

	for _, player := range all {
		p := player
		if err := p.LocalParty.Start(); err != nil && err.Cause() != nil {
			a.Fail("keygen failed to start: " + err.Cause().Error())
		}
	}

	fmt.Println("Setup done. Staring DKG")
	runDKG(a, all)
}

func passMsg(a *assert.Assertions, newMsg tss.Message, idToParty map[string]tss.Party) {
	bz, routing, err := newMsg.WireBytes()
	a.NoError(err)
	// parsedMsg doesn't contain routing, since it assumes this message arrive for this participant from outside.
	// as a result we'll use the routing of the wireByte msgs.
	parsedMsg, err := tss.ParseWireMessage(bz, routing.From, routing.IsBroadcast)
	a.NoError(err)

	if routing.IsBroadcast || routing.To == nil {
		for pID, p := range idToParty {
			if routing.From.GetId() == pID {
				continue
			}
			ok, err := p.Update(parsedMsg)
			a.True(ok, err.Error())

		}

		return
	}

	for _, id := range routing.To {
		p := idToParty[id.Id]
		ok, err := p.Update(parsedMsg)
		a.True(ok, err.Error())
	}
}

func runDKG(a *assert.Assertions, all []*dkgSetupPlayer) {
	done := 0

	idToFullPlayer := map[string]tss.Party{}
	for _, player := range all {
		idToFullPlayer[player.PartyID.Id] = player.LocalParty
	}

	guardians := make([]*GuardianStorage, Participants)
keygenLoop:
	for {
		bagOfMessages := make([]tss.Message, 0, Participants)
		for _, player := range all {
			select {
			case newMsg := <-player.Out:
				bagOfMessages = append(bagOfMessages, newMsg)

			case m := <-player.ProtocolEndOutput:
				player.handleKeygenEndMessage(m, guardians)
				done += 1

			case <-time.Tick(time.Millisecond * 500):
				fmt.Println("ticked")
			}

			if done >= Participants {
				break keygenLoop
			}
		}

		for _, msg := range bagOfMessages {
			passMsg(a, msg, idToFullPlayer)
		}
	}

	// for i, guardian := range guardians {
	// 	a.NotNil(guardian)
	// 	a.NoError(guardian.createSharedSecrets())
	// 	bts, err := json.MarshalIndent(guardian, "", "  ")
	// 	a.NoError(err)
	// 	fmt.Println(string(bts))

	// 	err = os.WriteFile(fmt.Sprintf("guardian%d.json", i), bts, 0777)
	// 	a.NoError(err)
	// }

}

func setupPlayers(a *assert.Assertions) []*dkgSetupPlayer {

	orderedKeysByPublicKey := getOrderedKeys(a)

	return genPlayers(orderedKeysByPublicKey)
}

func genPlayers(orderedKeysByPublicKey []*ecdsa.PrivateKey) []*dkgSetupPlayer {
	all := make([]*dkgSetupPlayer, Participants)
	partyIDS := make(tss.UnSortedPartyIDs, Participants)
	for i := 0; i < Participants; i++ {
		pnm := strconv.Itoa(i)
		pk, err := marshalEcdsaPublickey(&orderedKeysByPublicKey[i].PublicKey)
		if err != nil {
			panic(err)
		}
		partyIDS[i] = &tss.PartyID{
			MessageWrapper_PartyID: &tss.MessageWrapper_PartyID{
				Id:      pnm,
				Moniker: pnm,
				Key:     pk,
			},
			Index: -1, // not known until sorted
		}

		all[i] = &dkgSetupPlayer{
			SecretKey:      marshalEcdsaSecretkey(orderedKeysByPublicKey[i]),
			PartyID:        partyIDS[i],
			PeerContext:    nil, // known only all player IDs are known.
			Parameters:     nil,
			IdToPIDmapping: nil,
		}
	}

	sortedPartyIDS := tss.SortPartyIDs(partyIDS)
	IdToPIDmapping := map[string]*tss.PartyID{}
	for _, player := range all {
		IdToPIDmapping[player.PartyID.Id] = player.PartyID
	}

	for _, player := range all {
		player.PeerContext = tss.NewPeerContext(sortedPartyIDS)
		player.Parameters = tss.NewParameters(tss.S256(), player.PeerContext, player.PartyID, Participants, Threshold)
		player.IdToPIDmapping = IdToPIDmapping

		player.setNewKeygenHandler()
	}
	return all
}

func getOrderedKeys(a *assert.Assertions) []*ecdsa.PrivateKey {
	orderedKeysByPublicKey := make([]*ecdsa.PrivateKey, Participants)
	for i := range orderedKeysByPublicKey {
		sk, err := ecdsa.GenerateKey(tss.S256(), rand.Reader)
		a.NoError(err)

		orderedKeysByPublicKey[i] = sk

	}
	sort.Slice(orderedKeysByPublicKey, func(i, j int) bool {
		pk1, err := marshalEcdsaPublickey(&orderedKeysByPublicKey[i].PublicKey)
		a.NoError(err)
		pk2, err := marshalEcdsaPublickey(&orderedKeysByPublicKey[j].PublicKey)
		a.NoError(err)

		ibts := string(pk1)
		jbts := string(pk2)
		return ibts < jbts
	})
	return orderedKeysByPublicKey
}

func (player *dkgSetupPlayer) setNewKeygenHandler() {
	out := make(chan tss.Message, Participants)
	endOut := make(chan *keygen.LocalPartySaveData, 1) // ready for at least a single message.

	player.LocalParty = keygen.NewLocalParty(player.Parameters, out, endOut)
	player.Out = out
	player.ProtocolEndOutput = endOut
}

func (player *dkgSetupPlayer) handleKeygenEndMessage(m *keygen.LocalPartySaveData, guardians []*GuardianStorage) {
	i, err := m.OriginalIndex()
	if err != nil {
		panic(err)
	}
	guardians[i] = &GuardianStorage{
		Self:                  player.PartyID,
		Guardians:             player.PeerContext.IDs(),
		SecretKey:             player.SecretKey,
		Threshold:             Threshold,
		SavedSecretParameters: m,
	}
}
