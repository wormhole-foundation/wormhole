/*
To create a new set of positive-case test data:
	1) Find a Wormhole transactions and note down the transaction hash and the block hash: https://explorer.near.org/accounts/contract.wormhole_crypto.near
	2) Set that block ID as `BLOCK_ID_START`
	3) Update the transaction hash in the test files
*/

package mockserver

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

type (
	ForwardingCachingServer struct {
		upstreamHost      string
		cacheDir          string
		finalCounter      int
		latestFinalBlocks []string
		logger            *zap.Logger
	}
)

func NewForwardingCachingServer(logger *zap.Logger, upstreamHost string, cacheDir string, latestFinalBlocks []string) *ForwardingCachingServer {
	return &ForwardingCachingServer{
		upstreamHost:      upstreamHost,
		cacheDir:          cacheDir,
		finalCounter:      0,
		latestFinalBlocks: latestFinalBlocks,
		logger:            logger,
	}
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func serveCache(w http.ResponseWriter, req *http.Request, cacheDir string) (string, error) {
	reqBody, err := io.ReadAll(req.Body)
	if err != nil {
		return "", errors.New("error reading request")
	}
	req.Body = io.NopCloser(bytes.NewReader(reqBody))

	hashBytes := sha256.Sum256(reqBody)
	hashHex := hex.EncodeToString(hashBytes[:])
	fileName := filepath.Join(cacheDir, hashHex+".json")

	_, err = os.Stat(fileName)
	if err == nil {
		returnFile(w, fileName)
	}
	return fileName, err
}

func returnFile(w http.ResponseWriter, fileName string) {
	dat, err := os.ReadFile(fileName)
	check(err)
	_, err = w.Write(dat)
	check(err)
}

func (s *ForwardingCachingServer) ProxyReq(logger *zap.Logger, req *http.Request) *http.Request {
	reqBody, err := io.ReadAll(req.Body)
	check(err)
	req.Body = io.NopCloser(bytes.NewReader(reqBody))

	url := fmt.Sprintf("%s%s", s.upstreamHost, req.RequestURI)
	proxyReq, _ := http.NewRequestWithContext(req.Context(), req.Method, url, bytes.NewReader(reqBody))

	s.logger.Debug("proxy_req",
		zap.String("url", url),
		zap.String("reqBody", string(reqBody)),
	)

	// TODO: Maybe not forward all headers?
	proxyReq.Header = make(http.Header)
	for h, val := range req.Header {
		if h != "Content-Type" {
			continue
		}
		proxyReq.Header[h] = val
		s.logger.Debug("proxy_req",
			zap.String("header_key", h),
			zap.String("value", val[0]),
		)
	}
	return proxyReq
}

func (s *ForwardingCachingServer) RewriteReq(reqBody []byte) []byte {
	// if the query is for "finality": "final", then we need to rewrite it.
	if bytes.Contains(reqBody, []byte("\"finality\": \"final\"")) {
		nextFinalBlockHash := s.latestFinalBlocks[s.finalCounter]
		if s.finalCounter < len(s.latestFinalBlocks)-1 {
			s.finalCounter++
		}
		reqBody = []byte(
			fmt.Sprintf(`{"id": "dontcare", "jsonrpc": "2.0", "method": "block", "params": {"block_id": "%s"}}`, nextFinalBlockHash),
		)
	}
	return reqBody
}

func (s *ForwardingCachingServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	origReqBody, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	reqBody := s.RewriteReq(origReqBody)
	req.Body = io.NopCloser(bytes.NewReader(reqBody))

	cache_status := ""

	filename, err := serveCache(w, req, s.cacheDir)

	if err == nil {
		// cache was hit.
		cache_status = "cache_hit"
	} else if errors.Is(err, os.ErrNotExist) && s.upstreamHost == "" {
		// cached file does not exist and no upstreamHost defined
		w.WriteHeader(http.StatusNotFound)
		_, err = w.Write([]byte("Not Found"))
		check(err)
		return
	} else if errors.Is(err, os.ErrNotExist) {
		// upstream host is defined so we query upstream and save the response
		cache_status = "cache_miss"

		proxyReq := s.ProxyReq(s.logger, req)

		httpClient := http.Client{}
		resp, err := httpClient.Do(proxyReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}

		// cache the result
		err = os.WriteFile(filename, respBody, 0600)
		check(err)

		// return the result
		_, err = w.Write(respBody)
		check(err)
	} else {
		// Schrodinger: file may or may not exist. See err for details.
		check(err)
	}

	s.logger.Debug("request_received",
		zap.String("origReqBody", string(origReqBody)),
		zap.String("rewrittenReqBody", string(reqBody)),
		zap.String("cache", cache_status),
		zap.String("filename", filename),
	)
}
