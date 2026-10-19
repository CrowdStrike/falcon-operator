package tls

import (
	"crypto/x509"
	"encoding/pem"
	"net/url"
	"testing"
	"time"
)

func parseCertPEM(t *testing.T, certPEM []byte) *x509.Certificate {
	t.Helper()
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("failed to decode PEM block")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("failed to parse certificate: %v", err)
	}
	return cert
}

func TestFCGGenerateCA(t *testing.T) {
	caPEM, caKeyPEM, err := FCGGenerateCA("test-ca", 30)
	if err != nil {
		t.Fatalf("FCGGenerateCA returned error: %v", err)
	}
	if len(caPEM) == 0 {
		t.Fatal("caPEM is empty")
	}
	if len(caKeyPEM) == 0 {
		t.Fatal("caKeyPEM is empty")
	}

	cert := parseCertPEM(t, caPEM)

	if !cert.IsCA {
		t.Error("expected IsCA=true")
	}
	if !cert.BasicConstraintsValid {
		t.Error("expected BasicConstraintsValid=true")
	}
	if cert.Subject.CommonName != "test-ca" {
		t.Errorf("CommonName = %q, want %q", cert.Subject.CommonName, "test-ca")
	}

	wantKeyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign
	if cert.KeyUsage != wantKeyUsage {
		t.Errorf("KeyUsage = %v, want %v", cert.KeyUsage, wantKeyUsage)
	}
	if len(cert.ExtKeyUsage) != 0 {
		t.Errorf("CA should have no ExtKeyUsage, got %v", cert.ExtKeyUsage)
	}

	// Validity window: NotBefore should be at or before now, NotAfter ~30 days out.
	now := time.Now()
	if cert.NotBefore.After(now) {
		t.Errorf("NotBefore %v is after now %v", cert.NotBefore, now)
	}
	expectedExpiry := now.AddDate(0, 0, 30)
	if cert.NotAfter.Before(now) || cert.NotAfter.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("NotAfter %v outside expected range", cert.NotAfter)
	}

	// CA should verify itself.
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
		t.Errorf("CA cert does not self-verify: %v", err)
	}

	// Key PEM should be parseable.
	keyBlock, _ := pem.Decode(caKeyPEM)
	if keyBlock == nil || keyBlock.Type != "RSA PRIVATE KEY" {
		t.Fatal("caKeyPEM is not a valid RSA PRIVATE KEY block")
	}
	if _, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes); err != nil {
		t.Fatalf("failed to parse CA private key: %v", err)
	}
}

func TestFCGGenerateCA_DifferentCommonNames(t *testing.T) {
	for _, cn := range []string{"my-ca", "falcon-sensor-ca-default", ""} {
		caPEM, _, err := FCGGenerateCA(cn, 1)
		if err != nil {
			t.Fatalf("FCGGenerateCA(%q) error: %v", cn, err)
		}
		cert := parseCertPEM(t, caPEM)
		if cert.Subject.CommonName != cn {
			t.Errorf("CommonName = %q, want %q", cert.Subject.CommonName, cn)
		}
	}
}

func mustGenerateCA(t *testing.T) (caPEM, caKeyPEM []byte) {
	t.Helper()
	caPEM, caKeyPEM, err := FCGGenerateCA("test-ca", 730)
	if err != nil {
		t.Fatalf("FCGGenerateCA: %v", err)
	}
	return caPEM, caKeyPEM
}

func TestFCGSignCert_ServerCert(t *testing.T) {
	caPEM, caKeyPEM := mustGenerateCA(t)

	certPEM, keyPEM, err := FCGSignCert(caPEM, caKeyPEM, 365, FCGCertInfo{
		CommonName:  "falcon-api.default.svc",
		DNSNames:    []string{"falcon-api.default.svc", "falcon-api.default.svc.cluster.local"},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("FCGSignCert returned error: %v", err)
	}

	cert := parseCertPEM(t, certPEM)

	if cert.IsCA {
		t.Error("leaf cert should not be a CA")
	}
	if cert.Subject.CommonName != "falcon-api.default.svc" {
		t.Errorf("CommonName = %q", cert.Subject.CommonName)
	}
	if len(cert.DNSNames) != 2 || cert.DNSNames[0] != "falcon-api.default.svc" {
		t.Errorf("DNSNames = %v", cert.DNSNames)
	}
	if len(cert.ExtKeyUsage) != 1 || cert.ExtKeyUsage[0] != x509.ExtKeyUsageServerAuth {
		t.Errorf("ExtKeyUsage = %v, want [ServerAuth]", cert.ExtKeyUsage)
	}
	if len(cert.URIs) != 0 {
		t.Errorf("expected no URIs, got %v", cert.URIs)
	}

	// Cert must be signed by the CA.
	caCert := parseCertPEM(t, caPEM)
	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
		t.Errorf("server cert does not verify against CA: %v", err)
	}

	// Key block must be valid.
	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil || keyBlock.Type != "RSA PRIVATE KEY" {
		t.Fatal("keyPEM is not a valid RSA PRIVATE KEY block")
	}
}

