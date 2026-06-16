package jwt

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// LoadRSAPrivateKeyFromPEMFile 加载 PKCS1/PKCS8 私钥
func LoadRSAPrivateKeyFromPEMFile(path string) (*rsa.PrivateKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("decode private pem: empty")
	}

	// PKCS8
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rk, ok := k.(*rsa.PrivateKey); ok {
			return rk, nil
		}
		return nil, fmt.Errorf("private key is not RSA")
	}
	// PKCS1
	if rk, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return rk, nil
	}

	return nil, fmt.Errorf("parse private key failed")
}

// LoadRSAPublicKeyFromPEMFile 加载公钥
func LoadRSAPublicKeyFromPEMFile(path string) (*rsa.PublicKey, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read public key: %w", err)
	}
	block, _ := pem.Decode(b)
	if block == nil {
		return nil, fmt.Errorf("decode public pem: empty")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		if rk, ok := pub.(*rsa.PublicKey); ok {
			return rk, nil
		}
		return nil, fmt.Errorf("public key is not RSA")
	}

	// 兼容 “BEGIN RSA PUBLIC KEY”
	if rk, err := x509.ParsePKCS1PublicKey(block.Bytes); err == nil {
		return rk, nil
	}

	return nil, fmt.Errorf("parse public key failed")
}
