package tests

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/setavenger/blindbit-desktop/internal/scanner"
	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/types"
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/setavenger/blindbit-lib/wallet"
)

// var SecretKeyScan types.SecretKey
// var SecretKeySpend types.SecretKey

var (
	secretKeyScan  [32]byte
	secretKeySpend [32]byte
	publicKeyScan  [33]byte
	publicKeySpend [33]byte
)

func LoadKeysFromEnv(t *testing.T) {
	var secretKeyScanHex string = "1f7ffd723ac46bd47e31d312a4ce8f12ea6b4c77f71da8ac2ce32df545c381ee"
	var secretKeySpendHex string = "1f0d5b4d17e5cde8552aef0f13b2a1623bc244047b990ec9c7ed8fd38fc2cc9e"
	var publicKeyScanHex string = "039105fb6bbc3f5d5ed8cdb595c2871a7931d41d30717e65107977ae122bdd4c48"
	var publicKeySpendHex string = "036d4e8c328efa3c16215e1792f42f28e75e3d902e1aa7e158214d46bb937cd42d"

	secretKeyScanBytes, err := hex.DecodeString(secretKeyScanHex)
	if err != nil {
		t.Fatal(err)
	}
	secretKeySpendBytes, err := hex.DecodeString(secretKeySpendHex)
	if err != nil {
		t.Fatal(err)
	}
	publicKeyScanBytes, err := hex.DecodeString(publicKeyScanHex)
	if err != nil {
		t.Fatal(err)
	}
	publicKeySpendBytes, err := hex.DecodeString(publicKeySpendHex)
	if err != nil {
		t.Fatal(err)
	}

	secretKeyScan = utils.ConvertToFixedLength32(secretKeyScanBytes)
	secretKeySpend = utils.ConvertToFixedLength32(secretKeySpendBytes)

	publicKeyScan = utils.ConvertToFixedLength33(publicKeyScanBytes)
	publicKeySpend = utils.ConvertToFixedLength33(publicKeySpendBytes)
}

func TestScanRoutine(t *testing.T) {
	var err error
	LoadKeysFromEnv(t)
	err = LoadMockData()
	if err != nil {
		t.Fatal(err)
	}

	mockWallet := wallet.Wallet{
		Mnemonic:       "",
		Network:        types.NetworkMainnet,
		SecretKeyScan:  secretKeyScan,
		PubKeyScan:     publicKeyScan,
		SecretKeySpend: secretKeySpend,
		PubKeySpend:    publicKeySpend,
		BirthHeight:    0,
		LastScanHeight: 0,
	}
	err = mockWallet.ComputeLabelForM(0)
	if err != nil {
		t.Fatal(err)
		return
	}
	scanner, err := scanner.NewScannerFull(
		&MockBlindBitClient{},
		nil,
		&mockWallet,
		&logging.L,
		mockWallet.LabelSlice(),
		0,
		0,
		nil,
	)

	if err != nil {
		t.Fatal(err)
	}

	// Th way mock is constructed it should always be the same block data that is pulled
	utxos, err := scanner.ScanBlock(0)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("found %d utxos\n", len(utxos))
}
