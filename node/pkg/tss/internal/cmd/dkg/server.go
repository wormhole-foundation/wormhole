package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	engine "github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/internal/cmd"
)

var cnfgPath = flag.String("cnfg", "/workspaces/wormhole/node/pkg/tss/internal/cmd/dkg/dkg.json", "path to config file in json format used to run the protocol")

func main() {
	cnfgs := loadConfigsFromFlags()

	engineIds, _, err := cnfgs.IntoMaps()
	if err != nil {
		panic("failed to load configs, err: " + err.Error())
	}

	sortedIds := cmd.SortIdentities(engineIds)
	var self *engine.Identity
	// find self:
	for _, id := range sortedIds {
		if bytes.Equal(id.CertPem, cnfgs.Self.TlsX509) {
			// found self, set the communication index.

			self = id
			break
		}
	}

	if self == nil {
		panic("failed to find self in the sorted identities, please check the config file")
	}

	// Then Create a GuardianStorage.
	gst := engine.GuardianStorage{
		Configurations: engine.Configurations{},
		Self:           self,
		IdentitiesKeep: engine.IdentitiesKeep{
			Identities: sortedIds,
		},

		TlsX509:             cnfgs.Self.TlsX509,
		PrivateKey:          cnfgs.SelfSecret,
		Threshold:           cnfgs.WantedThreshold - 1,
		TSSSecrets:          nil,      // Hopefully, will be set at the end of this process.
		LoadDistributionKey: []byte{}, // don't need this for DKG
	}
	if err := gst.SetInnerFields(); err != nil {
		panic("failed to set inner fields of the GuardianStorage, err: " + err.Error())
	}

	// Then Create a KeyGenerator.
	keygen, err := engine.NewKeyGenerator(&gst)
	if err != nil {
		panic("failed to create a KeyGenerator, err: " + err.Error())
	}

	// Create Supervisor.

	fmt.Println(keygen)
	// Then start a Server.
	// Then Start the DKG.

	// Should load with basic knowledge of the tss package: Certs, and OWN TLS key.
}

func loadConfigsFromFlags() *cmd.SetupConfigs {
	flag.Parse()

	if *cnfgPath == "" {
		flag.PrintDefaults()

		panic("config path is empty, please provide a valid path to a config file")
	}

	f, err := os.ReadFile(*cnfgPath)
	if err != nil {
		panic("failed to read file, err: " + err.Error())
	}

	cnfg := &cmd.SetupConfigs{}
	err = json.Unmarshal(f, cnfg)
	if err != nil {
		panic("failed to unmarshal config, err: " + err.Error())
	}

	return cnfg
}
