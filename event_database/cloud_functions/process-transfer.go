package p

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/certusone/wormhole/node/pkg/vaa"

	"cloud.google.com/go/bigtable"
)

// ProcessTransfer is triggered by a PubSub message, once a TokenTransferPayload is written to a row.
func ProcessTransfer(ctx context.Context, m PubSubMessage) error {
	data := string(m.Data)
	if data == "" {
		return fmt.Errorf("no data to process in message")
	}

	signedVaa, err := vaa.Unmarshal(m.Data)
	if err != nil {
		log.Println("failed Unmarshaling VAA")
		return err
	}

	// create the bigtable identifier from the VAA data
	rowKey := makeRowKey(signedVaa.EmitterChain, signedVaa.EmitterAddress, signedVaa.Sequence)
	row, err := tbl.ReadRow(ctx, rowKey)
	if err != nil {
		log.Fatalf("Could not read row with key %s: %v", rowKey, err)
	}

	// get the payload data for this transfer
	var tokenAddress string
	var amount string
	for _, item := range row[columnFamilies[2]] {
		switch item.Column {
		case "TokenTransferPayload:OriginAddress":
			tokenAddress = string(item.Value)
		case "TokenTransferPayload:Amount":
			amount = string(item.Value)
		}
	}

	// lookup the asset meta for this transfer.
	// start with the emitterChain:emitterAddress prefix
	keyParts := strings.Split(row.Key(), ":")
	keyPrefix := strings.Join(keyParts[:2], ":")

	// find an AssetMeta message that matches the TokenAddress of the transfer
	var result bigtable.Row
	queryErr := tbl.ReadRows(ctx, bigtable.PrefixRange(keyPrefix), func(row bigtable.Row) bool {
		result = row
		return true
	}, bigtable.RowFilter(
		bigtable.ChainFilters(
			bigtable.FamilyFilter(columnFamilies[3]),
			bigtable.ColumnFilter("TokenAddress"),
			bigtable.ValueFilter(tokenAddress),
		)))

	if queryErr != nil {
		log.Fatalf("failed to read rows: %v", queryErr)
	}
	// now get the entire row
	assetMetaRow, assetMetaErr := tbl.ReadRow(ctx, result.Key())
	if assetMetaErr != nil {
		log.Fatalf("Could not read row with key %s: %v", rowKey, assetMetaErr)
	}
	if _, ok := assetMetaRow[columnFamilies[3]]; !ok {
		log.Println("did not find AssetMeta matching TokenAddress", tokenAddress)
		return fmt.Errorf("did not find AssetMeta matching TokenAddress %v", tokenAddress)
	}

	// get AssetMeta values
	var decimals int
	var symbol string
	var name string
	for _, item := range assetMetaRow[columnFamilies[3]] {
		switch item.Column {
		case "AssetMetaPayload:Decimals":
			decimalStr := string(item.Value)
			dec, err := strconv.Atoi(decimalStr)
			if err != nil {
				log.Fatalf("failed parsing decimals of row %v", assetMetaRow.Key())
			}
			decimals = dec
		case "AssetMetaPayload:Symbol":
			symbol = string(item.Value)
		case "AssetMetaPayload:Name":
			name = string(item.Value)
		}
	}

	// transfers created by the bridge UI will have at most 8 decimals.
	if decimals > 8 {
		decimals = 8
	}
	// ensure amount string is long enough
	if len(amount) < decimals {
		amount = fmt.Sprintf("%0*v", decimals, amount)
	}

	intAmount := amount[:len(amount)-decimals]
	decAmount := amount[len(amount)-decimals:]
	calculatedAmount := intAmount + "." + decAmount
	log.Printf("\ndetermined calculatedAmount: %v for row %v", calculatedAmount, rowKey)

	// write to BigTable
	colFam := columnFamilies[5]
	mutation := bigtable.NewMutation()
	ts := bigtable.Now()

	mutation.Set(colFam, "Amount", ts, []byte(calculatedAmount))
	mutation.Set(colFam, "OriginSymbol", ts, []byte(symbol))
	mutation.Set(colFam, "OriginName", ts, []byte(name))

	// TODO - find the symbol & name of the asset on the target chain?
	// mutation.Set(colFam, "TargetSymbol", ts, []byte())
	// mutation.Set(colFam, "TargetName", ts, []byte())

	// conditional mutation - don't write if row already has an Amount value.
	filter := bigtable.ChainFilters(
		bigtable.FamilyFilter(colFam),
		bigtable.ColumnFilter("Amount"))
	conditionalMutation := bigtable.NewCondMutation(filter, nil, mutation)

	writeErr := tbl.Apply(ctx, rowKey, conditionalMutation)
	if writeErr != nil {
		log.Printf("\nFailed to write TokenTransferDetails for %v to BigTable. err: %v", rowKey, writeErr)
		return writeErr
	}
	log.Println("done writing TokenTransferDetails to bigtable", rowKey)

	// success
	return nil
}
