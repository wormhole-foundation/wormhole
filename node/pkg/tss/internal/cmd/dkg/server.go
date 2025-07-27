package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"flag"
	"os"
	"path"
	"strconv"
	"time"

	"github.com/certusone/wormhole/node/pkg/internal/testutils"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	engine "github.com/certusone/wormhole/node/pkg/tss"
	"github.com/certusone/wormhole/node/pkg/tss/comm"
	"github.com/certusone/wormhole/node/pkg/tss/internal/cmd"
	"github.com/xlabs/multi-party-sig/protocols/frost/sign"
	"github.com/xlabs/tss-lib/v2/party"
	"go.uber.org/zap"
)

var cnfgPath = flag.String("cnfg", "", "path to config file in json format used to run the protocol")

var logger *zap.Logger

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ctx = testutils.MakeSupervisorContext(ctx)
	logger = supervisor.Logger(ctx)

	logger.Info("Loading KeyGenerator and GuardianStorage for DKG...")
	cnfgs := loadConfigsFromFlags()

	keygen, gst := keygeneratorSetup(cnfgs)

	keygen.Start(ctx)

	logger.Info("Setting up server...")
	srvr := createServer(gst, keygen)
	go func() {
		if err := srvr.Run(ctx); err != nil {
			logger.Error("Failed to run server", zap.Error(err))

			cancel() // stop the context if the server fails to run
		}
	}()

	time.Sleep(2 * time.Second) // wait for the server to start
	logger.Info("DKG server is running, waiting for connections...")
	if err := srvr.WaitForConnections(ctx); err != nil {
		logger.Error("Failed to wait for connections", zap.Error(err))
	}

	time.Sleep(time.Second * 3)

	logger.Info("Connections established, starting DKG...")

	run(ctx, keygen, gst, cnfgs)
}

func run(ctx context.Context, keygen engine.KeyGenerator, gst *engine.GuardianStorage, cnfgs *cmd.SetupConfigs) {
	for i := range 10 { // The loop should converge after 2~3 iterations.
		logger.Info("Making DKG Attemp", zap.Int("attempt", i))

		resChn, err := keygen.StartDKG(party.DkgTask{
			Threshold: gst.Threshold,
			Seed:      sha256.Sum256([]byte("dkg seed:" + strconv.Itoa(i))),
		})

		if err != nil {
			logger.Fatal("failed to start DKG", zap.Error(err))
		}

		var tssConfigs *party.TSSSecrets
		select {
		case tssConfigs = <-resChn:
		case <-ctx.Done():
			logger.Error("context expired before DKG finished", zap.Error(ctx.Err()))

			return
		case <-time.After(time.Second * 20):
			logger.Error("FAILED DKG. Starting an additional session")

			continue
		}

		lg := logger.With(zap.String("TrackingID", tssConfigs.TrackingID.ToString()))

		lg.Info("completed a DKG session")

		pkMarshal, err := tssConfigs.PublicKey.Clone().MarshalBinary()
		if err != nil {
			lg.Fatal("failed to marshal public key", zap.Error(err))
		}

		lg.Info("verifying resulting PK is valid for TSS usage",
			zap.String("pk", hex.EncodeToString(pkMarshal)),
		)

		lg.Info("verifying randomly chosen PK is valid for smart-contract usage")
		if !sign.PublicKeyValidForContract(tssConfigs.PublicKey) {
			continue
		}

		buff := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buff)

		if err := enc.Encode(tssConfigs); err != nil {
			lg.Fatal("failed to marshal frost configuration", zap.Error(err))
		}

		gst.TSSSecrets = buff.Bytes()
		if err := gst.SetInnerFields(); err != nil {
			lg.Fatal("failed to set inner fields of the GuardianStorage", zap.Error(err))
		}

		lg.Info("GuardianStorage updated with TSS secrets. Storing result into file", zap.Int("guardianIndex", i))

		toStore, err := json.MarshalIndent(gst, "", "  ")
		if err != nil {
			lg.Fatal("failed to marshal GuardianStorage", zap.Error(err))
		}

		fname := path.Join(cnfgs.StorageLocation, "secrets.json")

		lg.Info("Writing GuardianStorage to file", zap.String("file", fname))
		if err := os.WriteFile(fname, toStore, 0600); err != nil {
			lg.Fatal("failed to write GuardianStorage to file", zap.Error(err))
		}

		lg.Info("GuardianStorage successfully written to file", zap.String("file", fname))

		return
	}

	logger.Fatal("failed to complete DKG after 10 attempts, please check the logs for more details")
}

func createServer(gst *engine.GuardianStorage, keygen engine.KeyGenerator) comm.DirectLink {
	socketpath := "[::]:" + strconv.Itoa(gst.Self.Port)
	if gst.Self.Port == 0 {
		socketpath = "[::]:" + engine.DefaultPort
	}

	srvr, err := comm.NewServer(socketpath, logger, keygen)
	if err != nil {
		logger.Fatal("failed to create a new server", zap.Error(err))
	}
	return srvr
}

func keygeneratorSetup(cnfgs *cmd.SetupConfigs) (engine.KeyGenerator, *engine.GuardianStorage) {
	engineIds, _, err := cnfgs.IntoMaps()
	if err != nil {
		logger.Fatal("failed to convert configs engine's Identities", zap.Error(err))
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
		logger.Fatal("failed to find self in the sorted identities, please check the config file")
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
		logger.Fatal("failed to set inner fields of the GuardianStorage", zap.Error(err))
	}

	// Then Create a KeyGenerator.
	keygen, err := engine.NewKeyGenerator(gst)
	if err != nil {
		logger.Fatal("failed to create a KeyGenerator", zap.Error(err))
	}

	return keygen, gst
}

func loadConfigsFromFlags() *cmd.SetupConfigs {
	flag.Parse()

	if *cnfgPath == "" {
		flag.PrintDefaults()

		logger.Fatal("config path is empty, please provide a valid path to a config file")
	}

	logger.Info("Loading config file", zap.String("path", *cnfgPath))

	f, err := os.ReadFile(*cnfgPath)
	if err != nil {
		logger.Fatal("failed to read file, err: ", zap.Error(err))
	}

	cnfg := &cmd.SetupConfigs{}

	if err = json.Unmarshal(f, cnfg); err != nil {
		logger.Fatal("failed to unmarshal config file", zap.Error(err))
	}

	return cnfg
}
