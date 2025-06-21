// This file runs Distrubted Key Generation (DKG) protocol in local setting.
// That is, a gorotuine orchestrator simulates the network communication between the parties
// by collecting the outputs of the parties and feeding these output messages to the correct parties (using the `Update` method).

package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"sort"

	engine "github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	"github.com/fxamacker/cbor/v2"
	"github.com/xlabs/multi-party-sig/pkg/math/curve"
	"github.com/xlabs/multi-party-sig/pkg/math/polynomial"
	"github.com/xlabs/multi-party-sig/pkg/math/sample"
	"github.com/xlabs/multi-party-sig/pkg/party"
	"github.com/xlabs/multi-party-sig/protocols/frost"
	common "github.com/xlabs/tss-common"
)

var cnfgPath = flag.String("cnfg", "", "path to config file in json format used to run the protocol")

func main() {
	flag.Parse()

	if *cnfgPath == "" {
		flag.PrintDefaults()

		return
	}

	f, err := os.ReadFile(*cnfgPath)
	if err != nil {
		fmt.Println("failed to read file, err: ", err)

		return
	}

	cnfg := &LKGConfig{}
	err = json.Unmarshal(f, cnfg)
	if err != nil {
		fmt.Println("failed to unmarshal config, err: ", err)

		return
	}

	Run(cnfg)
}

type LKGConfig struct {
	NumParticipants int
	WantedThreshold int // should be non inclusive. That is, if you have n=19,f=6, then threshold=12 (13 guardians needed to sign).

	GuardianSpecifics []GuardianSpecifics
}

type GuardianSpecifics struct {
	Identifier         Identifier
	WhereToSaveSecrets string // where to save the secrets of this guardian.
}
type Identifier struct {
	// Self Signed, CA level cert.
	TlsX509 engine.PEM // PEM Encoded (see certs.go). Note, you must have the private key of this cert later.
}

type dkgPlayer struct {
	whereToStore string

	// the identity of the player, tied to the x509 cert and its private key.
	self *engine.Identity
	ids  engine.Identities

	selfCert []byte

	// same for all guardians // generated here.
	loadDistributionKey []byte

	// *tss.PeerContext
	// *tss.Parameters

	// localParty tss.Party
	frostconf *frost.Config

	// communication channels
	// out               <-chan tss.Message
	// protocolEndOutput <-chan *keygen.LocalPartySaveData
}

func Run(cnfg *LKGConfig) {
	if cnfg == nil {
		panic("config is nil")
	}

	// ensuring the threshold matches what the library expects.
	// (if n=5,f=1 and we want 2f+1 committees, then WantedThreshold should be equal to 2)
	cnfg.WantedThreshold -= 1

	fmt.Println("Setting up player secrets. This might take a while...")
	all, err := setupPlayers(cnfg)
	if err != nil {
		panic(err)
	}

	fmt.Println("all players generated their secrets. Starting the protocol. This might take several minutes.")
	simulateDKG(all, cnfg.WantedThreshold)
}

// this function simulates the results of running a DKG protocol.
func simulateDKG(all []*dkgPlayer, threshold int) {
	group := curve.Secp256k1{}
	secret := sample.Scalar(rand.Reader, group)
	f := polynomial.NewPolynomial(group, threshold, secret)
	publicKey := secret.ActOnBase()

	privateShares := make(map[party.ID]curve.Scalar, len(all))
	for _, p := range all {
		id := party.ID(p.self.Pid.GetID())

		privateShares[id] = f.Evaluate(id.Scalar(group))
	}

	verificationShares := make(map[party.ID]curve.Point, len(all))

	for _, p := range all {
		id := party.ID(p.self.Pid.GetID())

		point := privateShares[id].ActOnBase()
		verificationShares[id] = point
	}

	guardians := make([]*engine.GuardianStorage, len(all))
	for i, p := range all {
		id := party.ID(p.self.Pid.GetID())

		// can't be jsoned.
		cnf := &frost.Config{
			ID:           id,
			Threshold:    threshold,
			PrivateShare: privateShares[id],
			PublicKey:    publicKey,
			// ChainKey:           d.loadDistributionKey, // needed iff we use BIP32 key derivation.
			VerificationShares: party.NewPointMap(verificationShares),
		}

		cnfBytes, err := cbor.Marshal(cnf) // TODO: avoid cbor if possible later.
		if err != nil {
			panic(fmt.Sprintf("failed to marshal frost config for guardian %d: %v", i, err))
		}

		guardians[i] = &engine.GuardianStorage{
			Configurations: engine.Configurations{},
			Self:           p.self,
			TlsX509:        p.selfCert,
			PrivateKey:     nil, // each guardian stores this by themselves.
			Guardians:      p.ids,
			Threshold:      threshold,
			// SavedSecretParameters: ,
			LoadDistributionKey: p.loadDistributionKey,
			TSSSecrets:          cnfBytes,
		}
	}

	fmt.Println("All guardians have finished the protocol. Saving the secrets to disk.")

	for i, guardian := range guardians {
		if guardian == nil {
			panic(fmt.Sprintf("error in script. Guardian[%d] is nil", i))
		}

		bts, err := json.MarshalIndent(guardian, "", "  ")
		if err != nil {
			panic("")
		}

		if err := os.MkdirAll(all[i].whereToStore, 0777); err != nil {
			panic("Failed to create directory: " + err.Error())
		}

		fname := path.Join(all[i].whereToStore, "secrets.json")

		if err := os.WriteFile(fname, bts, 0777); err != nil {
			panic("Failed to write to disk: " + err.Error())
		}
	}

}

