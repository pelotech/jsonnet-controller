/*
Copyright 2021 Pelotech.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package gencert provides a quick and easy utility function for generating
// a self-signed server certificate and writing it to a temp directory on
// disk.
package gencert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// GenerateCert will generate a self-signed certificate and return the path
// to where the files can be loaded. If no error is returned a tls.crt and
// tls.key will be present in the path.
func GenerateCert() (path string, err error) {
	path, err = ioutil.TempDir("", "")
	if err != nil {
		return
	}

	// generate a key
	privKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return
	}

	// create a certificate template
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName:   "jsonnet-controller",
			Organization: []string{"pelotech"},
		},
		DNSNames: []string{
			"jsonnet-controller",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour * 24 * 365),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// self-sign the certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &privKey.PublicKey, privKey)
	if err != nil {
		return
	}

	// Write files to disk
	certPath := filepath.Join(path, "tls.crt")
	keyPath := filepath.Join(path, "tls.key")

	cf, err := os.Create(certPath)
	if err != nil {
		return
	}
	defer cf.Close()
	if err = pem.Encode(cf, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return
	}

	kf, err := os.Create(keyPath)
	if err != nil {
		return
	}
	defer kf.Close()
	if err = pem.Encode(kf, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(privKey)}); err != nil {
		return
	}

	return
}
