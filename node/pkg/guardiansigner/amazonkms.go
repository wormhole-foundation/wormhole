package guardiansigner

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kms_types "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go/aws"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

var (
	secp256k1N     = ethcrypto.S256().Params().N
	secp256k1HalfN = new(big.Int).Div(secp256k1N, big.NewInt(2))
)

type asn1EcSig struct {
	R asn1.RawValue
	S asn1.RawValue
}

type asn1EcPublicKey struct {
	EcPublicKeyInfo asn1EcPublicKeyInfo
	PublicKey       asn1.BitString
}

type asn1EcPublicKeyInfo struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.ObjectIdentifier
}

type AmazonKms struct {
	KeyId  string
	Region string
	svc    *kms.Client
}

func NewAmazonKmsSigner(unsafeDevMode bool, keyPath string) (*AmazonKms, error) {
	amazonKmsSigner := AmazonKms{
		KeyId: keyPath,
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("eu-north-1"))
	if err != nil {
		return nil, errors.New("failed to load default config")
	}

	amazonKmsSigner.svc = kms.NewFromConfig(cfg)

	return &amazonKmsSigner, nil
}

func (a *AmazonKms) Sign(hash []byte) (signature []byte, err error) {

	// request signing
	res, err := a.svc.Sign(context.TODO(), &kms.SignInput{
		KeyId:            aws.String(a.KeyId),
		Message:          hash,
		SigningAlgorithm: kms_types.SigningAlgorithmSpecEcdsaSha256,
		MessageType:      kms_types.MessageTypeDigest,
	})

	if err != nil {
		return nil, fmt.Errorf("Signing failed: %w", err)
	}

	// decode r and s values
	r, s := derSignatureToRS(res.Signature)

	// if s is greater than secp256k1HalfN, we need to substract secp256k1N from it
	sBigInt := new(big.Int).SetBytes(s)
	if sBigInt.Cmp(secp256k1HalfN) > 0 {
		s = new(big.Int).Sub(secp256k1N, sBigInt).Bytes()
	}

	// r and s need to be 32 bytes in size
	r = adjustBufferSize(r)
	s = adjustBufferSize(s)

	// AWS KMS does not provide the recovery id. But that doesn't matter too much, since we can
	// attempt recovery id's 0 and 1, and in the process ensure that the signature is valid.
	expectedPublicKey := a.PublicKey()
	signature = append(r, s...)

	// try recovery id 0
	ecSigWithRecid := append(signature, []byte{0}...)
	pubkey, err := ethcrypto.SigToPub(hash[:], ecSigWithRecid)

	if bytes.Equal(ethcrypto.CompressPubkey(pubkey), ethcrypto.CompressPubkey(&expectedPublicKey)) {
		return ecSigWithRecid, nil
	}

	ecSigWithRecid = append(signature, []byte{1}...)
	pubkey, err = ethcrypto.SigToPub(hash[:], ecSigWithRecid)

	// try recovery id 1
	if bytes.Equal(ethcrypto.CompressPubkey(pubkey), ethcrypto.CompressPubkey(&expectedPublicKey)) {
		return ecSigWithRecid, nil
	}

	return nil, fmt.Errorf("Failed to generate signature")
}

func (a *AmazonKms) PublicKey() ecdsa.PublicKey {
	pubKeyOutput, _ := a.svc.GetPublicKey(context.TODO(), &kms.GetPublicKeyInput{
		KeyId: aws.String(a.KeyId),
	})

	var asn1Pubkey asn1EcPublicKey
	_, _ = asn1.Unmarshal(pubKeyOutput.PublicKey, &asn1Pubkey)

	ecdsaPubkey := ecdsa.PublicKey{
		X: new(big.Int).SetBytes(asn1Pubkey.PublicKey.Bytes[1 : 1+32]),
		Y: new(big.Int).SetBytes(asn1Pubkey.PublicKey.Bytes[1+32:]),
	}

	return ecdsaPubkey
}

func (a *AmazonKms) Verify(sig []byte, hash []byte) (bool, error) {
	return true, nil
}

// https://bitcoin.stackexchange.com/questions/92680/what-are-the-der-signature-and-sec-format
//  1. 0x30 byte: header byte to indicate compound structure
//  2. one byte to encode the length of the following data
//  3. 0x02: header byte indicating an integer
//  4. one byte to encode the length of the following r value
//  5. the r value as a big-endian integer
//  6. 0x02: header byte indicating an integer
//  7. one byte to encode the length of the following s value
//  8. the s value as a big-endian integer
func derSignatureToRS(signature []byte) (rBytes []byte, sBytes []byte) {
	var sigAsn1 asn1EcSig
	_, err := asn1.Unmarshal(signature, &sigAsn1)

	if err != nil {
		panic(err)
	}

	return sigAsn1.R.Bytes, sigAsn1.S.Bytes
	// return rBytes, sBytes
}

func adjustBufferSize(b []byte) []byte {
	length := len(b)

	if length == 32 {
		return b
	}

	if length > 32 {
		return b[length-32:]
	}

	tmp := make([]byte, 32)
	copy(tmp[32-length:], b)

	return tmp
}
