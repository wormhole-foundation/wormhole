package guardiansigner

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/asn1"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kms_types "github.com/aws/aws-sdk-go-v2/service/kms/types"
	"github.com/aws/aws-sdk-go/aws"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

var (
	secp256k1N     = ethcrypto.S256().Params().N
	secp256k1HalfN = new(big.Int).Div(secp256k1N, big.NewInt(2))

	// The timeout for KMS operations. This is necessary to avoid situations where
	// the signing or verification is blocked indefinitely.
	KMS_TIMEOUT               = time.Second * 15
	MINIMUM_KMS_PUBKEY_LENGTH = 65
)

// The ASN.1 structure for an ECDSA signature produced by AWS KMS.
type asn1EcSig struct {
	R asn1.RawValue
	S asn1.RawValue
}

// The ASN.1 structure for an ECDSA public key produced by AWS KMS.
type asn1EcPublicKey struct {
	EcPublicKeyInfo asn1EcPublicKeyInfo
	PublicKey       asn1.BitString
}

// The ASN.1 structure for the public key info in an ECDSA public key produced by AWS KMS.
type asn1EcPublicKeyInfo struct {
	Algorithm  asn1.ObjectIdentifier
	Parameters asn1.ObjectIdentifier
}

// getRegionFromArn extracts the region from an ARN. The region is at index 3 in the ARN.
func getRegionFromArn(arn string) string {
	// Information in ARNs are colon-separated
	arn_parts := strings.Split(arn, ":")

	// https://docs.aws.amazon.com/IAM/latest/UserGuide/reference-arns.html#arns-syntax
	// The format of an ARN is arn:partition:service:region:account-id:resource-info, so
	// the region is at index 3.
	if len(arn) < 4 {
		return ""
	}

	return arn_parts[3]
}

// AmazonKms is a signer that uses AWS KMS to sign messages. The URI is expected to be
// in the format amazonkms://<key-arn>.
type AmazonKms struct {
	keyId     string
	region    string
	publicKey ecdsa.PublicKey
	client    *kms.Client
}

// NewAmazonKmsSigner creates a new AmazonKms signer. The keyPath is expected to be an ARN,
// identifying the key in AWS KMS. The region is extracted from the ARN, and the AWS KMS
// client is created with the region.
// NOTE: The public key is retrieved during signer creation, and stored as a property of the
// signer. This is because the public key is not expected to change during runtime.
func NewAmazonKmsSigner(ctx context.Context, unsafeDevMode bool, keyPath string) (*AmazonKms, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, KMS_TIMEOUT)
	defer cancel()

	// Extract the region from the key path. The region is required to create a new KMS client.
	// If the region is not present in the key path, the ARN is considered invalid.
	region := getRegionFromArn(keyPath)

	if region == "" {
		return nil, errors.New("Invalid KMS ARN")
	}

	amazonKmsSigner := AmazonKms{
		keyId:  keyPath,
		region: region,
	}

	// Create a configuration object to create a new KMS client from. The region passed to
	// `config.WithDefaultRegion()` must match the region in the actual ARN, otherwise the SDK throws
	// an error. This is why the region is first extracted from the keyPath.
	cfg, err := config.LoadDefaultConfig(timeoutCtx, config.WithDefaultRegion(amazonKmsSigner.region))
	if err != nil {
		return nil, errors.New("Failed to load KMS default config")
	}

	amazonKmsSigner.client = kms.NewFromConfig(cfg)

	// Get the public key here, and store it as a property. The public key shouldn't change during
	// runtime, so it's safe to fetch once and store it as a property.
	pubKeyOutput, err := amazonKmsSigner.client.GetPublicKey(timeoutCtx, &kms.GetPublicKeyInput{
		KeyId: aws.String(amazonKmsSigner.keyId),
	})

	if err != nil {
		return nil, fmt.Errorf("KMS signer creation failed: %w", err)
	}

	var asn1Pubkey asn1EcPublicKey
	_, err = asn1.Unmarshal(pubKeyOutput.PublicKey, &asn1Pubkey)

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal KMS public key: %w", err)
	}

	// The public key is expected to be at least `MINIMUM_KMS_PUBKEY_LENGTH` bytes long.
	if len(asn1Pubkey.PublicKey.Bytes) < MINIMUM_KMS_PUBKEY_LENGTH {
		return nil, errors.New("Invalid KMS public key length")
	}

	// It is possible to use `ethcrypto.UnmarshalPubkey(asn1Pubkey.PublicKey.Bytes)`` to get the public key,
	// but `UnmarshalPubkey()` uses elliptic.Unmarshal() internally, which has been marked as deprecated.
	// The following code implements similar logic, with the indexes meaning the following:
	// 0: The first byte is the prefix byte, which is 0x04 for uncompressed keys.
	// 1-32: The next 32 bytes are the X coordinate.
	// 33-64: The next 32 bytes are the Y coordinate.
	ecdsaPubkey := ecdsa.PublicKey{
		X: new(big.Int).SetBytes(asn1Pubkey.PublicKey.Bytes[1 : 1+32]),
		Y: new(big.Int).SetBytes(asn1Pubkey.PublicKey.Bytes[1+32:]),
	}

	amazonKmsSigner.publicKey = ecdsaPubkey

	return &amazonKmsSigner, nil
}

