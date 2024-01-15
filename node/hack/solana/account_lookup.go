package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gagliardetto/solana-go"
	lookup "github.com/gagliardetto/solana-go/programs/address-lookup-table"
	"github.com/gagliardetto/solana-go/rpc"
)

const RPC = "https://api.devnet.solana.com"

func populateLookupTableAccounts(ctx context.Context, tx *solana.Transaction, rpcClient *rpc.Client) error {
	if !tx.Message.IsVersioned() {
		return nil
	}

	tblKeys := tx.Message.GetAddressTableLookups().GetTableIDs()
	if len(tblKeys) == 0 {
		return nil
	}

	resolutions := make(map[solana.PublicKey]solana.PublicKeySlice)
	for _, key := range tblKeys {
		fmt.Println(key)
		info, err := rpcClient.GetAccountInfo(ctx, key)
		if err != nil {
			fmt.Println("We errored here!")
			return err
		}

		tableContent, err := lookup.DecodeAddressLookupTableState(info.GetBinary())
		if err != nil {
			return err
		}

		resolutions[key] = tableContent.Addresses
	}

	err := tx.Message.SetAddressTables(resolutions)
	if err != nil {
		return err
	}

	err = tx.Message.ResolveLookups()
	if err != nil {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()
	testTx, err := solana.SignatureFromBase58("2Jr3bAuEKwYBKmaqSmmFQ2R7xxxQpmjY8g3N3gMH49C62kBaweUgc9UCEcFhqcewAVnDLcBGWUSQrKZ7vdxpBbq4")
	if err != nil {
		log.Fatal("SignatureFromBase58 errored", err)
	}
	rpcClient := rpc.New(RPC)
	maxSupportedTransactionVersion := uint64(0)
	tx, err := rpcClient.GetTransaction(ctx, testTx, &rpc.GetTransactionOpts{
		Encoding:                       solana.EncodingBase64,
		MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
	})
	if err != nil {
		log.Fatal("getTransaction errored", err)
	}
	realTx, err := tx.Transaction.GetTransaction()
	if err != nil {
		log.Fatal("GetTransaction errored", err)
	}
	err = populateLookupTableAccounts(ctx, realTx, rpcClient)
	if err != nil {
		log.Fatal("populateLookupTableAccounts errored", err)
	}

}
