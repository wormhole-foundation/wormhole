package p

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

// shared code for the various functions, primarily response formatting.

// client is a global Bigtable client, to avoid initializing a new client for
// every request.
var client *bigtable.Client
var clientOnce sync.Once
var tbl *bigtable.Table

var storageClient *storage.Client
var cacheBucketName string
var cacheBucket *storage.BucketHandle

var pubsubClient *pubsub.Client
var pubSubTokenTransferDetailsTopic *pubsub.Topic

var coinGeckoCoins = map[string][]CoinGeckoCoin{}
var solanaTokens = map[string]SolanaToken{}

var releaseDay = time.Date(2021, 9, 13, 0, 0, 0, 0, time.UTC)

var loadCache = true

// init runs during cloud function initialization. So, this will only run during an
// an instance's cold start.
// https://cloud.google.com/functions/docs/bestpractices/networking#accessing_google_apis
func init() {
	defer timeTrack(time.Now(), "init")
	clientOnce.Do(func() {
		// Declare a separate err variable to avoid shadowing client.
		var err error
		project := os.Getenv("GCP_PROJECT")
		instance := os.Getenv("BIGTABLE_INSTANCE")
		client, err = bigtable.NewClient(context.Background(), project, instance)
		if err != nil {
			// http.Error(w, "Error initializing client", http.StatusInternalServerError)
			log.Printf("bigtable.NewClient error: %v", err)

			return
		}

		// create the topic that will be published to after decoding token transfer payloads
		tokenTransferDetailsTopic := os.Getenv("PUBSUB_TOKEN_TRANSFER_DETAILS_TOPIC")
		if tokenTransferDetailsTopic != "" {
			var pubsubErr error
			pubsubClient, pubsubErr = pubsub.NewClient(context.Background(), project)
			if pubsubErr != nil {
				log.Printf("pubsub.NewClient error: %v", pubsubErr)
				return
			}
			pubSubTokenTransferDetailsTopic = pubsubClient.Topic(tokenTransferDetailsTopic)
			// fetch the token lists once at start up
			coinGeckoCoins = fetchCoinGeckoCoins()
			solanaTokens = fetchSolanaTokenList()
		}
	})
	tbl = client.Open("v2Events")

	cacheBucketName = os.Getenv("CACHE_BUCKET")
	if cacheBucketName != "" {
		// Create storage client.
		var err error
		storageClient, err = storage.NewClient(context.Background())
		if err != nil {
			log.Fatalf("Failed to create storage client: %v", err)
		}
		cacheBucket = storageClient.Bucket(cacheBucketName)
	}

	tokenAllowlistFilePath := os.Getenv("TOKEN_ALLOWLIST")
	if tokenAllowlistFilePath != "" {
		loadJsonToInterface(context.Background(), tokenAllowlistFilePath, &sync.RWMutex{}, &tokenAllowlist)
	}

	loadCacheStr := os.Getenv("LOAD_CACHE")
	if val, err := strconv.ParseBool(loadCacheStr); err == nil {
		loadCache = val
		log.Printf("loadCache set to %v\n", loadCache)
	}
}

func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

// reads the specified file from the CACHE_BUCKET and unmarshals the json into the supplied interface.
func loadJsonToInterface(ctx context.Context, filePath string, mutex *sync.RWMutex, cacheMap interface{}) {
	if cacheBucket == nil {
		log.Println("no cacheBucket supplied, not going to read cache")
		return
	}
	defer timeTrack(time.Now(), fmt.Sprintf("reading %v", filePath))
	mutex.Lock()

	reader, readErr := cacheBucket.Object(filePath).NewReader(ctx)
	if readErr != nil {
		log.Printf("Failed reading %v in GCS. err: %v", filePath, readErr)
	}
	defer reader.Close()
	fileData, err := io.ReadAll(reader)
	if err != nil {
		log.Printf("loadJsonToInterface: unable to read data. file %q: %v", filePath, err)
	}
	unmarshalErr := json.Unmarshal(fileData, &cacheMap)
	mutex.Unlock()
	if unmarshalErr != nil {
		log.Printf("failed unmarshaling %v, err: %v", filePath, unmarshalErr)
	}
}

