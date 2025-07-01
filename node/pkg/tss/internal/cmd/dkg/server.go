package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/certusone/wormhole/node/pkg/internal/testutils"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	engine "github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/comm"
	"github.com/certusone/wormhole/node/pkg/tss/internal/cmd"
	"github.com/fxamacker/cbor/v2"
	"github.com/xlabs/multi-party-sig/protocols/frost"
	"github.com/xlabs/multi-party-sig/protocols/frost/sign"
	"github.com/xlabs/tss-lib/v2/party"
	"go.uber.org/zap"
)

var cnfgPath = flag.String("cnfg", "/workspaces/wormhole/node/pkg/tss/internal/cmd/dkg/5/dkg.json", "path to config file in json format used to run the protocol")

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = testutils.MakeSupervisorContext(ctx)
	logger := supervisor.Logger(ctx)

	logger.Info("Loading KeyGenerator and GuardianStorage for DKG...")
	cnfgs := loadConfigsFromFlags()

	keygen, gst := keygeneratorSetup(cnfgs)

	keygen.Start(ctx)

	logger.Info("Setting up server...", zap.String("networkName", gst.Self.NetworkName()))

	srvr, err := comm.NewServer(gst.Self.NetworkName(), logger, keygen)
	if err != nil {
		panic("failed to create a new server, err: " + err.Error())
	}

	go func() {
		if err := srvr.Run(ctx); err != nil {
			logger.Error("Failed to run server", zap.Error(err))
		}
	}()

	time.Sleep(2 * time.Second) // wait for the server to start
	logger.Info("DKG server is running, waiting for connections...")
	srvr.WaitForConnections(ctx)

	logger.Info("Connections established, starting DKG...")

	i := 0
	for { // The loop should converge after 2~3 iterations.
		i++
		logger.Info("Making DKG Attemp", zap.Int("iteration", i))

		sha256sum := sha256.Sum256([]byte("dkg seed:" + strconv.Itoa(i)))
		resChn, err := keygen.StartDKG(party.DkgTask{
			Threshold: 0,
			Seed:      sha256sum,
		})

		if err != nil {
			panic("failed to start DKG, err: " + err.Error())
		}

		var tssConfigs *frost.Config
		select {
		case tssConfigs = <-resChn:
		case <-time.After(time.Second * 20):
			logger.Error("FAILED DKG. attempting again.")

			continue
		}

		logger.Info("DKG completed successfully", zap.Int("iteration", i-1))

		logger.Info("verifying randomly chosen PK is valid for smart-contract usage")
		if !sign.PublicKeyValidForContract(tssConfigs.PublicKey) {
			continue
		}

		cnfBytes, err := cbor.Marshal(tssConfigs)
		if err != nil {
			panic(fmt.Sprintf("failed to marshal frost config for guardian %d: %v", i, err))
		}

		gst.TSSSecrets = cnfBytes
		if err := gst.SetInnerFields(); err != nil {
			panic(fmt.Sprintf("failed to set inner fields of the GuardianStorage for guardian %d, err: %v", i, err))
		}
		logger.Info("GuardianStorage updated with TSS secrets. Storing result into file", zap.Int("guardianIndex", i))

		toStore, err := json.MarshalIndent(gst, "", "  ")
		if err != nil {
			panic(fmt.Sprintf("failed to marshal GuardianStorage for guardian %d, err: %v", i, err))
		}

		fname := path.Join(cnfgs.StorageLocation, "secrets.json")
		if gst.Self.CommunicationIndex != 0 {
			return // TODO: don't forget to remove this!
		}
		if err := os.WriteFile(fname, toStore, 0777); err != nil {
			panic(fmt.Sprintf("failed to write GuardianStorage to file %s, err: %v", fname, err))
		}

		logger.Info("GuardianStorage successfully written to file", zap.String("file", fname))
		time.Sleep(1 * time.Second)
		return
	}
}

func keygeneratorSetup(cnfgs *cmd.SetupConfigs) (engine.KeyGenerator, *engine.GuardianStorage) {
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
	gst := &engine.GuardianStorage{
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
	keygen, err := engine.NewKeyGenerator(gst)
	if err != nil {
		panic("failed to create a KeyGenerator, err: " + err.Error())
	}

	return keygen, gst
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
