package p

import (
	"context"
	"encoding/json"
	"log"
	"math"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/bigtable"
	"cloud.google.com/go/pubsub"
	"github.com/certusone/wormhole/node/pkg/vaa"
)

// shared code for the various functions, primarily response formatting.

// client is a global Bigtable client, to avoid initializing a new client for
// every request.
var client *bigtable.Client
var clientOnce sync.Once
var tbl *bigtable.Table

var pubsubClient *pubsub.Client
var pubSubTokenTransferDetailsTopic *pubsub.Topic

var coinGeckoCoins = map[string][]CoinGeckoCoin{}
var solanaTokens = map[string]SolanaToken{}

var releaseDay = time.Date(2021, 9, 13, 0, 0, 0, 0, time.UTC)
var pwd string

func initCache(waitgroup *sync.WaitGroup, filePath string, mutex *sync.RWMutex, cacheInterface interface{}) {
	defer waitgroup.Done()
	loadJsonToInterface(filePath, mutex, cacheInterface)
}

// init runs during cloud function initialization. So, this will only run during an
// an instance's cold start.
// https://cloud.google.com/functions/docs/bestpractices/networking#accessing_google_apis
func init() {
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

		var pubsubErr error
		pubsubClient, pubsubErr = pubsub.NewClient(context.Background(), project)
		if pubsubErr != nil {
			log.Printf("pubsub.NewClient error: %v", pubsubErr)
			return
		}
	})
	tbl = client.Open("v2Events")

	// create the topic that will be published to after decoding token transfer payloads
	tokenTransferDetailsTopic := os.Getenv("PUBSUB_TOKEN_TRANSFER_DETAILS_TOPIC")
	if tokenTransferDetailsTopic != "" {
		pubSubTokenTransferDetailsTopic = pubsubClient.Topic(tokenTransferDetailsTopic)
		// fetch the token lists once at start up
		coinGeckoCoins = fetchCoinGeckoCoins()
		solanaTokens = fetchSolanaTokenList()
	}

	pwd, _ = os.Getwd()

	// initialize in-memory caches
	var initWG sync.WaitGroup

	initWG.Add(1)
	// populates cache used by amountsTransferredToInInterval
	go initCache(&initWG, warmTransfersToCacheFilePath, &muWarmTransfersToCache, &warmTransfersToCache)

	initWG.Add(1)
	// populates cache used by createTransfersOfInterval
	go initCache(&initWG, warmTransfersCacheFilePath, &muWarmTransfersCache, &warmTransfersCache)

	initWG.Add(1)
	// populates cache used by createAddressesOfInterval
	go initCache(&initWG, warmAddressesCacheFilePath, &muWarmAddressesCache, &warmAddressesCache)

	initWG.Add(1)
	// populates cache used by transferredToSince
	go initCache(&initWG, transferredToUpToYesterdayFilePath, &muTransferredToUpToYesterday, &transferredToUpToYesterday)

	// initWG.Add(1)
	// populates cache used by transferredSince
	// initCache(initWG, transferredUpToYesterdayFilePath, &muTransferredToUpYesterday, &transferredUpToYesterday)

	initWG.Add(1)
	// populates cache used by addressesTransferredToSince
	go initCache(&initWG, addressesToUpToYesterdayFilePath, &muAddressesToUpToYesterday, &addressesToUpToYesterday)

	initWG.Add(1)
	// populates cache used by createCumulativeAmountsOfInterval
	go initCache(&initWG, warmCumulativeCacheFilePath, &muWarmCumulativeCache, &warmCumulativeCache)

	initWG.Add(1)
	// populates cache used by createCumulativeAddressesOfInterval
	go initCache(&initWG, warmCumulativeAddressesCacheFilePath, &muWarmCumulativeAddressesCache, &warmCumulativeAddressesCache)

	initWG.Wait()
	log.Println("done initializing caches, starting.")

}

var gcpCachePath = "/workspace/src/p/cache"

