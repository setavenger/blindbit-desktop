package tests

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/networking"
	"github.com/setavenger/blindbit-lib/utils"
)

type MockBlindBitClient struct{}

var (
	MockFilter networking.Filter
	MockUTXOs  []*networking.UTXOServed
	MockTweaks [][33]byte
)

/*
	GetChainTip() (uint64, error)
	GetFilter(blockHeight uint64, filterType FilterType) (*Filter, error)
	GetSpentOutpointsIndex(blockHeight uint64) (SpentOutpointsIndex, error)
	GetTweaks(blockHeight uint64, dustLimit uint64) ([][33]byte, error)
	GetUTXOs(blockHeight uint64) ([]*UTXOServed, error)

*/

func (c *MockBlindBitClient) GetChainTip() (uint64, error) {
	return 0, nil
}

func (c *MockBlindBitClient) GetFilter(
	blockHeight uint64,
	filterType networking.FilterType,
) (*networking.Filter, error) {
	return &MockFilter, nil
}

func (c *MockBlindBitClient) GetSpentOutpointsIndex(blockHeight uint64) (networking.SpentOutpointsIndex, error) {
	return networking.SpentOutpointsIndex{}, nil
}

func (c *MockBlindBitClient) GetTweaks(blockHeight uint64, dustLimit uint64) ([][33]byte, error) {
	fmt.Println(len(MockTweaks))
	return MockTweaks, nil
}

func (c *MockBlindBitClient) GetUTXOs(blockHeight uint64) ([]*networking.UTXOServed, error) {
	// filter for the one utxo we want to make it easier

	var thoseWeWant []*networking.UTXOServed
	for i := range MockUTXOs {
		v := MockUTXOs[i]
		if hex.EncodeToString(v.Txid[:]) != "668317dd7ed4b0ab0b744e280506a21f1891b93e72d57a409f04279f6c7ca93e" {
			continue
		}
		thoseWeWant = append(thoseWeWant, v)
	}

	return thoseWeWant, nil
	// return MockUTXOs, nil
}

func LoadMockData() error {
	utxosFile, err := os.ReadFile("./mockdata/utxos-892230.json")
	if err != nil {
		return err
	}

	filterFile, err := os.ReadFile("./mockdata/filter-892230.json")
	if err != nil {
		return err
	}

	tweaksFile, err := os.ReadFile("./mockdata/tweaks-892230.json")
	if err != nil {
		return err
	}

	var data struct {
		FilterType  uint8  `json:"filter_type"`
		BlockHeight uint64 `json:"block_height"`
		BlockHash   string `json:"block_hash"`
		Data        string `json:"data"`
	}
	err = json.Unmarshal(filterFile, &data)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	blockHash, err := hex.DecodeString(data.BlockHash)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}
	filterData, err := hex.DecodeString(data.Data)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	MockFilter = networking.Filter{
		FilterType:  data.FilterType,
		BlockHeight: data.BlockHeight,
		BlockHash:   utils.ConvertToFixedLength32(blockHash),
		Data:        filterData,
	}

	var tweakData []string
	err = json.Unmarshal(tweaksFile, &tweakData)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}

	// Convert []string to [][33]byte
	for _, hexStr := range tweakData {
		// Each string should be exactly 66 characters long (33 bytes)
		if len(hexStr) != 66 {
			return fmt.Errorf("invalid hex string length: %d", len(hexStr))
		}
		// Decode hex string to byte slice
		byteSlice, err := hex.DecodeString(hexStr)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
		// Convert byte slice to [33]byte
		var byteArray [33]byte
		copy(byteArray[:], byteSlice[:])
		MockTweaks = append(MockTweaks, byteArray)
	}

	// if err = json.Unmarshal(utxosFile, &MockUTXOs); err != nil {
	// 	return err
	// }
	var dataSlice []struct {
		Txid         string `json:"txid"`
		Vout         uint32 `json:"vout"`
		Amount       uint64 `json:"value"` // todo refactor on backend as well, so json tag matches name
		ScriptPubKey string `json:"scriptpubkey"`
		BlockHeight  uint64 `json:"block_height"`
		BlockHash    string `json:"block_hash"`
		Timestamp    uint64 `json:"timestamp"`
		Spent        bool   `json:"spent"`
	}

	err = json.Unmarshal(utxosFile, &dataSlice)
	if err != nil {
		logging.L.Err(err).Msg("")
		return err
	}
	for _, data := range dataSlice {
		var blockHashBytes []byte
		blockHashBytes, err = hex.DecodeString(data.BlockHash)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
		var scriptPubKeyBytes []byte
		scriptPubKeyBytes, err = hex.DecodeString(data.ScriptPubKey)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}
		var txidBytes []byte
		txidBytes, err = hex.DecodeString(data.Txid)
		if err != nil {
			logging.L.Err(err).Msg("")
			return err
		}

		utxo := &networking.UTXOServed{
			Txid:         utils.ConvertToFixedLength32(txidBytes),
			Vout:         data.Vout,
			Amount:       data.Amount,
			BlockHeight:  data.BlockHeight,
			BlockHash:    utils.ConvertToFixedLength32(blockHashBytes),
			ScriptPubKey: [34]byte(scriptPubKeyBytes),
			Timestamp:    data.Timestamp,
			Spent:        data.Spent,
		}

		MockUTXOs = append(MockUTXOs, utxo)
	}

	return nil
}
