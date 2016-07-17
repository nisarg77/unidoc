/*
 * This file is subject to the terms and conditions defined in
 * file 'LICENSE.txt', which is part of this source code package.
 */

package license

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	_ "github.com/unidoc/unidoc/common"
)

const (
	licenseKeyHeader = "-----BEGIN UNIDOC LICENSE KEY-----"
	licenseKeyFooter = "-----END UNIDOC LICENSE KEY-----"
)

var pubKey = pubProdKey

// Returns signed content in a base64 format which is in format:
//
// Base64OriginalContent
// +
// Base64Signature
func signContent(privKey string, content []byte) (string, error) {
	privBlock, _ := pem.Decode([]byte(privKey))
	if privBlock == nil {
		return "", fmt.Errorf("PrivKey failed")
	}

	priv, err := x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	if err != nil {
		return "", err
	}

	hash := sha512.New()
	hash.Write(content)
	digest := hash.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA512, digest)
	if err != nil {
		return "", err
	}

	ret := base64.StdEncoding.EncodeToString(content)
	ret += "\n+\n"
	ret += base64.StdEncoding.EncodeToString(signature)

	return ret, nil
}

// Verifies and reconstructs the original content
func verifyContent(pubKey string, content string) ([]byte, error) {
	// Empty + line is the delimiter between content and signature.
	// We need to cope with both unix and windows newline, default to unix
	// one and try Windows one as fallback.
	separator := "\n+\n"
	separatorFallback := "\r\n+\r\n"

	sepIdx := strings.Index(content, separator)
	if sepIdx == -1 {
		sepIdx = strings.Index(content, separatorFallback)
		if sepIdx == -1 {
			return nil, fmt.Errorf("Invalid input, signature separator")
		}
	}

	// Original is from start until the separator - 1
	original := content[:sepIdx]

	// Signature is from after the separator until the end of file.
	signatureStarts := sepIdx + len(separator)
	signature := content[signatureStarts:]

	if original == "" || signature == "" {
		return nil, fmt.Errorf("Invalid input, missing original or signature")
	}

	originalBytes, err := base64.StdEncoding.DecodeString(original)
	if err != nil {
		return nil, fmt.Errorf("Invalid input original")
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return nil, fmt.Errorf("Invalid input signature")
	}

	pubBlock, _ := pem.Decode([]byte(pubKey))
	if pubBlock == nil {
		return nil, fmt.Errorf("PubKey failed")
	}

	tempPub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return nil, err
	}

	pub := tempPub.(*rsa.PublicKey)
	if pub == nil {
		return nil, fmt.Errorf("PubKey conversion failed")
	}

	hash := sha512.New()
	hash.Write(originalBytes)
	digest := hash.Sum(nil)

	err = rsa.VerifyPKCS1v15(pub, crypto.SHA512, digest, signatureBytes)
	if err != nil {
		return nil, err
	}

	return originalBytes, nil
}

// Returns the content wrap around the headers
func getWrappedContent(header string, footer string, content string) (string, error) {
	// Find all content between header and footer.
	headerIdx := strings.Index(content, header)
	if headerIdx == -1 {
		return "", fmt.Errorf("Header not found")
	}

	footerIdx := strings.Index(content, footer)
	if footerIdx == -1 {
		return "", fmt.Errorf("Footer not found")
	}

	start := headerIdx + len(header) + 1
	return content[start : footerIdx-1], nil
}

func UseDevKeys() {
	pubKey = pubDevKey
}

func licenseKeyDecode(content string) (LicenseKey, error) {
	var ret LicenseKey

	data, err := getWrappedContent(licenseKeyHeader, licenseKeyFooter, content)
	if err != nil {
		return ret, err
	}

	verifiedRet, err := verifyContent(pubKey, data)
	if err != nil {
		return ret, err
	}

	err = json.Unmarshal(verifiedRet, &ret)
	if err != nil {
		return ret, err
	}

	ret.CreatedAt = time.Unix(ret.CreatedAtInt, 0)
	ret.ExpiresAt = time.Unix(ret.ExpiresAtInt, 0)

	return ret, nil
}
