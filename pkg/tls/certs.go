package tls

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"time"
)

type CertInfo struct {
	CommonName string
	DNSNames   []string
}

// CertSetup will generate and return tls certs
func CertSetup(namespace string, days int, certInfo CertInfo) ([]byte, []byte, []byte, error) {
	// set up our CA certificate
	ca := &x509.Certificate{
		SerialNumber: new(big.Int).Lsh(big.NewInt(1), 128),
		Subject: pkix.Name{
			CommonName: namespace + " ca",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(0, 0, days),
		IsCA:                  true,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		BasicConstraintsValid: true,
	}

	// create our private and public key
	caPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	// create the CA
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	// pem encode
	caPEM := new(bytes.Buffer)
	if err = pem.Encode(caPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: caBytes,
	}); err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	caPrivKeyPEM := new(bytes.Buffer)
	if err = pem.Encode(caPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(caPrivKey),
	}); err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	// set up our server certificate
	cert := &x509.Certificate{
		SerialNumber: new(big.Int).Lsh(big.NewInt(1), 128),
		Subject: pkix.Name{
			CommonName: certInfo.CommonName,
		},
		NotBefore:      time.Now(),
		NotAfter:       time.Now().AddDate(0, 0, days),
		AuthorityKeyId: ca.SubjectKeyId,
		ExtKeyUsage:    []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		KeyUsage:       x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		IsCA:           false,
		DNSNames:       certInfo.DNSNames,
	}

	certPrivKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, cert, ca, &certPrivKey.PublicKey, caPrivKey)
	if err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	certPEM := new(bytes.Buffer)
	if err = pem.Encode(certPEM, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certBytes,
	}); err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	certPrivKeyPEM := new(bytes.Buffer)
	if err = pem.Encode(certPrivKeyPEM, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(certPrivKey),
	}); err != nil {
		return []byte{}, []byte{}, []byte{}, err
	}

	certpool := x509.NewCertPool()
	certpool.AppendCertsFromPEM(caPEM.Bytes())

	return certPEM.Bytes(),
		certPrivKeyPEM.Bytes(),
		caPEM.Bytes(),
		nil
}
