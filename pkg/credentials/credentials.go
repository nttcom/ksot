/*
 Copyright (c) 2022 NTT Communications Corporation

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

package credentials

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func NewTLSConfig(opts ...TLSConfigOpts) (*tls.Config, error) {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	for _, opt := range opts {
		if err := opt(cfg); err != nil {
			return nil, fmt.Errorf("new tls config: %w", err)
		}
	}
	return cfg, nil
}

type TLSConfigOpts func(c *tls.Config) error

type TLSConfigBase struct {
	// Disable TLS
	NoTLS bool

	// Path to the cert file
	CrtPath string

	// Path to the private key file
	KeyPath string

	// Path to the CA cert file
	CACrtPath string

	// CertData holds PEM-encoded bytes (typically read from a client certificate file).
	// CertData takes precedence over CrtPath
	CrtData []byte

	// KeyData holds PEM-encoded bytes (typically read from a client certificate key file).
	// KeyData takes precedence over KeyPath
	KeyData []byte

	// CACrtData holds PEM-encoded bytes (typically read from a root certificates bundle).
	// CACrtData takes precedence over CACrtPath
	CACrtData []byte
}

// Certificates sets certificate to tls.Config by loading cert key-pairs from files.
func (o *TLSConfigBase) Certificates(required bool) TLSConfigOpts {
	return func(cfg *tls.Config) error {
		if o.NoTLS {
			return nil
		}
		crtBlock, _ := o.loadCrt()
		keyBlock, _ := o.loadKey()
		if crtBlock == nil || keyBlock == nil {
			if required {
				return errors.WithStack(fmt.Errorf("TLS key-pair must be provided to enable TLS"))
			} else {
				return nil
			}
		}
		cert, err := tls.X509KeyPair(crtBlock, keyBlock)
		if err != nil {
			return err
		}
		cfg.Certificates = []tls.Certificate{cert}
		return nil
	}
}

func (o *TLSConfigBase) loadCrt() ([]byte, error) {
	if o.CrtData != nil {
		return o.CrtData, nil
	}
	block, err := os.ReadFile(o.CrtPath)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (o *TLSConfigBase) loadKey() ([]byte, error) {
	if o.KeyData != nil {
		return o.KeyData, nil
	}
	block, err := os.ReadFile(o.KeyPath)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (o *TLSConfigBase) loadCACrt() ([]byte, error) {
	if o.CACrtData != nil {
		return o.CACrtData, nil
	}
	block, err := os.ReadFile(o.CACrtPath)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func (o *TLSConfigBase) caCertPool() (*x509.CertPool, error) {
	if o.CACrtPath == "" && o.CACrtData == nil {
		return nil, nil
	}
	caData, err := o.loadCACrt()
	if caData == nil {
		return nil, errors.WithStack(fmt.Errorf("read CA cert file: %w", err))
	}
	certPool := x509.NewCertPool()
	if ok := certPool.AppendCertsFromPEM(caData); !ok {
		return nil, errors.WithStack(fmt.Errorf("append ca cert from PEM"))
	}
	return certPool, nil
}

type TLSClientConfig struct {
	TLSConfigBase

	// Skip verifying server cert
	SkipVerifyServer bool

	// To verify the server hostname
	ServerName string
}

// VerifyServer applies server verification settings on tls.Config.
func (o *TLSClientConfig) VerifyServer() TLSConfigOpts {
	return func(cfg *tls.Config) error {
		if o.NoTLS {
			return nil
		}
		if o.SkipVerifyServer {
			cfg.InsecureSkipVerify = true
			return nil
		}
		if o.ServerName != "" {
			cfg.ServerName = o.ServerName
		}
		if certPool, err := o.caCertPool(); err != nil {
			return err
		} else if certPool != nil {
			cfg.RootCAs = certPool
		}
		return nil
	}
}

type TLSServerConfig struct {
	TLSConfigBase

	// ClientAuth specifies how to handle client cert
	ClientAuth tls.ClientAuthType
}

// VerifyClient applies client certificate verification settings on tls.Config.
func (o *TLSServerConfig) VerifyClient() TLSConfigOpts {
	return func(cfg *tls.Config) error {
		if o.NoTLS {
			return nil
		}
		cfg.ClientAuth = o.ClientAuth
		if certPool, err := o.caCertPool(); err != nil {
			return err
		} else {
			cfg.ClientCAs = certPool
		}
		return nil
	}
}

// GRPCServerCredentials returns grpc.ServerOption according to the given credential config.
func GRPCServerCredentials(o *TLSServerConfig) ([]grpc.ServerOption, error) {
	if o.NoTLS {
		return []grpc.ServerOption{}, nil
	}

	tlsCfg, err := NewTLSConfig(o.Certificates(true), o.VerifyClient())
	if err != nil {
		return nil, fmt.Errorf("new tls config: %w", err)
	}

	tCred := credentials.NewTLS(tlsCfg)
	return []grpc.ServerOption{grpc.Creds(tCred)}, nil
}
