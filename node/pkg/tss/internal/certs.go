package internal

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log"
)

// CreateCert invokes x509.CreateCertificate and returns it in the x509.Certificate format
func CreateCert(template, parent *x509.Certificate, pub *ecdsa.PublicKey, parentPriv *ecdsa.PrivateKey) (
	cert *x509.Certificate, certPEM []byte, err error) {

	certDER, err := x509.CreateCertificate(rand.Reader, template, parent, pub, parentPriv)
	if err != nil {
		return
	}
	// parse the resulting certificate so we can use it again
	cert, err = x509.ParseCertificate(certDER)
	if err != nil {
		return
	}
	// PEM encode the certificate (this is a standard TLS encoding)
	b := pem.Block{Type: "CERTIFICATE", Bytes: certDER}
	certPEM = pem.EncodeToMemory(&b)
	return
}

// NewTLSCredentials creates a self signed certificate
func NewTLSCredentials(secretKey *ecdsa.PrivateKey, template *x509.Certificate) *x509.Certificate {
	// this cert will be the CA that we will use to sign the server cert
	template.IsCA = true
	// describe what the certificate will be used for
	template.KeyUsage = x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature
	template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth}

	rootCert, _, err := CreateCert(template, template, &secretKey.PublicKey, secretKey)
	if err != nil {
		log.Fatalf("error creating cert: %v", err)
	}

	return rootCert
}

func PrivateKeyToPem(pkey *ecdsa.PrivateKey) []byte {
	keyBytes, err := x509.MarshalPKCS8PrivateKey(pkey)
	if err != nil {
		log.Fatal("shouldn't happen")
	}

	return pem.EncodeToMemory(&pem.Block{
		Type: "PRIVATE KEY", Bytes: keyBytes,
	})
}

func PublicKeyToPem(pkey *ecdsa.PublicKey) ([]byte, error) {
	keyBytes, err := x509.MarshalPKIXPublicKey(pkey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal public key: %w", err)
	}

	return pem.EncodeToMemory(&pem.Block{
		Type: "PUBLIC KEY", Bytes: keyBytes,
	}), nil
}

func PemToPublicKey(pemBytes []byte) (*ecdsa.PublicKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing the public key")
	}
	if block.Type != "PUBLIC KEY" {
		return nil, errors.New("PEM block is not a public key")
	}

	k, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	publicKey, ok := k.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("parsed public key is not an ECDSA key")
	}

	return publicKey, nil
}
func PemToPrivateKey(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing the private key")
	}
	if block.Type != "PRIVATE KEY" {
		return nil, errors.New("PEM block is not a private key")
	}

	k, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	privateKey, ok := k.(*ecdsa.PrivateKey)
	if !ok {
		return nil, errors.New("parsed private key is not an ECDSA key")
	}

	return privateKey, nil
}

func CertToPem(cert *x509.Certificate) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: cert.Raw})
}

func PemToCert(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode PEM block containing the certificate")
	}
	if block.Type != "CERTIFICATE" {
		return nil, errors.New("PEM block is not a certificate")
	}

	return x509.ParseCertificate(block.Bytes)
}
