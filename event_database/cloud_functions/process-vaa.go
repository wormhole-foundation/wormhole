package p

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"strings"
	"sync"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/pubsub"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/holiman/uint256"
)

type PubSubMessage struct {
	Data []byte `json:"data"`
}

// The keys are emitterAddress hex values, so that we can quickly check a message against the index to see if it
// meets the criteria for saving payload info: if it is a token transfer, or an NFT transfer.
var nftEmitters = map[string]string{
	// mainnet
	"0def15a24423e1edd1a5ab16f557b9060303ddbab8c803d2ee48f4b78a1cfd6b": "WnFt12ZrnzZrFZkt2xsNsaNWoQribnuQ5B5FrDbwDhD", // solana
	"0000000000000000000000006ffd7ede62328b3af38fcd61461bbfc52f5651fe": "0x6FFd7EdE62328b3Af38FCD61461Bbfc52F5651fE",  // ethereum
	"0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde": "0x5a58505a96D1dbf8dF91cB21B54419FC36e93fdE",  // bsc
	"00000000000000000000000090bbd86a6fe93d3bc3ed6335935447e75fab7fcf": "0x90bbd86a6fe93d3bc3ed6335935447e75fab7fcf",  // polygon
	"000000000000000000000000f7B6737Ca9c4e08aE573F75A97B73D7a813f5De5": "0xf7B6737Ca9c4e08aE573F75A97B73D7a813f5De5",  // avalanche
	"00000000000000000000000004952d522ff217f40b5ef3cbf659eca7b952a6c1": "0x04952d522ff217f40b5ef3cbf659eca7b952a6c1",  // oasis
	"000000000000000000000000A9c7119aBDa80d4a4E0C06C8F4d8cF5893234535": "0xA9c7119aBDa80d4a4E0C06C8F4d8cF5893234535",  // fantom
	"0000000000000000000000006dcC0484472523ed9Cdc017F711Bcbf909789284": "0x6dcC0484472523ed9Cdc017F711Bcbf909789284",  // aurora
	"000000000000000000000000b91e3638F82A1fACb28690b37e3aAE45d2c33808": "0xb91e3638F82A1fACb28690b37e3aAE45d2c33808",  // acala & karura
	"000000000000000000000000A6A377d75ca5c9052c9a77ED1e865Cc25Bd97bf3": "0xA6A377d75ca5c9052c9a77ED1e865Cc25Bd97bf3",  // celo
	"0000000000000000000000003c3c561757BAa0b78c5C025CdEAa4ee24C1dFfEf": "0x3c3c561757BAa0b78c5C025CdEAa4ee24C1dFfEf",  // klaytn

	// devnet
	"96ee982293251b48729804c8e8b24b553eb6b887867024948d2236fd37a577ab": "NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA",  // solana
	"00000000000000000000000026b4afb60d6c903165150c6f0aa14f8016be4aec": "0x26b4afb60d6c903165150c6f0aa14f8016be4aec",   // ethereum goerli
	"0000000000000000000000000fe5c51f539a651152ae461086d733777a54a134": "terra1plju286nnfj3z54wgcggd4enwaa9fgf5kgrgzl", // terra
	"000000000000000000000000cD16E5613EF35599dc82B24Cb45B5A93D779f1EE": "0xcD16E5613EF35599dc82B24Cb45B5A93D779f1EE",   // bsc
	"00000000000000000000000051a02d0dcb5e52F5b92bdAA38FA013C91c7309A9": "0x51a02d0dcb5e52F5b92bdAA38FA013C91c7309A9",   // polygon
	"000000000000000000000000D601BAf2EEE3C028344471684F6b27E789D9075D": "0xD601BAf2EEE3C028344471684F6b27E789D9075D",   // avalanche
	"000000000000000000000000C5c25B41AB0b797571620F5204Afa116A44c0ebA": "0xC5c25B41AB0b797571620F5204Afa116A44c0ebA",   // oasis
	"0000000000000000000000002b048Da40f69c8dc386a56705915f8E966fe1eba": "0x2b048Da40f69c8dc386a56705915f8E966fe1eba",   // ethereum ropsten
}
var muNFTEmitters sync.RWMutex

// NFTEmitters will be populated with lowercase addresses
var NFTEmitters = map[string]string{}

