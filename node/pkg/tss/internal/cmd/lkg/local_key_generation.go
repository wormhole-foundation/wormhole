// This file runs Distrubted Key Generation (DKG) protocol in local setting.
// That is, a gorotuine orchestrator simulates the network communication between the parties
// by collecting the outputs of the parties and feeding these output messages to the correct parties (using the `Update` method).

package main

import (
	"bytes"
	"crypto/rand"
	"encoding/gob"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"

	engine "github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/internal/cmd"
	"github.com/xlabs/multi-party-sig/pkg/math/curve"
	"github.com/xlabs/multi-party-sig/pkg/math/polynomial"
	"github.com/xlabs/multi-party-sig/pkg/math/sample"
	"github.com/xlabs/multi-party-sig/pkg/party"
	"github.com/xlabs/multi-party-sig/protocols/frost"
	"github.com/xlabs/multi-party-sig/protocols/frost/sign"
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

	cnfg := &cmd.SetupConfigs{}
	err = json.Unmarshal(f, cnfg)
	if err != nil {
		fmt.Println("failed to unmarshal config, err: ", err)

		return
	}

	Run(cnfg)
}

type LKGConfig cmd.SetupConfigs
type Identifier = cmd.Identifier

type dkgPlayer struct {
	whereToStore string

	// the identity of the player, tied to the x509 cert and its private key.
	self *engine.Identity
	ids  engine.IdentitiesKeep

	selfCert   []byte
	certKeyPEM []byte

	// same for all guardians // generated here.
	loadDistributionKey []byte
}

func Run(cnfg *cmd.SetupConfigs) {
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

	publicKey, f := makefrostKey(group, threshold)

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

		buff := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buff)

		if err := enc.Encode(cnf); err != nil {
			panic(fmt.Sprintf("failed to marshal frost config for guardian %d: %v", i, err))
		}

		guardians[i] = &engine.GuardianStorage{
			Configurations: engine.Configurations{},
			Self:           p.self,
			TlsX509:        p.selfCert,
			PrivateKey:     p.certKeyPEM,
			IdentitiesKeep: p.ids,
			Threshold:      threshold,
			// SavedSecretParameters: ,
			LoadDistributionKey: p.loadDistributionKey,
			TSSSecrets:          buff.Bytes(),
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

		if err := os.MkdirAll(all[i].whereToStore, 0700); err != nil {
			panic("Failed to create directory: " + err.Error())
		}

		fname := path.Join(all[i].whereToStore, "secrets.json")

		if err := os.WriteFile(fname, bts, 0600); err != nil {
			panic("Failed to write to disk: " + err.Error())
		}
	}

}

func makefrostKey(group curve.Secp256k1, threshold int) (curve.Point, *polynomial.Polynomial) {
	for range 128 {
		secret := sample.Scalar(rand.Reader, group)
		f := polynomial.NewPolynomial(group, threshold, secret)
		publicKey := secret.ActOnBase()
		if sign.PublicKeyValidForContract(publicKey) {
			return publicKey, f
		}
	}

	panic("could not generate a valid frost key after 128 attempts")
}

func setupPlayers(cnfg *cmd.SetupConfigs) ([]*dkgPlayer, error) {
	if err := cnfg.Validate(); err != nil {
		return nil, err
	}

	loadBalancingKey := make([]byte, 32)
	_, err := rand.Read(loadBalancingKey)
	if err != nil {
		return nil, err
	}

	unsortedIdentities, mp, err := cnfg.IntoMaps()
	if err != nil {
		return nil, err
	}

	sortedIDS := cmd.SortIdentities(unsortedIdentities)

	all := make([]*dkgPlayer, cnfg.NumParticipants)
	for _, id := range sortedIDS {
		index := 0
		for _, v := range cnfg.Peers {
			if string(v.TlsX509) == string(id.CertPem) {
				break
			}

			index++
		}

		tmp := make([]byte, 32)
		copy(tmp, loadBalancingKey)

		// peerContext := tss.NewPeerContext(sortedPids)

		gspecific := mp[string(id.KeyPEM)]

		p := &dkgPlayer{
			self:                id,
			whereToStore:        cnfg.SaveLocation[index],
			selfCert:            gspecific.TlsX509,
			certKeyPEM:          cnfg.Secrets[index],
			loadDistributionKey: tmp,

			ids: engine.IdentitiesKeep{
				Identities: sortedIDS,
			},
		}
		all[id.CommunicationIndex] = p

	}

	return all, nil
}
