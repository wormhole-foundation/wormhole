package p

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"math"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types/bech32"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"cloud.google.com/go/bigtable"

	"github.com/gagliardetto/solana-go"
)

var tokenAddressExceptions = map[string]string{
	// terra native tokens do not have a bech32 address like cw20s do, handle them manually.
	// terra (classic)
	"0100000000000000000000000000000000000000000000000000000075757364": "uusd",
	"010000000000000000000000000000000000000000000000000000756c756e61": "uluna",
	// near
	"0000000000000000000000000000000000000000000000000000000000000000": "near",
	"67499b7b8f58eaeb3cd81aea1d1ce9f7f722fd7750ceb2bed13e255073c25e2a": "token.sweat",
	// terra2
	"01fa6c6fbc36d8c245b0a852a43eb5d644e8b4c477b27bfab9537c10945939da": "uluna",
	// xpla
	"017ce8aec5af3bb3ac0158d49771d4c8feba2e54a614fa2a1c0c95e9c4c37185": "axpla",
}

// returns a pair of dates before and after the input time.
// useful for creating a time rage for querying historical price APIs.
func rangeFromTime(t time.Time, hours int) (start time.Time, end time.Time) {
	duration := time.Duration(hours) * time.Hour
	return t.Add(-duration), t.Add(duration)
}

func transformHexAddressToNative(chain vaa.ChainID, address string) string {
	switch chain {
	case vaa.ChainIDSolana:
		addr, err := hex.DecodeString(address)
		if err != nil {
			log.Fatalf("failed to decode solana string: %v", err)
		}
		if len(addr) != 32 {
			log.Fatalf("address must be 32 bytes. address: %v", address)
		}
		solPk := solana.PublicKeyFromBytes(addr[:])
		return solPk.String()
	case vaa.ChainIDEthereum,
		vaa.ChainIDBSC,
		vaa.ChainIDPolygon,
		vaa.ChainIDAvalanche,
		vaa.ChainIDOasis,
		vaa.ChainIDEthereumRopsten,
		vaa.ChainIDAurora,
		vaa.ChainIDFantom,
		vaa.ChainIDKarura,
		vaa.ChainIDAcala,
		vaa.ChainIDKlaytn,
		vaa.ChainIDCelo,
		vaa.ChainIDMoonbeam:
		addr := fmt.Sprintf("0x%v", address[(len(address)-40):])
		return addr
	case vaa.ChainIDTerra:
		// handle terra native assets manually
		if val, ok := tokenAddressExceptions[address]; ok {
			return val
		}
		return humanAddressTerra(address)
	case vaa.ChainIDAlgorand:
		assetId := big.Int{}
		_, ok := assetId.SetString(address, 16)
		if ok {
			return assetId.String()
		}
		return address
	case vaa.ChainIDNear:
		if val, ok := tokenAddressExceptions[address]; ok {
			return val
		}
		// TODO for now use hex/wormhole address string, we'll need to do a contract query to get the native address
		return address
	case vaa.ChainIDTerra2:
		// handle terra2 native assets manually
		if val, ok := tokenAddressExceptions[address]; ok {
			return val
		}
		// terra2 has 32 byte addresses for contracts and 20 for wallets
		if isLikely20ByteTerra(address) {
			return humanAddressTerra(address)
		}
		// TODO for now use hex/wormhole address string, we'll need to do a contract query to get the native address
		return address
	case vaa.ChainIDAptos:
		if val, ok := tokenAddressExceptions[address]; ok {
			return val
		}
		// TODO for now use hex/wormhole address string, we'll need to do a contract query to get the native address
		return address
	case vaa.ChainIDXpla:
		if val, ok := tokenAddressExceptions[address]; ok {
			return val
		}
		// TODO for now use hex/wormhole address string, we'll need to do a contract query to get the native address
		return address
	default:
		log.Println("cannot process address for unknown chain: ", chain)
		return ""
	}
}

func isLikely20ByteTerra(address string) bool {
	return strings.HasPrefix(address, "00000000000000000000")
}