var tokenTransferEmitters = map[string]string{
	// mainnet
	"ec7372995d5cc8732397fb0ad35c0121e0eaa90d26f828a534cab54391b3a4f5": "wormDTUJ6AWPNvk59vGQbDvGJmqbDTdgWgAqcLBCgUb",                      // solana
	"0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585": "0x3ee18B2214AFF97000D974cf647E7C347E8fa585",                       // ethereum
	"0000000000000000000000007cf7b764e38a0a5e967972c1df77d432510564e2": "terra10nmmwe8r3g99a9newtqa7a75xfgs2e8z87r2sf",                     // terra
	"000000000000000000000000b6f6d86a8f9879a9c87f643768d9efc38c1da6e7": "0xB6F6D86a8f9879A9c87f643768d9efc38c1Da6E7",                       // bsc
	"0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde": "0x5a58505a96d1dbf8df91cb21b54419fc36e93fde",                       // polygon
	"0000000000000000000000000e082f06ff657d94310cb8ce8b0d9a04541d8052": "0x0e082F06FF657D94310cB8cE8B0D9a04541d8052",                       // avalanche
	"0000000000000000000000005848c791e09901b40a9ef749f2a6735b418d7564": "0x5848c791e09901b40a9ef749f2a6735b418d7564",                       // oasis
	"0000000000000000000000007c9fc5741288cdfdd83ceb07f3ea7e22618d79d2": "0x7c9fc5741288cdfdd83ceb07f3ea7e22618d79d2",                       // fantom
	"00000000000000000000000051b5123a7b0F9b2bA265f9c4C8de7D78D52f510F": "0x51b5123a7b0F9b2bA265f9c4C8de7D78D52f510F",                       // aurora
	"000000000000000000000000ae9d7fe007b3327AA64A32824Aaac52C42a6E624": "0xae9d7fe007b3327AA64A32824Aaac52C42a6E624",                       // acala & karura
	"000000000000000000000000796Dff6D74F3E27060B71255Fe517BFb23C93eed": "0x796Dff6D74F3E27060B71255Fe517BFb23C93eed",                       // celo
	"0000000000000000000000005b08ac39EAED75c0439FC750d9FE7E1F9dD0193F": "0x5b08ac39EAED75c0439FC750d9FE7E1F9dD0193F",                       // klaytn
	"a463ad028fb79679cfc8ce1efba35ac0e77b35080a1abe9bebe83461f176b0a3": "terra153366q50k7t8nn7gec00hg66crnhkdggpgdtaxltaq6xrutkkz3s992fw9", // terra2

	// devnet
	"c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f": "B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE",                     // solana
	"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16": "0x0290fb167208af455bb137780163b7b7a9a10c16",                       // ethereum goerli
	"000000000000000000000000784999135aaa8a3ca5914468852fdddbddd8789d": "terra10pyejy66429refv3g35g2t7am0was7ya7kz2a4",                     // terra
	"0000000000000000000000009dcF9D205C9De35334D646BeE44b2D2859712A09": "0x9dcF9D205C9De35334D646BeE44b2D2859712A09",                       // bsc
	"000000000000000000000000377D55a7928c046E18eEbb61977e714d2a76472a": "0x377D55a7928c046E18eEbb61977e714d2a76472a",                       // polygon
	"00000000000000000000000061E44E506Ca5659E6c0bba9b678586fA2d729756": "0x61E44E506Ca5659E6c0bba9b678586fA2d729756",                       // avalanche
	"00000000000000000000000088d8004A9BdbfD9D28090A02010C19897a29605c": "0x88d8004A9BdbfD9D28090A02010C19897a29605c",                       // oasis
	"000000000000000000000000F174F9A837536C449321df1Ca093Bb96948D5386": "0xF174F9A837536C449321df1Ca093Bb96948D5386",                       // ethereum ropsten
	"9e28beafa966b2407bffb0d48651e94972a56e69f3c0897d9e8facbdaeb98386": "terra1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrquka9l6", // terra2
}

var muTokenTransferEmitters sync.RWMutex

// TokenTransferEmitters will be populated with lowercase addresses
var TokenTransferEmitters = map[string]string{}

// this address is an emitter for BSC and Polygon.
var sharedEmitterAddress = "0000000000000000000000005a58505a96d1dbf8df91cb21b54419fc36e93fde"

type (
	TokenTransfer struct {
		PayloadId     uint8
		Amount        uint256.Int
		OriginAddress [32]byte
		OriginChain   uint16
		TargetAddress [32]byte
		TargetChain   uint16
	}
	NFTTransfer struct {
		PayloadId     uint8
		OriginAddress [32]byte
		OriginChain   uint16
		Symbol        [32]byte
		Name          [32]byte
		TokenId       uint256.Int
		URI           []byte
		TargetAddress [32]byte
		TargetChain   uint16
	}
	AssetMeta struct {
		PayloadId    uint8
		TokenAddress [32]byte
		TokenChain   uint16
		Decimals     uint8
		Symbol       [32]byte
		Name         [32]byte
	}
)