// writes the supplied interface to the CACHE_BUCKET/filePath.
func persistInterfaceToJson(ctx context.Context, filePath string, mutex *sync.RWMutex, cacheMap interface{}) {
	if cacheBucket == nil {
		log.Println("no cacheBucket supplied, not going to persist cache")
		return
	}
	defer timeTrack(time.Now(), fmt.Sprintf("writing %v", filePath))
	mutex.Lock()

	cacheBytes, marshalErr := json.MarshalIndent(cacheMap, "", "  ")
	if marshalErr != nil {
		log.Fatal("failed marshaling cacheMap.", marshalErr)
	}
	wc := cacheBucket.Object(filePath).NewWriter(ctx)
	reader := bytes.NewReader(cacheBytes)
	if _, writeErr := io.Copy(wc, reader); writeErr != nil {
		log.Printf("failed writing to file %v, err: %v", filePath, writeErr)
	}
	mutex.Unlock()
	if err := wc.Close(); err != nil {
		log.Printf("Writer.Close with error: %v", err)
	}
}

var columnFamilies = []string{
	"MessagePublication",
	"QuorumState",
	"TokenTransferPayload",
	"AssetMetaPayload",
	"NFTTransferPayload",
	"TokenTransferDetails",
	"ChainDetails",
}
var messagePubFam = columnFamilies[0]
var quorumStateFam = columnFamilies[1]
var transferPayloadFam = columnFamilies[2]
var metaPayloadFam = columnFamilies[3]
var nftPayloadFam = columnFamilies[4]
var transferDetailsFam = columnFamilies[5]
var chainDetailsFam = columnFamilies[6]

type (
	// Summary is MessagePublication data & QuorumState data
	Summary struct {
		EmitterChain    string
		EmitterAddress  string
		Sequence        string
		InitiatingTxID  string
		Payload         []byte
		SignedVAABytes  []byte
		QuorumTime      string
		TransferDetails *TransferDetails
	}
	// Details is a Summary extended with all the post-processing ColumnFamilies
	Details struct {
		Summary
		SignedVAA            *vaa.VAA
		TokenTransferPayload *TokenTransferPayload
		AssetMetaPayload     *AssetMetaPayload
		NFTTransferPayload   *NFTTransferPayload
		ChainDetails         *ChainDetails
	}
	// The following structs match the ColumnFamiles they are named after
	TokenTransferPayload struct {
		Amount        string
		OriginAddress string
		OriginChain   string
		TargetAddress string
		TargetChain   string
	}
	AssetMetaPayload struct {
		TokenAddress    string
		TokenChain      string
		Decimals        string
		Symbol          string
		Name            string
		CoinGeckoCoinId string
		NativeAddress   string
	}
	NFTTransferPayload struct {
		OriginAddress string
		OriginChain   string
		Symbol        string
		Name          string
		TokenId       string
		URI           string
		TargetAddress string
		TargetChain   string
	}
	TransferDetails struct {
		Amount             string
		Decimals           string
		NotionalUSDStr     string
		TokenPriceUSDStr   string
		TransferTimestamp  string
		OriginSymbol       string
		OriginName         string
		OriginTokenAddress string
	}
	ChainDetails struct {
		SenderAddress   string
		ReceiverAddress string
	}
)

func chainIdStringToType(chainId string) vaa.ChainID {
	switch chainId {
	case "1":
		return vaa.ChainIDSolana
	case "2":
		return vaa.ChainIDEthereum
	case "3":
		return vaa.ChainIDTerra
	case "4":
		return vaa.ChainIDBSC
	case "5":
		return vaa.ChainIDPolygon
	case "6":
		return vaa.ChainIDAvalanche
	case "7":
		return vaa.ChainIDOasis
	case "8":
		return vaa.ChainIDAlgorand
	case "9":
		return vaa.ChainIDAurora
	case "10":
		return vaa.ChainIDFantom
	case "11":
		return vaa.ChainIDKarura
	case "12":
		return vaa.ChainIDAcala
	case "13":
		return vaa.ChainIDKlaytn
	case "14":
		return vaa.ChainIDCelo
	case "18":
		return vaa.ChainIDTerra2
	case "10001":
		return vaa.ChainIDEthereumRopsten
	}
	return vaa.ChainIDUnset
}