func loadJsonToInterface(filePath string, mutex *sync.RWMutex, cacheMap interface{}) {
	// create path to the static cache dir
	path := gcpCachePath + filePath
	// create path to the "hot" cache dir
	hotPath := "/tmp" + filePath
	if strings.HasSuffix(pwd, "cmd") {
		// alter the path to be correct when running locally, and in Tilt devnet
		path = "../cache" + filePath
		hotPath = ".." + hotPath
	}
	mutex.Lock()
	// first check to see if there is a cache file in the tmp dir of the cloud function.
	// if so, this is a long running instance with a recently generated cache available.
	fileData, readErrTmp := os.ReadFile(hotPath)
	if readErrTmp != nil {
		log.Printf("failed reading from tmp cache %v, err: %v", hotPath, readErrTmp)
		var readErr error
		fileData, readErr = os.ReadFile(path)
		if readErr != nil {
			log.Printf("failed reading %v, err: %v", path, readErr)
		} else {
			log.Printf("successfully read from cache: %v", path)
		}
	} else {
		log.Printf("successfully read from tmp cache: %v", hotPath)
	}
	unmarshalErr := json.Unmarshal(fileData, &cacheMap)
	mutex.Unlock()
	if unmarshalErr != nil {
		log.Printf("failed unmarshaling %v, err: %v", path, unmarshalErr)
	}
}
func persistInterfaceToJson(filePath string, mutex *sync.RWMutex, cacheMap interface{}) {
	path := "/tmp" + filePath
	if strings.HasSuffix(pwd, "cmd") {
		// alter the path to be correct when running locally, and in Tilt devnet
		path = "../cache" + filePath
	}
	mutex.Lock()
	cacheBytes, marshalErr := json.MarshalIndent(cacheMap, "", "  ")
	if marshalErr != nil {
		log.Fatal("failed marshaling cacheMap.", marshalErr)
	}
	writeErr := os.WriteFile(path, cacheBytes, 0666)
	mutex.Unlock()
	if writeErr != nil {
		log.Fatalf("failed writing to file %v, err: %v", path, writeErr)
	}
	log.Printf("successfully wrote cache to file: %v", path)
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
		// fields below exist on the row, but no need to return them currently.
		// NotionalUSD        uint64
		// TokenPriceUSD      uint64
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
	// NotionalUSD and TokenPriceUSD are more percise than the string versions returned,
	// however the precision is not required, so leaving this commented out for now.
	// if _, ok := row[transferDetailsFam]; ok {
	// 	for _, item := range row[transferDetailsFam] {
	// 		switch item.Column {
	// 		case "TokenTransferDetails:NotionalUSD":
	// 			reader := bytes.NewReader(item.Value)
	// 			var notionalUSD uint64
	// 			if err := binary.Read(reader, binary.BigEndian, &notionalUSD); err != nil {
	// 				log.Fatalf("failed to read NotionalUSD of row: %v. err %v ", row.Key(), err)
	// 			}
	// 			deets.TransferDetails.NotionalUSD = notionalUSD

	// 		case "TokenTransferDetails:TokenPriceUSD":
	// 			reader := bytes.NewReader(item.Value)
	// 			var tokenPriceUSD uint64
	// 			if err := binary.Read(reader, binary.BigEndian, &tokenPriceUSD); err != nil {
	// 				log.Fatalf("failed to read TokenPriceUSD of row: %v. err %v", row.Key(), err)
	// 			}
	// 			deets.TransferDetails.TokenPriceUSD = tokenPriceUSD
	// 		}
	// 	}
	// }
	if _, ok := row[chainDetailsFam]; ok {
		chainDetails := &ChainDetails{}
		for _, item := range row[chainDetailsFam] {
			switch item.Column {
			case "ChainDetails:SenderAddress":
				chainDetails.SenderAddress = string(item.Value)
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

var mux = newMux()

// Entry is the cloud function entry point
func Entry(w http.ResponseWriter, r *http.Request) {
	mux.ServeHTTP(w, r)
}

func newMux() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/notionaltransferred", NotionalTransferred)
	mux.HandleFunc("/notionaltransferredto", NotionalTransferredTo)
	mux.HandleFunc("/notionaltransferredtocumulative", NotionalTransferredToCumulative)
	mux.HandleFunc("/addressestransferredto", AddressesTransferredTo)
	mux.HandleFunc("/addressestransferredtocumulative", AddressesTransferredToCumulative)
	mux.HandleFunc("/totals", Totals)
	mux.HandleFunc("/recent", Recent)
	mux.HandleFunc("/transaction", Transaction)
	mux.HandleFunc("/readrow", ReadRow)
	mux.HandleFunc("/findvalues", FindValues)

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	return mux
}