func DecodeTokenTransfer(data []byte) (*TokenTransfer, error) {
	tt := &TokenTransfer{}
	tt.PayloadId = data[0]

	reader := bytes.NewReader(data[1:])

	if err := binary.Read(reader, binary.BigEndian, &tt.Amount); err != nil {
		return nil, fmt.Errorf("failed to read Amount: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &tt.OriginAddress); err != nil {
		return nil, fmt.Errorf("failed to read OriginAddress: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &tt.OriginChain); err != nil {
		return nil, fmt.Errorf("failed to read OriginChain: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &tt.TargetAddress); err != nil {
		return nil, fmt.Errorf("failed to read TargetAddress: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &tt.TargetChain); err != nil {
		return nil, fmt.Errorf("failed to read TargetChain: %w", err)
	}

	return tt, nil
}
func DecodeNFTTransfer(data []byte) (*NFTTransfer, error) {
	nt := &NFTTransfer{}
	nt.PayloadId = data[0]

	reader := bytes.NewReader(data[1:])

	if err := binary.Read(reader, binary.BigEndian, &nt.OriginAddress); err != nil {
		return nil, fmt.Errorf("failed to read OriginAddress: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &nt.OriginChain); err != nil {
		return nil, fmt.Errorf("failed to read OriginChain: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &nt.Symbol); err != nil {
		return nil, fmt.Errorf("failed to read Symbol: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &nt.Name); err != nil {
		return nil, fmt.Errorf("failed to read Name: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &nt.TokenId); err != nil {
		return nil, fmt.Errorf("failed to read TokenId: %w", err)
	}

	// uri len
	uriLen, er := reader.ReadByte()
	if er != nil {
		return nil, fmt.Errorf("failed to read URI length")
	}

	// uri
	uri := make([]byte, int(uriLen))
	n, err := reader.Read(uri)
	if err != nil || n == 0 {
		return nil, fmt.Errorf("failed to read uri [%d]: %w", n, err)
	}
	nt.URI = uri[:n]

	if err := binary.Read(reader, binary.BigEndian, &nt.TargetAddress); err != nil {
		return nil, fmt.Errorf("failed to read : %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &nt.TargetChain); err != nil {
		return nil, fmt.Errorf("failed to read : %w", err)
	}

	return nt, nil
}

func DecodeAssetMeta(data []byte) (*AssetMeta, error) {
	am := &AssetMeta{}
	am.PayloadId = data[0]

	reader := bytes.NewReader(data[1:])

	tokenAddress := [32]byte{}
	if n, err := reader.Read(tokenAddress[:]); err != nil || n != 32 {
		return nil, fmt.Errorf("failed to read TokenAddress [%d]: %w", n, err)
	}
	am.TokenAddress = tokenAddress

	if err := binary.Read(reader, binary.BigEndian, &am.TokenChain); err != nil {
		return nil, fmt.Errorf("failed to read TokenChain: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &am.Decimals); err != nil {
		return nil, fmt.Errorf("failed to read Decimals: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &am.Symbol); err != nil {
		return nil, fmt.Errorf("failed to read Symbol: %w", err)
	}

	if err := binary.Read(reader, binary.BigEndian, &am.Name); err != nil {
		return nil, fmt.Errorf("failed to read Name: %w", err)
	}

	return am, nil
}

// TEMP: until this https://forge.certus.one/c/wormhole/+/1850 lands
func makeRowKey(emitterChain vaa.ChainID, emitterAddress vaa.Address, sequence uint64) string {
	// left-pad the sequence with zeros to 16 characters, because bigtable keys are stored lexicographically
	return fmt.Sprintf("%d:%s:%016d", emitterChain, emitterAddress, sequence)
}
func writePayloadToBigTable(ctx context.Context, rowKey string, colFam string, mutation *bigtable.Mutation, forceWrite bool) error {
	mut := mutation
	if !forceWrite {
		filter := bigtable.ChainFilters(
			bigtable.FamilyFilter(colFam),
			bigtable.ColumnFilter("PayloadId"))
		mut = bigtable.NewCondMutation(filter, nil, mutation)
	}

	err := tbl.Apply(ctx, rowKey, mut)
	if err != nil {
		log.Printf("Failed to write payload for %v to BigTable. err: %v", rowKey, err)
		return err
	}
	return nil
}
func TrimUnicodeFromByteArray(b []byte) []byte {
	// Escaped Unicode that has been observed in payload's token names and symbol:
	null := "\u0000"
	start := "\u0002"
	ack := "\u0006"
	tab := "\u0009"
	control := "\u0012"
	return bytes.Trim(b, null+start+ack+tab+control)
}

func addReceiverAddressToMutation(mut *bigtable.Mutation, ts bigtable.Timestamp, chainID uint16, hexAddress string) {
	nativeAddress := transformHexAddressToNative(vaa.ChainID(chainID), hexAddress)
	if vaa.ChainID(chainID) == vaa.ChainIDSolana {
		nativeAddress = fetchSolanaAccountOwner(nativeAddress)
	}
	if nativeAddress != "" {
		mut.Set(columnFamilies[6], "ReceiverAddress", ts, []byte(nativeAddress))
	}
}

// ProcessVAA is triggered by a PubSub message, emitted after row is saved to BigTable by guardiand
func ProcessVAA(ctx context.Context, m PubSubMessage) error {
	muNFTEmitters.Lock()
	if len(NFTEmitters) == 0 {
		for k, v := range nftEmitters {
			NFTEmitters[strings.ToLower(k)] = strings.ToLower(v)
		}
	}
	muNFTEmitters.Unlock()
	muTokenTransferEmitters.Lock()
	if len(TokenTransferEmitters) == 0 {
		for k, v := range tokenTransferEmitters {
			TokenTransferEmitters[strings.ToLower(k)] = strings.ToLower(v)
		}
	}
	muTokenTransferEmitters.Unlock()

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
	emitterHex := strings.ToLower(signedVaa.EmitterAddress.String())
	payloadId := int(signedVaa.Payload[0])

	// BSC and Polygon have the same contract address: "0x5a58505a96d1dbf8df91cb21b54419fc36e93fde".
	// The BSC contract is the NFT emitter address.
	// The Polygon contract is the token transfer emitter address.
	// Due to that, ensure that the block below only runs for token transfers by checking for chain == 4 and emitter address.
	if _, ok := TokenTransferEmitters[emitterHex]; ok && !(signedVaa.EmitterChain == 4 && signedVaa.EmitterAddress.String() == sharedEmitterAddress) {
		// figure out if it's a transfer or asset metadata

		if payloadId == 1 {
			// token transfer
			payload, decodeErr := DecodeTokenTransfer(signedVaa.Payload)
			if decodeErr != nil {
				log.Println("failed decoding payload for row ", rowKey)
				return decodeErr
			}
			log.Printf("Processing Transfer: Amount %v\n", fmt.Sprint(payload.Amount[3]))

			// save payload to bigtable, then publish a new PubSub message for further processing
			colFam := columnFamilies[2]
			mutation := bigtable.NewMutation()
			ts := bigtable.Now()
			mutation.Set(colFam, "PayloadId", ts, []byte(fmt.Sprint(payload.PayloadId)))
			// TODO: find a better way of representing amount as a string
			amount := []byte(fmt.Sprint(payload.Amount[3]))
			if payload.Amount[2] != 0 {
				log.Printf("payload.Amount is larger than uint64 for row %v", rowKey)
				amount = payload.Amount.Bytes()
			}
			targetAddressHex := hex.EncodeToString(payload.TargetAddress[:])
			mutation.Set(colFam, "Amount", ts, amount)
			mutation.Set(colFam, "OriginAddress", ts, []byte(hex.EncodeToString(payload.OriginAddress[:])))
			mutation.Set(colFam, "OriginChain", ts, []byte(fmt.Sprint(payload.OriginChain)))
			mutation.Set(colFam, "TargetAddress", ts, []byte(targetAddressHex))
			mutation.Set(colFam, "TargetChain", ts, []byte(fmt.Sprint(payload.TargetChain)))

			addReceiverAddressToMutation(mutation, ts, payload.TargetChain, targetAddressHex)

			writeErr := writePayloadToBigTable(ctx, rowKey, colFam, mutation, false)
			if writeErr != nil {
				return writeErr
			}

			// now that the payload is saved to BigTable,
			// pass along the message to the topic that will calculate TokenTransferDetails
			pubSubTokenTransferDetailsTopic.Publish(ctx, &pubsub.Message{Data: m.Data})
		} else if payloadId == 2 {
			// asset meta
			payload, decodeErr := DecodeAssetMeta(signedVaa.Payload)
			if decodeErr != nil {
				log.Println("failed decoding payload for row ", rowKey)
				return decodeErr
			}

			addressHex := hex.EncodeToString(payload.TokenAddress[:])
			chainID := vaa.ChainID(payload.TokenChain)
			nativeAddress := transformHexAddressToNative(chainID, addressHex)
			name := string(TrimUnicodeFromByteArray(payload.Name[:]))
			symbol := string(TrimUnicodeFromByteArray(payload.Symbol[:]))

			// find the CoinGecko id of this token
			coinGeckoCoinId, foundSymbol, foundName := fetchCoinGeckoCoinId(chainID, nativeAddress, symbol, name)

			// populate the symbol & name if they were blank, and we found values
			if symbol == "" && foundSymbol != "" {
				symbol = foundSymbol
			}
			if name == "" && foundName != "" {
				name = foundName
			}

			// special case for terra-classic
			if symbol == "LUNA" && chainID == vaa.ChainIDTerra {
				coinGeckoCoinId = "terra-luna"
				name = "LUNA"
			}

			log.Printf("Processing AssetMeta: Name %v, Symbol: %v, AddressHex: %v, NativeAddress: %v, CoingeckoID: %v, ChainID: %v\n", name, symbol, addressHex, nativeAddress, coinGeckoCoinId, chainID)

			// save payload to bigtable
			colFam := columnFamilies[3]
			mutation := bigtable.NewMutation()
			ts := bigtable.Now()

			mutation.Set(colFam, "PayloadId", ts, []byte(fmt.Sprint(payload.PayloadId)))
			mutation.Set(colFam, "TokenAddress", ts, []byte(addressHex))
			mutation.Set(colFam, "TokenChain", ts, []byte(fmt.Sprint(payload.TokenChain)))
			mutation.Set(colFam, "Decimals", ts, []byte(fmt.Sprint(payload.Decimals)))
			mutation.Set(colFam, "Name", ts, []byte(name))
			mutation.Set(colFam, "Symbol", ts, []byte(symbol))
			mutation.Set(colFam, "CoinGeckoCoinId", ts, []byte(coinGeckoCoinId))
			mutation.Set(colFam, "NativeAddress", ts, []byte(nativeAddress))

			writeErr := writePayloadToBigTable(ctx, rowKey, colFam, mutation, false)
			return writeErr
		} else {
			// unknown payload type
			log.Println("encountered unknown payload type for row ", rowKey)
			return nil
		}
	} else if _, ok := NFTEmitters[emitterHex]; ok {
		if payloadId == 1 {
			// NFT transfer
			payload, decodeErr := DecodeNFTTransfer(signedVaa.Payload)
			if decodeErr != nil {
				log.Println("failed decoding payload for row ", rowKey)
				return decodeErr
			}
			log.Printf("Processing NTF: Name %v, Symbol %v\n", string(TrimUnicodeFromByteArray(payload.Name[:])), string(TrimUnicodeFromByteArray(payload.Symbol[:])))

			// save payload to bigtable
			colFam := columnFamilies[4]
			mutation := bigtable.NewMutation()
			ts := bigtable.Now()

			targetAddressHex := hex.EncodeToString(payload.TargetAddress[:])
			mutation.Set(colFam, "PayloadId", ts, []byte(fmt.Sprint(payload.PayloadId)))
			mutation.Set(colFam, "OriginAddress", ts, []byte(hex.EncodeToString(payload.OriginAddress[:])))
			mutation.Set(colFam, "OriginChain", ts, []byte(fmt.Sprint(payload.OriginChain)))
			mutation.Set(colFam, "Symbol", ts, TrimUnicodeFromByteArray(payload.Symbol[:]))
			mutation.Set(colFam, "Name", ts, TrimUnicodeFromByteArray(payload.Name[:]))
			mutation.Set(colFam, "TokenId", ts, payload.TokenId.Bytes())
			mutation.Set(colFam, "URI", ts, TrimUnicodeFromByteArray(payload.URI))
			mutation.Set(colFam, "TargetAddress", ts, []byte(targetAddressHex))
			mutation.Set(colFam, "TargetChain", ts, []byte(fmt.Sprint(payload.TargetChain)))

			addReceiverAddressToMutation(mutation, ts, payload.TargetChain, targetAddressHex)

			writeErr := writePayloadToBigTable(ctx, rowKey, colFam, mutation, false)
			return writeErr
		} else {
			// unknown payload type
			log.Println("encountered unknown payload type for row ", rowKey)
			return nil
		}
	}

	// this is not a payload we are ready to decode & save. return success
	return nil
}