func chainIDToNumberString(c vaa.ChainID) string {
	return strconv.FormatUint(uint64(c), 10)
}

func makeSummary(row bigtable.Row) *Summary {
	summary := &Summary{}
	if _, ok := row[messagePubFam]; ok {

		for _, item := range row[messagePubFam] {
			switch item.Column {
			case "MessagePublication:InitiatingTxID":
				summary.InitiatingTxID = string(item.Value)
			case "MessagePublication:Payload":
				summary.Payload = item.Value
			case "MessagePublication:EmitterChain":
				summary.EmitterChain = string(item.Value)
			case "MessagePublication:EmitterAddress":
				summary.EmitterAddress = string(item.Value)
			case "MessagePublication:Sequence":
				summary.Sequence = string(item.Value)
			}
		}
	} else {
		// Some rows have a QuorumState, but no MessagePublication,
		// so populate Summary values from the rowKey.
		keyParts := strings.Split(row.Key(), ":")
		chainId := chainIdStringToType(keyParts[0])
		summary.EmitterChain = chainId.String()
		summary.EmitterAddress = keyParts[1]
		seq := strings.TrimLeft(keyParts[2], "0")
		if seq == "" {
			seq = "0"
		}
		summary.Sequence = seq
	}
	if _, ok := row[quorumStateFam]; ok {
		item := row[quorumStateFam][0]
		summary.SignedVAABytes = item.Value
		summary.QuorumTime = item.Timestamp.Time().String()
	}
	if _, ok := row[transferDetailsFam]; ok {
		transferDetails := &TransferDetails{}
		for _, item := range row[transferDetailsFam] {
			switch item.Column {
			case "TokenTransferDetails:Amount":
				transferDetails.Amount = string(item.Value)
			case "TokenTransferDetails:Decimals":
				transferDetails.Decimals = string(item.Value)
			case "TokenTransferDetails:NotionalUSDStr":
				transferDetails.NotionalUSDStr = string(item.Value)
			case "TokenTransferDetails:TokenPriceUSDStr":
				transferDetails.TokenPriceUSDStr = string(item.Value)
			case "TokenTransferDetails:TransferTimestamp":
				transferDetails.TransferTimestamp = string(item.Value)
			case "TokenTransferDetails:OriginSymbol":
				transferDetails.OriginSymbol = string(item.Value)
			case "TokenTransferDetails:OriginName":
				transferDetails.OriginName = string(item.Value)
			case "TokenTransferDetails:OriginTokenAddress":
				transferDetails.OriginTokenAddress = string(item.Value)
			}
		}
		summary.TransferDetails = transferDetails
	}
	return summary
}

