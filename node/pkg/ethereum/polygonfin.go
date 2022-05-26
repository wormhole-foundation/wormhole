// This implements the finality check for Polygon.
//
// It can take up to 512 blocks for polygon blocks to be finalized. Rather than wait that long, we will query the checkpoint to see if they are finalized sooner.
//
// TestNet query URL: "https://apis.matic.network/api/v1/mumbai/block-included/"
// MainNet query URL: "https://apis.matic.network/api/v1/matic/block-included/block-number/"

package ethereum

import (
	"context"
	"encoding/json"
	common "github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"
	"io/ioutil"
	"math/big"
	"net/http"
)

type PolygonFinalizer struct {
	Url               string
	logger            *zap.Logger
	networkName       string
	highestCheckpoint big.Int
}

func UsePolygonFinalizer(extraParams []string) bool {
	return len(extraParams) != 0 && extraParams[0] != ""
}

func (f *PolygonFinalizer) SetLogger(l *zap.Logger, netName string) {
	f.logger = l
	f.networkName = netName
	f.logger.Info("using Polygon specific finality check", zap.String("eth_network", f.networkName), zap.String("query_url", f.Url))
}

func (f *PolygonFinalizer) DialContext(ctx context.Context, _rawurl string) (err error) {
	return nil
}

func (f *PolygonFinalizer) IsBlockFinalized(ctx context.Context, block *common.NewBlock) (bool, error) {
	if block.Number.Cmp(&f.highestCheckpoint) <= 0 {
		return true, nil
	}

	url := f.Url + block.Number.String()
	response, err := http.Get(url)
	if err != nil {
		return false, err
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return false, err
	}

	var result map[string]string
	json.Unmarshal([]byte(responseData), &result)

	status := result["message"]
	if status == "" || status == "No block found" {
		f.logger.Info("DEBUG: not finalized", zap.String("eth_network", f.networkName), zap.String("requested_block", block.Number.String()))
		return false, nil
	}

	if status != "success" {
		f.logger.Error("unexpected checkpoint status", zap.String("eth_network", f.networkName),
			zap.String("requested_block", block.Number.String()), zap.String("status", status))
		return false, nil
	}

	// If we get this far, we know this block is finalized, so we will return true even in the error cases.

	endStr := result["end"]
	if endStr == "" {
		f.logger.Error("checkpoint reply is missing end", zap.String("eth_network", f.networkName), zap.String("requested_block", block.Number.String()))
		return true, nil
	}

	end, ok := new(big.Int).SetString(endStr, 10)
	if !ok {
		f.logger.Error("checkpoint reply contains unexpected end", zap.String("eth_network", f.networkName),
			zap.String("requested_block", block.Number.String()), zap.String("end_str", endStr))
		return true, nil
	}

	f.highestCheckpoint = *end

	f.logger.Info("checkpoint query returned", zap.String("eth_network", f.networkName),
		zap.String("requested_block", block.Number.String()), zap.String("reply", string(responseData)))

	return (block.Number.Cmp(&f.highestCheckpoint) <= 0), nil
}
