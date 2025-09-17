package cmd

import (
	"crypto/ecdsa"
	"crypto/sha512"
	"encoding/hex"
	"fmt"

	engine "github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/internal"
	common "github.com/xlabs/tss-common"
)

type Identifier struct {
	Hostname string
	TlsX509  engine.PEM // PEM Encoded (see certs.go). Note, you must have the private key of this cert later.
	Port     int        // if one needs different ports, tell it here.
}

type SetupConfigs struct {
	NumParticipants int
	WantedThreshold int // should be non inclusive. That is, if you have n=19,f=6, then threshold=12 (13 guardians needed to sign).

	Self            Identifier
	SelfSecret      engine.PEM // PEM Encoded (see certs.go). Note, you must have the private key of this cert later.
	StorageLocation string     // The folder where secrets.json file will be saved.

	Peers        []Identifier
	Secrets      []engine.PEM
	SaveLocation []string
}

func (cnfg *SetupConfigs) Validate() error {
	if cnfg.NumParticipants < 1 {
		return fmt.Errorf("number of participants should be at least 1")
	}

	if len(cnfg.Peers) != cnfg.NumParticipants {
		return fmt.Errorf("number of guardian identifiers should be equal to number of participants")
	}

	if cnfg.WantedThreshold >= cnfg.NumParticipants-1 {
		return fmt.Errorf("threshold should be less than number of participants")
	}

	return nil
}

func (cnfg *SetupConfigs) IntoMaps() (keyToEngineIdentity map[string]*engine.Identity, keyToID map[string]*Identifier, err error) {
	keyToEngineIdentity = make(map[string]*engine.Identity, cnfg.NumParticipants)

	keyToID = map[string]*Identifier{}
	for i, peer := range cnfg.Peers {
		crt, err := internal.PemToCert(peer.TlsX509)
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

		pidbytes := sha512.Sum512_256(bts)
		pid := hex.EncodeToString(pidbytes[:]) // just to make sure it's valid hex.
		keyToEngineIdentity[string(bts)] = &engine.Identity{
			Pid: &common.PartyID{
				ID: string(pid),
			},
			KeyPEM:             bts,
			CertPem:            peer.TlsX509,
			Cert:               nil, // not stored, since we have the certPem.
			CommunicationIndex: 0,   // unknown until all identites are sorted according to their Party ids.
			Hostname:           peer.Hostname,
			Port:               peer.Port,
			Key:                nil, // Filled by the guardian storage on boot.
			VAAv1PubKey:        nil, // not used in dkg, so nil.
		}

		keyToID[string(bts)] = &cnfg.Peers[i]
	}

	return keyToEngineIdentity, keyToID, nil
}

// sorts the identities based on the partyID key (as tss-lib expects the parties to be sorted).
// then sets the communication index according to the sorted order.
func SortIdentities(unsortedIdentities map[string]*engine.Identity) []*engine.Identity {
	pids := make([]*common.PartyID, 0, len(unsortedIdentities))
	pidsToCertKey := make(map[string]string, len(unsortedIdentities))
	for _, p := range unsortedIdentities {
		pids = append(pids, p.Pid)
		pidsToCertKey[string(p.Pid.GetID())] = string(p.KeyPEM)
	}

	sortedPids := common.SortPartyIDs(pids)

	sortedIDS := make([]*engine.Identity, len(sortedPids))
	for i, pid := range sortedPids {
		key := pidsToCertKey[string(pid.GetID())]
		sortedIDS[i] = unsortedIdentities[key]
		sortedIDS[i].CommunicationIndex = engine.SenderIndex(i)
	}

	return sortedIDS
}
