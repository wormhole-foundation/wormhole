package reporter

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"go.uber.org/zap"
	"google.golang.org/api/option"

	"cloud.google.com/go/bigtable"
)

type BigTableConnectionConfig struct {
	GcpProjectID    string
	GcpInstanceName string
	GcpKeyFilePath  string
	TableName       string
}

type bigTableWriter struct {
	connectionConfig *BigTableConnectionConfig
	events           *AttestationEventReporter
}

// rowKey returns a string with the input vales delimited by colons.
func makeRowKey(emitterChain vaa.ChainID, emitterAddress vaa.Address, sequence uint64) string {
	return fmt.Sprintf("%d:%s:%d", emitterChain, emitterAddress, sequence)
}

func BigTableWriter(events *AttestationEventReporter, connectionConfig *BigTableConnectionConfig) func(ctx context.Context) error {
	return func(ctx context.Context) error {

		e := &bigTableWriter{events: events, connectionConfig: connectionConfig}

		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}

		errC := make(chan error)
		logger := supervisor.Logger(ctx)

		client, err := bigtable.NewClient(ctx,
			e.connectionConfig.GcpProjectID,
			e.connectionConfig.GcpInstanceName,
			option.WithCredentialsFile(e.connectionConfig.GcpKeyFilePath))
		if err != nil {
			return fmt.Errorf("failed to create BigTable client: %w", err)
		}
		tbl := client.Open(e.connectionConfig.TableName)

		// call to subscribe to event channels
		sub := e.events.Subscribe()
		logger.Info("subscribed to AttestationEvents")

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-sub.Channels.MessagePublicationC:
					colFam := "MessagePublication"
					mutation := bigtable.NewMutation()
					ts := bigtable.Now()

					mutation.Set(colFam, "Version", ts, []byte(fmt.Sprint(msg.VAA.Version)))
					mutation.Set(colFam, "GuardianSetIndex", ts, []byte(fmt.Sprint(msg.VAA.GuardianSetIndex)))
					mutation.Set(colFam, "Timestamp", ts, []byte(ts.Time().String()))
					mutation.Set(colFam, "Nonce", ts, []byte(fmt.Sprint(msg.VAA.Nonce)))
					mutation.Set(colFam, "EmitterChain", ts, []byte(msg.VAA.EmitterChain.String()))
					mutation.Set(colFam, "EmitterAddress", ts, []byte(msg.VAA.EmitterAddress.String()))
					mutation.Set(colFam, "Sequence", ts, []byte(fmt.Sprint(msg.VAA.Sequence)))
					mutation.Set(colFam, "InitiatingTxID", ts, []byte(msg.InitiatingTxID.Hex()))
					mutation.Set(colFam, "Payload", ts, msg.VAA.Payload)

					mutation.Set(colFam, "ReporterHostname", ts, []byte(hostname))

					// filter to see if there is a row with this rowKey, and has a value for EmitterAddress
					filter := bigtable.ChainFilters(
						bigtable.FamilyFilter(colFam),
						bigtable.ColumnFilter("EmitterAddress"))
					conditionalMutation := bigtable.NewCondMutation(filter, nil, mutation)

					rowKey := makeRowKey(msg.VAA.EmitterChain, msg.VAA.EmitterAddress, msg.VAA.Sequence)
					err := tbl.Apply(ctx, rowKey, conditionalMutation)
					if err != nil {
						logger.Warn("Failed to write message publication to BigTable",
							zap.String("rowKey", rowKey),
							zap.String("columnFamily", colFam),
							zap.Error(err))
						errC <- err
					}
				case msg := <-sub.Channels.VerifiedSignatureC:
					colFam := "Signatures"
					mutation := bigtable.NewMutation()
					ts := bigtable.Now()
					addrHex := msg.GuardianAddress.Hex()

					mutation.Set(colFam, addrHex, ts, msg.Signature)

					// filter to see if this row already has this signature.
					filter := bigtable.ChainFilters(
						bigtable.FamilyFilter(colFam),
						bigtable.ColumnFilter(addrHex))
					conditionalMutation := bigtable.NewCondMutation(filter, nil, mutation)

					rowKey := makeRowKey(msg.EmitterChain, msg.EmitterAddress, msg.Sequence)
					err := tbl.Apply(ctx, rowKey, conditionalMutation)
					if err != nil {
						logger.Warn("Failed to write signature to BigTable",
							zap.String("rowKey", rowKey),
							zap.String("columnFamily", colFam),
							zap.Error(err))
						errC <- err
					}
				case msg := <-sub.Channels.VAAStateUpdateC:
					colFam := "VAAState"
					mutation := bigtable.NewMutation()
					ts := bigtable.Now()

					buf := new(bytes.Buffer)
					vaa.MustWrite(buf, binary.BigEndian, uint8(len(msg.Signatures)))
					for _, sig := range msg.Signatures {
						vaa.MustWrite(buf, binary.BigEndian, sig.Index)
						buf.Write(sig.Signature[:])
					}
					mutation.Set(colFam, "Signatures", ts, buf.Bytes())
					// TODO: conditional mutation that considers number of signatures in the VAA.

					rowKey := makeRowKey(msg.EmitterChain, msg.EmitterAddress, msg.Sequence)
					err := tbl.Apply(ctx, rowKey, mutation)
					if err != nil {
						logger.Warn("Failed to write VAA update to BigTable",
							zap.String("rowKey", rowKey),
							zap.String("columnFamily", colFam),
							zap.Error(err))
						errC <- err
					}
				case msg := <-sub.Channels.VAAQuorumC:
					colFam := "QuorumState"
					mutation := bigtable.NewMutation()
					ts := bigtable.Now()
					// TODO - record signed VAAs from gossip messages.

					b, marshalErr := msg.Marshal()
					if marshalErr != nil {
						logger.Error("failed to marshal VAAQuorum VAA.")
					}
					mutation.Set(colFam, "SignedVAA", ts, b)

					// filter to see if this row already has the signature.
					filter := bigtable.ChainFilters(
						bigtable.FamilyFilter(colFam),
						bigtable.ColumnFilter("SignedVAA"))
					conditionalMutation := bigtable.NewCondMutation(filter, nil, mutation)

					rowKey := makeRowKey(msg.EmitterChain, msg.EmitterAddress, msg.Sequence)
					err := tbl.Apply(ctx, rowKey, conditionalMutation)
					if err != nil {
						logger.Warn("Failed to write persistence info to BigTable",
							zap.String("rowKey", rowKey),
							zap.String("columnFamily", colFam),
							zap.Error(err))
						errC <- err
					}
				}
			}
		}()

		select {
		case <-ctx.Done():
			e.events.Unsubscribe(sub.ClientId)
			if err = client.Close(); err != nil {
				logger.Error("Could not close BigTable client", zap.Error(err))
			}
			return ctx.Err()
		case err := <-errC:
			logger.Error("bigtablewriter encountered an error", zap.Error(err))

			e.events.Unsubscribe(sub.ClientId)

			// try to close the connection before returning
			if closeErr := client.Close(); closeErr != nil {
				logger.Error("Could not close BigTable client", zap.Error(closeErr))
			}

			return err
		}
	}
}
