package lib

import (
	"bytes"
	"errors"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// GetAddressFromZpub derives a native segwit address (P2WPKH) from a zpub key.
func GetAddressFromZpub(key *hdkeychain.ExtendedKey, external bool, index int) (string, error) {
	changeIndex := uint32(0)
	if !external {
		changeIndex = 1
	}

	changeKey, err := key.Derive(changeIndex)
	if err != nil {
		return "", err
	}

	addressKey, err := changeKey.Derive(uint32(index))
	if err != nil {
		return "", err
	}

	pk, err := addressKey.ECPubKey()
	if err != nil {
		return "", err
	}

	witness := btcutil.Hash160(pk.SerializeCompressed())
	awpkh, err := btcutil.NewAddressWitnessPubKeyHash(witness, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	return awpkh.EncodeAddress(), nil
}

// GetAddressFromXpub derives a legacy P2PKH address from an xpub key.
func GetAddressFromXpub(key *hdkeychain.ExtendedKey, external bool, index int) (string, error) {
	changeIndex := uint32(0)
	if !external {
		changeIndex = 1
	}

	changeKey, err := key.Derive(changeIndex)
	if err != nil {
		return "", err
	}

	addressKey, err := changeKey.Derive(uint32(index))
	if err != nil {
		return "", err
	}

	pk, err := addressKey.ECPubKey()
	if err != nil {
		return "", err
	}

	hash160 := btcutil.Hash160(pk.SerializeCompressed())
	addr, err := btcutil.NewAddressPubKeyHash(hash160, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	return addr.EncodeAddress(), nil
}

// GetAddressFromYpub derives a nested segwit address (P2SH-P2WPKH) from a ypub key.
func GetAddressFromYpub(key *hdkeychain.ExtendedKey, external bool, index int) (string, error) {
	changeIndex := uint32(0)
	if !external {
		changeIndex = 1
	}

	changeKey, err := key.Derive(changeIndex)
	if err != nil {
		return "", err
	}

	addressKey, err := changeKey.Derive(uint32(index))
	if err != nil {
		return "", err
	}

	pk, err := addressKey.ECPubKey()
	if err != nil {
		return "", err
	}

	witness := btcutil.Hash160(pk.SerializeCompressed())
	// Create the redeem script for a P2SH-P2WPKH address:
	// [OP_0, 20-byte-hash]
	redeemScript, err := txscript.NewScriptBuilder().
		AddOp(txscript.OP_0).
		AddData(witness).
		Script()
	if err != nil {
		return "", err
	}

	addr, err := btcutil.NewAddressScriptHash(redeemScript, &chaincfg.MainNetParams)
	if err != nil {
		return "", err
	}

	return addr.EncodeAddress(), nil
}

// GetAddressFromExtPubKey now supports zpub, xpub, and ypub.
func GetAddressFromExtPubKey(extPubKey string, external bool, addressIndex int) (string, error) {
	key, err := hdkeychain.NewKeyFromString(extPubKey)
	if err != nil {
		return "", err
	}

	version := key.Version()
	switch {
	// zpub: native segwit, version bytes 0x04, 0xb2, 0x47, 0x46.
	case bytes.Equal(version, []byte{0x04, 0xb2, 0x47, 0x46}):
		return GetAddressFromZpub(key, external, addressIndex)
	// xpub: legacy, version bytes 0x04, 0x88, 0xb2, 0x1e.
	case bytes.Equal(version, []byte{0x04, 0x88, 0xb2, 0x1e}):
		return GetAddressFromXpub(key, external, addressIndex)
	// ypub: nested segwit, version bytes 0x04, 0x9d, 0x7c, 0xb2.
	case bytes.Equal(version, []byte{0x04, 0x9d, 0x7c, 0xb2}):
		return GetAddressFromYpub(key, external, addressIndex)
	default:
		return "", errors.New("unsupported key version")
	}
}
