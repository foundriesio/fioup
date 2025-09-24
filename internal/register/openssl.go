// Copyright (c) Qualcomm Technologies, Inc. and/or its subsidiaries.
// SPDX-License-Identifier: BSD-3-Clause-Clear

package register

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"log/slog"
)

// Create a new EC key and CSR
func openSSLCreateCSR(opt *RegisterOptions) (key string, csr string, err error) {
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", "", err
	}

	subj := pkix.Name{
		CommonName:         opt.UUID,
		OrganizationalUnit: []string{opt.Factory},
	}
	if opt.Production {
		subj.ExtraNames = append(subj.ExtraNames, pkix.AttributeTypeAndValue{
			Type:  []int{2, 5, 4, 15}, // businessCategory OID
			Value: "production",
		})
	}

	template := x509.CertificateRequest{
		Subject:            subj,
		SignatureAlgorithm: x509.ECDSAWithSHA256,
	}

	slog.Debug("Generating CSR", "factory", opt.Factory)
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, priv)
	if err != nil {
		return "", "", err
	}
	csrPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE REQUEST", Bytes: csrDER})

	privBytes, err := x509.MarshalECPrivateKey(priv)
	if err != nil {
		return "", "", err
	}
	privPEMBytes := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: privBytes})

	return string(privPEMBytes), string(csrPEMBytes), nil
}