func (a *AmazonKms) Sign(ctx context.Context, hash []byte) (signature []byte, err error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, KMS_TIMEOUT)
	defer cancel()

	// Call the AWS KMS service to sign the input hash.
	res, err := a.client.Sign(timeoutCtx, &kms.SignInput{
		KeyId:            aws.String(a.keyId),
		Message:          hash,
		SigningAlgorithm: kms_types.SigningAlgorithmSpecEcdsaSha256,
		MessageType:      kms_types.MessageTypeDigest,
	})

	if err != nil {
		return nil, fmt.Errorf("KMS Signing failed: %w", err)
	}

	// Decode r and s values
	r, s, err := derSignatureToRS(res.Signature)

	if err != nil {
		return nil, fmt.Errorf("Failed to decode signature: %w", err)
	}

	// if s is greater than secp256k1HalfN, we need to subtract secp256k1N from it
	sBigInt := new(big.Int).SetBytes(s)
	if sBigInt.Cmp(secp256k1HalfN) > 0 {
		s = new(big.Int).Sub(secp256k1N, sBigInt).Bytes()
	}

	// r and s need to be 32 bytes in size
	r = adjustBufferSize(r)
	s = adjustBufferSize(s)

	// AWS KMS does not provide the recovery id. But that doesn't matter too much, since we can
	// attempt recovery id's 0 and 1, and in the process ensure that the signature is valid.
	expectedPublicKey := a.PublicKey(ctx)
	signature = append(r, s...)

	// try recovery id 0
	ecSigWithRecid := append(signature, []byte{0}...)
	pubkey, _ := ethcrypto.SigToPub(hash[:], ecSigWithRecid)

	if bytes.Equal(ethcrypto.CompressPubkey(pubkey), ethcrypto.CompressPubkey(&expectedPublicKey)) {
		return ecSigWithRecid, nil
	}

	ecSigWithRecid = append(signature, []byte{1}...)
	pubkey, _ = ethcrypto.SigToPub(hash[:], ecSigWithRecid)

	// try recovery id 1
	if bytes.Equal(ethcrypto.CompressPubkey(pubkey), ethcrypto.CompressPubkey(&expectedPublicKey)) {
		return ecSigWithRecid, nil
	}

	// Reaching this return implies that it wasn't possible to generate a valid signature. This shouldn't
	// happen, unless there is something seriously wrong with the KMS service.
	return nil, fmt.Errorf("Failed to generate valid signature")
}

func (a *AmazonKms) PublicKey(ctx context.Context) ecdsa.PublicKey {
	return a.publicKey
}

func (a *AmazonKms) Verify(ctx context.Context, sig []byte, hash []byte) (bool, error) {
	timeoutCtx, cancel := context.WithTimeout(ctx, KMS_TIMEOUT)
	defer cancel()

	// Use ethcrypto to recover the public key
	recoveredPubKey, err := ethcrypto.SigToPub(hash, sig)

	if err != nil {
		return false, err
	}

	// Load the KMS signer's public key
	kmsPublicKey := a.PublicKey(timeoutCtx)

	return recoveredPubKey.Equal(kmsPublicKey), nil
}

// Return the signer type as "amazonkms".
func (a *AmazonKms) TypeAsString() string {
	return "amazonkms"
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
func derSignatureToRS(signature []byte) ([]byte, []byte, error) {
	var sigAsn1 asn1EcSig
	_, err := asn1.Unmarshal(signature, &sigAsn1)

	if err != nil {
		return nil, nil, err
	}

	return sigAsn1.R.Bytes, sigAsn1.S.Bytes, nil
}

// adjustBufferSize takes an input buffer and
// a) trims it down to 32 bytes, starting at the most significant byte, if the input length is greater than 32, or
// b) returns the input as-is, if the input length is equal to 32, or
// c) left-pads it to 32 bytes, if the input length is less than 32.
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
