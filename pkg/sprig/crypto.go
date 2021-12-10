package sprigcrypto

/*
https://github.com/Masterminds/sprig/blob/master/crypto.go


Copyright (C) 2013-2020 Masterminds

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

import (
	"bytes"
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"

	"strings"
)

func GeneratePrivateKey(typ string) string {
	var priv interface{}
	var err error
	switch typ {
	case "", "rsa":
		// good enough for government work
		priv, err = rsa.GenerateKey(rand.Reader, 4096)
	case "ecdsa":
		// again, good enough for government work
		priv, err = ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	case "ed25519":
		_, priv, err = ed25519.GenerateKey(rand.Reader)
	default:
		return "Unknown type " + typ
	}
	if err != nil {
		return fmt.Sprintf("failed to generate private key: %s", err)
	}

	return string(pem.EncodeToMemory(pemBlockForKey(priv)))
}

// DSAKeyFormat stores the format for DSA keys.
// Used by pemBlockForKey
type DSAKeyFormat struct {
	Version       int
	P, Q, G, Y, X *big.Int
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, _ := x509.MarshalECPrivateKey(k)
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		// attempt PKCS#8 format for all other keys
		b, err := x509.MarshalPKCS8PrivateKey(k)
		if err != nil {
			return nil
		}
		return &pem.Block{Type: "PRIVATE KEY", Bytes: b}
	}
}

func parsePrivateKeyPEM(pemBlock string) (crypto.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemBlock))
	if block == nil {
		return nil, errors.New("no PEM data in input")
	}

	if block.Type == "PRIVATE KEY" {
		priv, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("decoding PEM as PKCS#8: %s", err)
		}
		return priv, nil
	} else if !strings.HasSuffix(block.Type, " PRIVATE KEY") {
		return nil, fmt.Errorf("no private key data in PEM block of type %s", block.Type)
	}

	switch block.Type[:len(block.Type)-12] { // strip " PRIVATE KEY"
	case "RSA":
		priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing RSA private key from PEM: %s", err)
		}
		return priv, nil
	case "EC":
		priv, err := x509.ParseECPrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parsing EC private key from PEM: %s", err)
		}
		return priv, nil
	default:
		return nil, fmt.Errorf("invalid private key type %s", block.Type)
	}
}

func getPublicKey(priv crypto.PrivateKey) (crypto.PublicKey, error) {
	switch k := priv.(type) {
	case interface{ Public() crypto.PublicKey }:
		return k.Public(), nil
	default:
		return nil, fmt.Errorf("unable to get public key for type %T", priv)
	}
}

type Certificate struct {
	Cert string
	Key  string
}

func GenerateCertificateAuthority(
	cn string,
	daysValid int,
) (Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return Certificate{}, fmt.Errorf("error generating rsa key: %s", err)
	}

	return generateCertificateAuthorityWithKeyInternal(cn, daysValid, priv)
}

func GenerateCertificateAuthorityWithPEMKey(
	cn string,
	daysValid int,
	privPEM string,
) (Certificate, error) {
	priv, err := parsePrivateKeyPEM(privPEM)
	if err != nil {
		return Certificate{}, fmt.Errorf("parsing private key: %s", err)
	}
	return generateCertificateAuthorityWithKeyInternal(cn, daysValid, priv)
}

func generateCertificateAuthorityWithKeyInternal(
	cn string,
	daysValid int,
	priv crypto.PrivateKey,
) (Certificate, error) {
	ca := Certificate{}

	template, err := getBaseCertTemplate(cn, nil, nil, daysValid)
	if err != nil {
		return ca, err
	}
	// Override KeyUsage and IsCA
	template.KeyUsage = x509.KeyUsageKeyEncipherment |
		x509.KeyUsageDigitalSignature |
		x509.KeyUsageCertSign
	template.IsCA = true

	ca.Cert, ca.Key, err = getCertAndKey(template, priv, template, priv)

	return ca, err
}

func GenerateSelfSignedCertificate(
	cn string,
	ips []interface{},
	alternateDNS []interface{},
	daysValid int,
) (Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return Certificate{}, fmt.Errorf("error generating rsa key: %s", err)
	}
	return generateSelfSignedCertificateWithKeyInternal(cn, ips, alternateDNS, daysValid, priv)
}

func generateSelfSignedCertificateWithKeyInternal(
	cn string,
	ips []interface{},
	alternateDNS []interface{},
	daysValid int,
	priv crypto.PrivateKey,
) (Certificate, error) {
	cert := Certificate{}

	template, err := getBaseCertTemplate(cn, ips, alternateDNS, daysValid)
	if err != nil {
		return cert, err
	}

	cert.Cert, cert.Key, err = getCertAndKey(template, priv, template, priv)

	return cert, err
}

func GenerateSignedCertificate(
	cn string,
	ips []interface{},
	alternateDNS []interface{},
	daysValid int,
	ca Certificate,
) (Certificate, error) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return Certificate{}, fmt.Errorf("error generating rsa key: %s", err)
	}
	return generateSignedCertificateWithKeyInternal(cn, ips, alternateDNS, daysValid, ca, priv)
}

func GenerateSignedCertificateWithPEMKey(
	cn string,
	ips []interface{},
	alternateDNS []interface{},
	daysValid int,
	ca Certificate,
	privPEM string,
) (Certificate, error) {
	priv, err := parsePrivateKeyPEM(privPEM)
	if err != nil {
		return Certificate{}, fmt.Errorf("parsing private key: %s", err)
	}
	return generateSignedCertificateWithKeyInternal(cn, ips, alternateDNS, daysValid, ca, priv)
}

func generateSignedCertificateWithKeyInternal(
	cn string,
	ips []interface{},
	alternateDNS []interface{},
	daysValid int,
	ca Certificate,
	priv crypto.PrivateKey,
) (Certificate, error) {
	cert := Certificate{}

	decodedSignerCert, _ := pem.Decode([]byte(ca.Cert))
	if decodedSignerCert == nil {
		return cert, errors.New("unable to decode Certificate")
	}
	signerCert, err := x509.ParseCertificate(decodedSignerCert.Bytes)
	if err != nil {
		return cert, fmt.Errorf(
			"error parsing Certificate: decodedSignerCert.Bytes: %s",
			err,
		)
	}
	signerKey, err := parsePrivateKeyPEM(ca.Key)
	if err != nil {
		return cert, fmt.Errorf(
			"error parsing private key: %s",
			err,
		)
	}

	template, err := getBaseCertTemplate(cn, ips, alternateDNS, daysValid)
	if err != nil {
		return cert, err
	}

	cert.Cert, cert.Key, err = getCertAndKey(
		template,
		priv,
		signerCert,
		signerKey,
	)

	return cert, err
}

func getCertAndKey(
	template *x509.Certificate,
	signeeKey crypto.PrivateKey,
	parent *x509.Certificate,
	signingKey crypto.PrivateKey,
) (string, string, error) {
	signeePubKey, err := getPublicKey(signeeKey)
	if err != nil {
		return "", "", fmt.Errorf("error retrieving public key from signee key: %s", err)
	}
	derBytes, err := x509.CreateCertificate(
		rand.Reader,
		template,
		parent,
		signeePubKey,
		signingKey,
	)
	if err != nil {
		return "", "", fmt.Errorf("error creating Certificate: %s", err)
	}

	certBuffer := bytes.Buffer{}
	if err := pem.Encode(
		&certBuffer,
		&pem.Block{Type: "Certificate", Bytes: derBytes},
	); err != nil {
		return "", "", fmt.Errorf("error pem-encoding Certificate: %s", err)
	}

	keyBuffer := bytes.Buffer{}
	if err := pem.Encode(
		&keyBuffer,
		pemBlockForKey(signeeKey),
	); err != nil {
		return "", "", fmt.Errorf("error pem-encoding key: %s", err)
	}

	return certBuffer.String(), keyBuffer.String(), nil
}

func getBaseCertTemplate(
	cn string,
	ips []interface{},
	alternateDNS []interface{},
	daysValid int,
) (*x509.Certificate, error) {
	ipAddresses, err := getNetIPs(ips)
	if err != nil {
		return nil, err
	}
	dnsNames, err := getAlternateDNSStrs(alternateDNS)
	if err != nil {
		return nil, err
	}
	serialNumberUpperBound := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, err := rand.Int(rand.Reader, serialNumberUpperBound)
	if err != nil {
		return nil, err
	}
	return &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName: cn,
		},
		IPAddresses: ipAddresses,
		DNSNames:    dnsNames,
		NotBefore:   time.Now(),
		NotAfter:    time.Now().Add(time.Hour * 24 * time.Duration(daysValid)),
		KeyUsage:    x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{
			x509.ExtKeyUsageServerAuth,
			x509.ExtKeyUsageClientAuth,
		},
		BasicConstraintsValid: true,
	}, nil
}

func getNetIPs(ips []interface{}) ([]net.IP, error) {
	if ips == nil {
		return []net.IP{}, nil
	}
	var ipStr string
	var ok bool
	var netIP net.IP
	netIPs := make([]net.IP, len(ips))
	for i, ip := range ips {
		ipStr, ok = ip.(string)
		if !ok {
			return nil, fmt.Errorf("error parsing ip: %v is not a string", ip)
		}
		netIP = net.ParseIP(ipStr)
		if netIP == nil {
			return nil, fmt.Errorf("error parsing ip: %s", ipStr)
		}
		netIPs[i] = netIP
	}
	return netIPs, nil
}

func getAlternateDNSStrs(alternateDNS []interface{}) ([]string, error) {
	if alternateDNS == nil {
		return []string{}, nil
	}
	var dnsStr string
	var ok bool
	alternateDNSStrs := make([]string, len(alternateDNS))
	for i, dns := range alternateDNS {
		dnsStr, ok = dns.(string)
		if !ok {
			return nil, fmt.Errorf(
				"error processing alternate dns name: %v is not a string",
				dns,
			)
		}
		alternateDNSStrs[i] = dnsStr
	}
	return alternateDNSStrs, nil
}