func humanAddressTerra(address string) string {
	trimmed := address[(len(address) - 40):]
	data, decodeErr := hex.DecodeString(trimmed)
	if decodeErr != nil {
		fmt.Printf("failed to decode unpadded string: %v\n", decodeErr)
	}
	encodedAddr, convertErr := bech32.ConvertAndEncode("terra", data)
	if convertErr != nil {
		fmt.Println("convert error from cosmos bech32. err", convertErr)
	}
	return encodedAddr
}

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
	var tokenChain vaa.ChainID
	var amount string
	for _, item := range row[columnFamilies[2]] {
		switch item.Column {
		case "TokenTransferPayload:OriginAddress":
			tokenAddress = string(item.Value)
		case "TokenTransferPayload:OriginChain":
			chainInt, _ := strconv.ParseUint(string(item.Value), 10, 32)
			chainID := vaa.ChainID(chainInt)
			tokenChain = chainID
		case "TokenTransferPayload:Amount":
			amount = string(item.Value)
		}
	}

	// lookup the asset meta for this transfer.
	// find an AssetMeta message that matches the OriginChain & TokenAddress of the transfer
	var result bigtable.Row
	chainIDPrefix := fmt.Sprintf("%d", tokenChain) // create a string containing the tokenChain chainID, ie "2"
	queryErr := tbl.ReadRows(ctx, bigtable.PrefixRange(chainIDPrefix), func(row bigtable.Row) bool {
		result = row
		return true
	}, bigtable.RowFilter(
		bigtable.ChainFilters(
			bigtable.FamilyFilter(columnFamilies[3]),
			bigtable.ColumnFilter("TokenAddress"),
			bigtable.ValueFilter(tokenAddress),
		),
	))

	if queryErr != nil {
		log.Fatalf("failed to read rows: %v", queryErr)
	}
	if result == nil {
		log.Printf("did not find AssetMeta row for tokenAddress: %v. Transfer rowKey: %v\n", tokenAddress, rowKey)
		return fmt.Errorf("did not find AssetMeta row for tokenAddress %v", tokenAddress)
	}
	// now get the entire row
	assetMetaRow, assetMetaErr := tbl.ReadRow(ctx, result.Key(), bigtable.RowFilter(bigtable.LatestNFilter(1)))
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
	var coinId string
	var nativeTokenAddress string
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
		case "AssetMetaPayload:CoinGeckoCoinId":
			coinId = string(item.Value)
		case "AssetMetaPayload:NativeAddress":
			nativeTokenAddress = string(item.Value)
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

	timestamp := signedVaa.Timestamp.UTC()
	price, _ := fetchCoinGeckoPrice(coinId, timestamp)

	// convert the amount string so it can be used for math
	amountFloat, convErr := strconv.ParseFloat(calculatedAmount, 64)
	if convErr != nil {
		log.Fatalf("failed parsing calculatedAmount '%v' to float64. err %v", calculatedAmount, convErr)
	}
	notional := amountFloat * price
	notionalStr := fmt.Sprintf("%f", notional)

	log.Printf("processed transfer of $%0.2f = %v %v * $%0.2f\n", notional, calculatedAmount, symbol, price)

	// write to BigTable
	colFam := columnFamilies[5]
	mutation := bigtable.NewMutation()
	ts := bigtable.Now()

	mutation.Set(colFam, "Amount", ts, []byte(calculatedAmount))
	mutation.Set(colFam, "Decimals", ts, []byte(fmt.Sprint(decimals)))
	var notionalbuf [8]byte
	binary.BigEndian.PutUint64(notionalbuf[:], math.Float64bits(notional))
	mutation.Set(colFam, "NotionalUSD", ts, notionalbuf[:])
	mutation.Set(colFam, "NotionalUSDStr", ts, []byte(notionalStr))
	var priceBuf [8]byte
	binary.BigEndian.PutUint64(priceBuf[:], math.Float64bits(price))
	mutation.Set(colFam, "TokenPriceUSD", ts, priceBuf[:])
	mutation.Set(colFam, "TokenPriceUSDStr", ts, []byte(fmt.Sprintf("%f", price)))
	mutation.Set(colFam, "TransferTimestamp", ts, []byte(timestamp.String()))
	mutation.Set(colFam, "OriginSymbol", ts, []byte(symbol))
	mutation.Set(colFam, "OriginName", ts, []byte(name))
	mutation.Set(colFam, "OriginTokenAddress", ts, []byte(nativeTokenAddress))
	mutation.Set(colFam, "CoinGeckoCoinId", ts, []byte(coinId))

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
		log.Printf("Failed to write TokenTransferDetails for %v to BigTable. err: %v\n", rowKey, writeErr)
		return writeErr
	}

	// success
	return nil
}
