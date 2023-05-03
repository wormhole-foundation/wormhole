package reporter

import (
	"context"
	"fmt"
	"os"

	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/api/option"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/pubsub"
)

type BigTableConnectionConfig struct {
	GcpProjectID    string
	GcpInstanceName string
	GcpKeyFilePath  string
	TableName       string
	TopicName       string
}

type bigTableWriter struct {
	connectionConfig *BigTableConnectionConfig
	events           *AttestationEventReporter
}

// rowKey returns a string with the input vales delimited by colons.
func MakeRowKey(emitterChain vaa.ChainID, emitterAddress vaa.Address, sequence uint64) string {
	// left-pad the sequence with zeros to 16 characters, because bigtable keys are stored lexicographically
	return fmt.Sprintf("%d:%s:%016d", emitterChain, emitterAddress, sequence)
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

		pubsubClient, err := pubsub.NewClient(ctx,
			e.connectionConfig.GcpProjectID,
			option.WithCredentialsFile(e.connectionConfig.GcpKeyFilePath))
		if err != nil {
			logger.Error("failed to create GCP PubSub client", zap.Error(err))
			return fmt.Errorf("failed to create GCP PubSub client: %w", err)
		}
		logger.Info("GCP PubSub.NewClient initialized")

		pubsubTopic := pubsubClient.Topic(e.connectionConfig.TopicName)
		logger.Info("GCP PubSub.Topic initialized",
			zap.String("Topic", e.connectionConfig.TopicName))
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

					rowKey := MakeRowKey(msg.VAA.EmitterChain, msg.VAA.EmitterAddress, msg.VAA.Sequence)
					err := tbl.Apply(ctx, rowKey, conditionalMutation)
					if err != nil {
						logger.Error("Failed to write message publication to BigTable",
							zap.String("rowKey", rowKey),
							zap.String("columnFamily", colFam),
							zap.Error(err))
						errC <- err
					}
				case msg := <-sub.Channels.VAAQuorumC:
					colFam := "QuorumState"
					mutation := bigtable.NewMutation()
					ts := bigtable.Now()

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

					rowKey := MakeRowKey(msg.EmitterChain, msg.EmitterAddress, msg.Sequence)
					err := tbl.Apply(ctx, rowKey, conditionalMutation)
					if err != nil {
						logger.Error("Failed to write persistence info to BigTable",
							zap.String("rowKey", rowKey),
							zap.String("columnFamily", colFam),
							zap.Error(err))
						errC <- err
					}
					publishResult := pubsubTopic.Publish(ctx, &pubsub.Message{Data: b})
					if _, err = publishResult.Get(ctx); err != nil {
						logger.Error("Failed getting GCP PubSub publish reciept",
							zap.String("rowKey", rowKey),
							zap.Error(err))
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
			if pubsubErr := pubsubClient.Close(); pubsubErr != nil {
				logger.Error("Could not close GCP PubSub client", zap.Error(pubsubErr))
			}
			return ctx.Err()
		case err := <-errC:
			logger.Error("bigtablewriter encountered an error", zap.Error(err))

			e.events.Unsubscribe(sub.ClientId)

			// try to close the connection before returning
			if closeErr := client.Close(); closeErr != nil {
				logger.Error("Could not close BigTable client", zap.Error(closeErr))
			}
			if pubsubErr := pubsubClient.Close(); pubsubErr != nil {
				logger.Error("Could not close GCP PubSub client", zap.Error(pubsubErr))
			}

			return err
		}
	}
}