func (cnfg *LKGConfig) validate() error {
	if cnfg.NumParticipants < 1 {
		return fmt.Errorf("number of participants should be at least 1")
	}

	if len(cnfg.GuardianSpecifics) != cnfg.NumParticipants {
		return fmt.Errorf("number of guardian identifiers should be equal to number of participants")
	}

	if cnfg.WantedThreshold >= cnfg.NumParticipants-1 {
		return fmt.Errorf("threshold should be less than number of participants")
	}

	return nil
}

func setupPlayers(cnfg *LKGConfig) ([]*dkgPlayer, error) {
	if err := cnfg.validate(); err != nil {
		return nil, err
	}

	loadBalancingKey := make([]byte, 32)
	_, err := rand.Read(loadBalancingKey)
	if err != nil {
		return nil, err
	}

	unsortedIdentities, mp, err := cnfg.intoMaps()
	if err != nil {
		return nil, err
	}

	sortedIDS := sortIdentities(unsortedIdentities)

	all := make([]*dkgPlayer, cnfg.NumParticipants)
	for _, id := range sortedIDS {
		tmp := make([]byte, 32)
		copy(tmp, loadBalancingKey)

		// peerContext := tss.NewPeerContext(sortedPids)

		gspecific := mp[string(id.Pid.GetID())]

		p := &dkgPlayer{
			self:                id,
			whereToStore:        gspecific.WhereToSaveSecrets,
			selfCert:            gspecific.Identifier.TlsX509,
			loadDistributionKey: tmp,

			ids: engine.Identities{
				Identities: sortedIDS,
			},
		}
		all[id.CommunicationIndex] = p

	}

	return all, nil
}

// sorts the identities based on the partyID key (as tss-lib expects the parties to be sorted).
// then sets the communication index according to the sorted order.
func sortIdentities(unsortedIdentities map[string]*engine.Identity) []*engine.Identity {
	pids := make([]*common.PartyID, 0, len(unsortedIdentities))
	for _, p := range unsortedIdentities {
		pids = append(pids, p.Pid)
	}

	sortedPids := common.SortPartyIDs(pids)

	sortedIDS := make([]*engine.Identity, len(sortedPids))
	for i, pid := range sortedPids {
		sortedIDS[i] = unsortedIdentities[string(pid.GetID())]
		sortedIDS[i].CommunicationIndex = engine.SenderIndex(i)
	}

	return sortedIDS
}

func (cnfg *LKGConfig) intoMaps() (unsortedIdentities map[string]*engine.Identity, keyToSpecifics map[string]*GuardianSpecifics, err error) {
	unsortedIdentities = make(map[string]*engine.Identity, cnfg.NumParticipants)

	keyToSpecifics = map[string]*GuardianSpecifics{}
	for i, dt := range cnfg.GuardianSpecifics {
		crt, err := internal.PemToCert(dt.Identifier.TlsX509)
		if err != nil {
			return nil, nil, err
		}

		if len(crt.DNSNames) == 0 {
			return nil, nil, fmt.Errorf("expected DNS names in the cert")
		}

		pk, ok := crt.PublicKey.(*ecdsa.PublicKey) // TODO
		if !ok {
			return nil, nil, fmt.Errorf("expected ecdsa public key in certs")
		}

		bts, err := internal.PublicKeyToPem(pk)
		if err != nil {
			return nil, nil, err
		}

		dnsName := extractDnsFromCert(crt)

		unsortedIdentities[string(bts)] = &engine.Identity{
			Pid: &common.PartyID{
				ID: string(bts),
			},
			KeyPEM:             bts,
			CertPem:            dt.Identifier.TlsX509,
			Cert:               nil, // not stored, since we have the certPem.
			CommunicationIndex: 0,   // unknown until sorted.
			Hostname:           dnsName,
			Port:               0, // undefined.
		}

		keyToSpecifics[string(bts)] = &cnfg.GuardianSpecifics[i]
	}

	return unsortedIdentities, keyToSpecifics, nil
}

func extractDnsFromCert(crt *x509.Certificate) string {
	if len(crt.DNSNames) == 0 {
		panic("expected DNS names in the certificate")
	}

	sort.Strings(crt.DNSNames)

	dnsName := ""
	for _, name := range crt.DNSNames {
		if len(name) > 0 && name != "localhost" {
			dnsName = name // using the first non localhost name.
		}
	}

	return dnsName
}