func makeDetails(row bigtable.Row) *Details {
	deets := &Details{}
	sum := makeSummary(row)
	deets.Summary = Summary{
		EmitterChain:    sum.EmitterChain,
		EmitterAddress:  sum.EmitterAddress,
		Sequence:        sum.Sequence,
		InitiatingTxID:  sum.InitiatingTxID,
		Payload:         sum.Payload,
		SignedVAABytes:  sum.SignedVAABytes,
		QuorumTime:      sum.QuorumTime,
		TransferDetails: sum.TransferDetails,
	}

	if _, ok := row[quorumStateFam]; ok {
		item := row[quorumStateFam][0]
		deets.SignedVAA, _ = vaa.Unmarshal(item.Value)
	}
	if _, ok := row[transferPayloadFam]; ok {
		tokenTransferPayload := &TokenTransferPayload{}
		for _, item := range row[transferPayloadFam] {
			switch item.Column {
			case "TokenTransferPayload:Amount":
				tokenTransferPayload.Amount = string(item.Value)
			case "TokenTransferPayload:OriginAddress":
				tokenTransferPayload.OriginAddress = string(item.Value)
			case "TokenTransferPayload:OriginChain":
				tokenTransferPayload.OriginChain = string(item.Value)
			case "TokenTransferPayload:TargetAddress":
				tokenTransferPayload.TargetAddress = string(item.Value)
			case "TokenTransferPayload:TargetChain":
				tokenTransferPayload.TargetChain = string(item.Value)
			}
		}
		deets.TokenTransferPayload = tokenTransferPayload
	}
	if _, ok := row[metaPayloadFam]; ok {
		assetMetaPayload := &AssetMetaPayload{}
		for _, item := range row[metaPayloadFam] {
			switch item.Column {
			case "AssetMetaPayload:TokenAddress":
				assetMetaPayload.TokenAddress = string(item.Value)
			case "AssetMetaPayload:TokenChain":
				assetMetaPayload.TokenChain = string(item.Value)
			case "AssetMetaPayload:Decimals":
				assetMetaPayload.Decimals = string(item.Value)
			case "AssetMetaPayload:Symbol":
				assetMetaPayload.Symbol = string(item.Value)
			case "AssetMetaPayload:Name":
				assetMetaPayload.Name = string(item.Value)
			case "AssetMetaPayload:CoinGeckoCoinId":
				assetMetaPayload.CoinGeckoCoinId = string(item.Value)
			case "AssetMetaPayload:NativeAddress":
				assetMetaPayload.NativeAddress = string(item.Value)
			}
		}
		deets.AssetMetaPayload = assetMetaPayload
	}
	if _, ok := row[nftPayloadFam]; ok {
		nftTransferPayload := &NFTTransferPayload{}
		for _, item := range row[nftPayloadFam] {
			switch item.Column {
			case "NFTTransferPayload:OriginAddress":
				nftTransferPayload.OriginAddress = string(item.Value)
			case "NFTTransferPayload:OriginChain":
				nftTransferPayload.OriginChain = string(item.Value)
			case "NFTTransferPayload:Symbol":
				nftTransferPayload.Symbol = string(item.Value)
			case "NFTTransferPayload:Name":
				nftTransferPayload.Name = string(item.Value)
			case "NFTTransferPayload:TokenId":
				nftTransferPayload.TokenId = string(item.Value)
			case "NFTTransferPayload:URI":
				nftTransferPayload.URI = string(TrimUnicodeFromByteArray(item.Value))
			case "NFTTransferPayload:TargetAddress":
				nftTransferPayload.TargetAddress = string(item.Value)
			case "NFTTransferPayload:TargetChain":
				nftTransferPayload.TargetChain = string(item.Value)
			}
		}
		deets.NFTTransferPayload = nftTransferPayload
	}
	if _, ok := row[chainDetailsFam]; ok {
		chainDetails := &ChainDetails{}
		for _, item := range row[chainDetailsFam] {
			switch item.Column {
			// TEMP - until we have this backfilled/populating for new messages
			// case "ChainDetails:SenderAddress":
			// 	chainDetails.SenderAddress = string(item.Value)
			case "ChainDetails:ReceiverAddress":
				chainDetails.ReceiverAddress = string(item.Value)
			}
		}
		deets.ChainDetails = chainDetails
	}
	return deets
}

func roundToTwoDecimalPlaces(num float64) float64 {
	return math.Round(num*100) / 100
}
func createCachePrefix(prefix string) string {
	cachePrefix := prefix
	if prefix == "" {
		cachePrefix = "*"
	}
	return cachePrefix
}

// useCache allows overriding the cache for a given day.
// This is useful for debugging, to generate fresh data
func useCache(date string) bool {
	skipDates := map[string]bool{
		// for example, add to skip:
		// "2022-02-01": true,
	}
	if _, ok := skipDates[date]; ok {
		return false
	}
	return true
}

// tokens allowed in TVL calculation
var tokenAllowlist = map[string]map[string]bool{}

func isTokenAllowed(chainId string, tokenAddress string) bool {
	if tokenAddresses, ok := tokenAllowlist[chainId]; ok {
		if _, ok := tokenAddresses[tokenAddress]; ok {
			return true
		}
	}
	return false
}

// tokens with no trading activity recorded by exchanges integrated on CoinGecko since the specified date
var inactiveTokens = map[string]map[string]string{
	chainIDToNumberString(vaa.ChainIDEthereum): {
		"0x707f9118e33a9b8998bea41dd0d46f38bb963fc8": "2022-06-15", // Anchor bETH token
	},
}

func isTokenActive(chainId string, tokenAddress string, date string) bool {
	if deactivatedDates, ok := inactiveTokens[chainId]; ok {
		if deactivatedDate, ok := deactivatedDates[tokenAddress]; ok {
			return date < deactivatedDate
		}
	}
	return true
}
