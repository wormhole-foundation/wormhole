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

// token addresses blacklisted from TVL calculation
var tokensToSkip = map[string]bool{
	"0x04132bf45511d03a58afd4f1d36a29d229ccc574":   true,
	"0xa79bd679ce21a2418be9e6f88b2186c9986bbe7d":   true,
	"0x931c3987040c90b6db09981c7c91ba155d3fa31f":   true,
	"0x8fb1a59ca2d57b51e5971a85277efe72c4492983":   true,
	"0xd52d9ba6fcbadb1fe1e3aca52cbb72c4d9bbb4ec":   true,
	"0x1353c55fd2beebd976d7acc4a7083b0618d94689":   true,
	"0xf0fbdb8a402ec0fc626db974b8d019c902deb486":   true,
	"0x1fd4a95f4335cf36cac85730289579c104544328":   true,
	"0x358aa13c52544eccef6b0add0f801012adad5ee3":   true,
	"0xbe32b7acd03bcc62f25ebabd169a35e69ef17601":   true,
	"0x7ffb3d637014488b63fb9858e279385685afc1e2":   true,
	"0x337dc89ebcc33a337307d58a51888af92cfdc81b":   true,
	"0x5Cb89Ac06F34f73B1A6b8000CEb0AfBc97d58B6b":   true,
	"0xd9F0446AedadCf16A12692E02FA26C617FA4D217":   true,
	"0xD7b41531456b636641F7e867eC77120441D1E1E8":   true,
	"0x9f607027b69f6e123bc3bd56a686b735fa75f30a":   true,
	"0x2a35965bbad6fd3964ef815d011c51ab1c546e67":   true,
	"0x053c070f0923a5b770cc59d7bf74ecff991cd0b8":   true,
	"0xA18036c8ecb3235087d990c886c242546D1E560f":   true,
	"0x6B3105826942071E7B6346cbE9867d37Ed7f98Eb":   true,
	"0x0749902ae8ed9c6a508271bad18f185dba7185d4":   true, // fake WETH on poly
	"0x4411146b7714f5dc7aa4445fcb44e3ca120c8a1e":   true, // testWETH on poly
	"0xE389Ac691BD2b0228DAFFfF548fbcE38470373E8":   true, // fake WMATIC on poly
	"0x7e347498dfef39a88099e3e343140ae17cde260e":   true, // fake wAVAX on bsc
	"0x685629e5e99e3959254c4d23cd9097fbaef01fb2":   true, // amWeth
	"terra1vpehfldr2u2m2gw38zaryp4tfw7fe2kw2lryjf": true, //fake btc on terra
	"0xe9986beb0bcfff418dc4a252904cec370dfb14b8":   true, // fake Dai Stablecoin on bsc
}

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

	// ensure blacklisted tokens are lowercase
	for k := range tokensToSkip {
		tokensToSkip[strings.ToLower(k)] = true
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
	case "10001":
		return vaa.ChainIDEthereumRopsten
	}
	return vaa.ChainIDUnset
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