func TestFCGSignCert_ClientCertWithSPIFFE(t *testing.T) {
	caPEM, caKeyPEM := mustGenerateCA(t)

	spiffeURI, _ := url.Parse("spiffe://cluster.local/ns/default/sa/falcon-sensor")
	certPEM, _, err := FCGSignCert(caPEM, caKeyPEM, 365, FCGCertInfo{
		CommonName:  "falcon-sensor-client",
		URIs:        []*url.URL{spiffeURI},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		t.Fatalf("FCGSignCert returned error: %v", err)
	}

	cert := parseCertPEM(t, certPEM)

	if cert.IsCA {
		t.Error("leaf cert should not be a CA")
	}
	if cert.Subject.CommonName != "falcon-sensor-client" {
		t.Errorf("CommonName = %q", cert.Subject.CommonName)
	}
	if len(cert.ExtKeyUsage) != 1 || cert.ExtKeyUsage[0] != x509.ExtKeyUsageClientAuth {
		t.Errorf("ExtKeyUsage = %v, want [ClientAuth]", cert.ExtKeyUsage)
	}
	if len(cert.DNSNames) != 0 {
		t.Errorf("expected no DNSNames, got %v", cert.DNSNames)
	}
	if len(cert.URIs) != 1 || cert.URIs[0].String() != spiffeURI.String() {
		t.Errorf("URIs = %v, want [%s]", cert.URIs, spiffeURI)
	}

	// Cert must be signed by the CA. Use KeyUsages to match clientAuth.
	caCert := parseCertPEM(t, caPEM)
	pool := x509.NewCertPool()
	pool.AddCert(caCert)
	if _, err := cert.Verify(x509.VerifyOptions{
		Roots:     pool,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}); err != nil {
		t.Errorf("client cert does not verify against CA: %v", err)
	}
}

func TestFCGSignCert_KeyUsage(t *testing.T) {
	caPEM, caKeyPEM := mustGenerateCA(t)

	certPEM, _, err := FCGSignCert(caPEM, caKeyPEM, 1, FCGCertInfo{
		CommonName:  "test",
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("FCGSignCert: %v", err)
	}

	cert := parseCertPEM(t, certPEM)
	wantKeyUsage := x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment
	if cert.KeyUsage != wantKeyUsage {
		t.Errorf("KeyUsage = %v, want %v", cert.KeyUsage, wantKeyUsage)
	}
}

func TestFCGSignCert_ValidityPeriod(t *testing.T) {
	caPEM, caKeyPEM := mustGenerateCA(t)

	certPEM, _, err := FCGSignCert(caPEM, caKeyPEM, 90, FCGCertInfo{
		CommonName:  "test",
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("FCGSignCert: %v", err)
	}

	cert := parseCertPEM(t, certPEM)
	now := time.Now()
	if cert.NotBefore.After(now) {
		t.Errorf("NotBefore %v is after now", cert.NotBefore)
	}
	expectedExpiry := now.AddDate(0, 0, 90)
	if cert.NotAfter.Before(now) || cert.NotAfter.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("NotAfter %v outside expected range for 90-day cert", cert.NotAfter)
	}
}

func TestFCGSignCert_InvalidCAPEM(t *testing.T) {
	_, caKeyPEM := mustGenerateCA(t)

	_, _, err := FCGSignCert([]byte("not-a-pem"), caKeyPEM, 1, FCGCertInfo{CommonName: "test"})
	if err == nil {
		t.Error("expected error for invalid CA PEM, got nil")
	}
}

func TestFCGSignCert_InvalidCAKeyPEM(t *testing.T) {
	caPEM, _ := mustGenerateCA(t)

	_, _, err := FCGSignCert(caPEM, []byte("not-a-pem"), 1, FCGCertInfo{CommonName: "test"})
	if err == nil {
		t.Error("expected error for invalid CA key PEM, got nil")
	}
}

func TestFCGSignCert_LeafNotTrustedByDifferentCA(t *testing.T) {
	caPEM1, caKeyPEM1 := mustGenerateCA(t)
	caPEM2, _, _ := FCGGenerateCA("other-ca", 730)

	certPEM, _, err := FCGSignCert(caPEM1, caKeyPEM1, 1, FCGCertInfo{
		CommonName:  "test",
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		t.Fatalf("FCGSignCert: %v", err)
	}

	cert := parseCertPEM(t, certPEM)
	otherCA := parseCertPEM(t, caPEM2)
	pool := x509.NewCertPool()
	pool.AddCert(otherCA)
	if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err == nil {
		t.Error("cert should not verify against a different CA")
	}
}
