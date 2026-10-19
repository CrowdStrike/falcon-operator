package tls

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net/url"
	"time"
)

// FCGCertInfo holds parameters for signing a ClusterGuard leaf certificate.
type FCGCertInfo struct {
	CommonName  string
	DNSNames    []string
	URIs        []*url.URL
	ExtKeyUsage []x509.ExtKeyUsage
}

// FCGGenerateCA creates a self-signed CA certificate and returns the PEM-encoded
// certificate and private key.
func FCGGenerateCA(commonName string, days int) (caPEM, caKeyPEM []byte, err error) {
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	ca := &x509.Certificate{
		SerialNumber: new(big.Int).Lsh(big.NewInt(1), 128),
		Subject: pkix.Name{
			CommonName: commonName,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, days),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	caBuf := new(bytes.Buffer)
	if err = pem.Encode(caBuf, &pem.Block{Type: "CERTIFICATE", Bytes: caBytes}); err != nil {
		return nil, nil, err
	}

	caKeyBuf := new(bytes.Buffer)
	if err = pem.Encode(caKeyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey)}); err != nil {
		return nil, nil, err
	}

	return caBuf.Bytes(), caKeyBuf.Bytes(), nil
}

// FCGSignCert signs a leaf certificate using the provided CA PEM data.
// The caller controls DNSNames, URIs, and ExtKeyUsage via FCGCertInfo so that
// server certs can use only ExtKeyUsageServerAuth and client certs can use only
// ExtKeyUsageClientAuth.
func FCGSignCert(caPEM, caKeyPEM []byte, days int, info FCGCertInfo) (certPEM, keyPEM []byte, err error) {
	caBlock, _ := pem.Decode(caPEM)
	if caBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA certificate PEM")
	}
	ca, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	caKeyBlock, _ := pem.Decode(caKeyPEM)
	if caKeyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode CA private key PEM")
	}
	caPrivKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return nil, nil, err
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	cert := &x509.Certificate{
		SerialNumber: new(big.Int).Lsh(big.NewInt(1), 128),
		Subject: pkix.Name{
			CommonName: info.CommonName,
		},
		NotBefore:   time.Now(),
		NotAfter:    time.Now().AddDate(0, 0, days),
		KeyUsage:    x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage: info.ExtKeyUsage,
		DNSNames:    info.DNSNames,
		URIs:        info.URIs,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return nil, nil, err
	}

	certBuf := new(bytes.Buffer)
	if err = pem.Encode(certBuf, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return nil, nil, err
	}

	keyBuf := new(bytes.Buffer)
	if err = pem.Encode(keyBuf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey)}); err != nil {
		return nil, nil, err
	}

	return certBuf.Bytes(), keyBuf.Bytes(), nil
}
