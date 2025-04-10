package lib

import (
	"strings"
	"testing"
)

// Test vectors: Replace these with actual BIP-44, BIP-49, and BIP-84 test vectors.
var testVectors = []struct {
	extPubKey      string
	expectedAddr   string
	expectedPrefix string
	desc           string
}{
	// xpub: BIP-44 (Legacy P2PKH, Base58, starts with "1")
	{
		"xpub6CnZRRFQYwSurSaFd18GbnFTibUxYB8zHNQxEx4Bxfmj2D6ag3ccxLEyW6KwrJPZeFFkKUwkPAgd5Q8Z5qFyVJwHQaHSP13N3xSXLpPTZWz",
		"1M1526FgFUzZjvCSzUpG6PDyxePqpWgFgJ", // Example expected address
		"1",
		"Legacy P2PKH (xpub)",
	},

	// ypub: BIP-49 (Nested SegWit P2SH-P2WPKH, Base58, starts with "3")
	{
		"ypub6YHtkqW2UMLWS3D4f5YDwh5jdpH8k6iirTetr8cCxELCkVTNnsDrdWvanMEmxvRNrqtSF1LD8gttdD7vvqnGH8bQTNdDszdaCWzDYhbzD6v",
		"3FhVafrkkzXKJxtFeGkHqDzWKuiPhNBi3D", // Example expected address
		"3",
		"Nested SegWit P2SH-P2WPKH (ypub)",
	},

	// zpub: BIP-84 (Native SegWit P2WPKH, Bech32, starts with "bc1")
	{
		"zpub6qvq3PUnYFhMEcCzWP86Xw5fNfQmUFBkaG91FQmdK7o2EcrEnqkPXY5x3wXQJBiVs9JnvurFJT6vYKQmAaiZxNcbSBreN23Lx2TQZqGJsAh",
		"bc1q56caystznj5r2pptyxx7ajm3axudhgpq6w6srw", // Example expected address
		"bc1",
		"Native SegWit P2WPKH (zpub)",
	},
}

// Invalid extended keys (wrong prefixes, invalid length, corrupted values)
var invalidKeys = []string{
	"tpubD6NzVbkrYhZ4WVW2e7oYtM3mpVKwVJ9PpZ3ajp2XJwFmJRWu7v6T7P1UzxwD3BhwZkHzrKfK8kixHVEF9HfUX9qgYiSfw3oyq3c9WqZL", // Testnet (unsupported)
	"randomstring",
	"zprvAfhtdKG53dks19hGqxdpFtZhwnrV1AHw6wpk1RXvrocdvYLENjxWRq3c221gtz5TLL3j23njxqBhTfNVY6RHJrYwBKG1p6UyonpV1Daqf48", // Private key (should be rejected)
	"ypubINVALIDDATA",
	"zpubINVALIDDATA",
}

// ** Test valid xpub, ypub, and zpub extended keys and compare expected output **
func TestGetAddressFromExtPubKey_Valid(t *testing.T) {
	for _, tc := range testVectors {
		addr, err := GetAddressFromExtPubKey(tc.extPubKey, true, 0)
		if err != nil {
			t.Fatalf("Unexpected error for %s: %v", tc.desc, err)
		}
		if addr != tc.expectedAddr {
			t.Errorf("Mismatch for %s: expected %s, got %s", tc.desc, tc.expectedAddr, addr)
		}
		if !strings.HasPrefix(addr, tc.expectedPrefix) {
			t.Errorf("Expected %s address to start with %q, got: %s", tc.desc, tc.expectedPrefix, addr)
		}
	}
}

// ** Test invalid xpub, ypub, zpub keys to ensure they return an error **
func TestGetAddressFromExtPubKey_Invalid(t *testing.T) {
	for _, key := range invalidKeys {
		_, err := GetAddressFromExtPubKey(key, true, 0)
		if err == nil {
			t.Errorf("Expected error for invalid key %q, but got none", key)
		}
	}
}
