package tss

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"math/big"
	"net"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/internal/testutils"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	"github.com/stretchr/testify/assert"
	"github.com/yossigi/tss-lib/v2/ecdsa/keygen"
	"github.com/yossigi/tss-lib/v2/tss"
)

const (
	Participants = 5
	Threshold    = 2 // not including, meaning 3 guardians are needed to sign.
)

type dkgSetupPlayer struct {
	secretKey *ecdsa.PrivateKey
	*tss.PartyID

	//generated from the secretKey.

	// sorted according to theIdToPIDMapping.
	peerCerts     []PEM
	tlsPEM        PEM
	tlsPrivateKey PEM

	//same for all guardians
	LoadDistributionKey []byte

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

func TestSetUpGroup(t *testing.T) {
	t.SkipNow() // manual test only.
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

	for i, guardian := range guardians {
		a.NotNil(guardian)

		bts, err := json.MarshalIndent(guardian, "", "  ")
		a.NoError(err)
		fmt.Println(string(bts))

		fname, err := testutils.GetMockGuardianTssStorage(i)
		a.NoError(err)

		// fname := fmt.Sprintf("%s.json", strings.Split(guardian.Self.Id, ":")[0])
		err = os.WriteFile(fname, bts, 0777)
		a.NoError(err)
	}

}

func setupPlayers(a *assert.Assertions) []*dkgSetupPlayer {

	orderedKeysByPublicKey := getOrderedKeys(a)

	return genPlayers(orderedKeysByPublicKey)
}

// var listOfGuardians = []string{
// 	"t-gcp-threshsignnet-asia-01.gcp.testnet.xlabs.xyz",
// 	"t-gcp-threshsignnet-usw-01.gcp.testnet.xlabs.xyz",
// 	"t-gcp-threshsignnet-use-01.gcp.testnet.xlabs.xyz",
// 	"t-gcp-threshsignnet-euc-01.gcp.testnet.xlabs.xyz",
// 	"t-gcp-threshsignnet-euw-01.gcp.testnet.xlabs.xyz",
// }

func genPlayers(orderedKeysByPublicKey []*ecdsa.PrivateKey) []*dkgSetupPlayer {
	all := make([]*dkgSetupPlayer, Participants)
	partyIDS := make(tss.UnSortedPartyIDs, Participants)
	for i := 0; i < Participants; i++ {
		pnm := strconv.Itoa(i)
		pk, err := internal.PublicKeyToPem(&orderedKeysByPublicKey[i].PublicKey)
		if err != nil {
			panic(err)
		}
		partyIDS[i] = &tss.PartyID{
			MessageWrapper_PartyID: &tss.MessageWrapper_PartyID{
				// Id:      fmt.Sprintf("%s:%v", listOfGuardians[i], 8998),
				Id:      pnm,
				Moniker: pnm,
				Key:     pk,
			},
			Index: -1, // not known until sorted
		}

		all[i] = &dkgSetupPlayer{
			secretKey:      orderedKeysByPublicKey[i],
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

	loadBalancingKey := make([]byte, 32)
	_, err := rand.Read(loadBalancingKey)
	if err != nil {
		panic(err)
	}

	x509Certs := make([]PEM, len(sortedPartyIDS))
	for i, player := range all {
		player.PeerContext = tss.NewPeerContext(sortedPartyIDS)
		player.Parameters = tss.NewParameters(tss.S256(), player.PeerContext, player.PartyID, Participants, Threshold)
		player.IdToPIDmapping = IdToPIDmapping

		tmpl := createX509Cert(strings.Split(player.Id, ":")[0])

		x509 := internal.NewTLSCredentials(player.secretKey, tmpl)
		x509Certs[i] = internal.CertToPem(x509)

		player.peerCerts = x509Certs
		player.tlsPEM = internal.CertToPem(x509)
		player.tlsPrivateKey = internal.PrivateKeyToPem(player.secretKey)

		tmp := make([]byte, 32)
		copy(tmp, loadBalancingKey)
		player.LoadDistributionKey = tmp

		player.setNewKeygenHandler()
	}
	return all
}

func createX509Cert(dnsName string) *x509.Certificate {
	// using random serial number
	var serialNumberLimit = new(big.Int).Lsh(big.NewInt(1), 128)

	serialNumber, err := rand.Int(rand.Reader, serialNumberLimit)
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

func getOrderedKeys(a *assert.Assertions) []*ecdsa.PrivateKey {
	orderedKeysByPublicKey := make([]*ecdsa.PrivateKey, Participants)
	for i := range orderedKeysByPublicKey {
		sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		a.NoError(err)

		orderedKeysByPublicKey[i] = sk

	}
	sort.Slice(orderedKeysByPublicKey, func(i, j int) bool {
		pk1, err := internal.PublicKeyToPem(&orderedKeysByPublicKey[i].PublicKey)
		a.NoError(err)
		pk2, err := internal.PublicKeyToPem(&orderedKeysByPublicKey[j].PublicKey)
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
		Self: player.PartyID,

		Guardians: player.PeerContext.IDs(),

		TlsX509:    player.tlsPEM,
		PrivateKey: player.tlsPrivateKey,

		GuardianCerts: player.peerCerts,

		Threshold:             Threshold,
		SavedSecretParameters: m,
		LoadDistributionKey:   player.LoadDistributionKey,
		signingKey:            &ecdsa.PrivateKey{},

		Configurations: Configurations{
			MaxSignerTTL:   defaultMaxSignerTTL,
			DelayGraceTime: defaultDelayGraceTime,
		},
	}
}
