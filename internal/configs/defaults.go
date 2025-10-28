package configs

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/setavenger/blindbit-lib/logging"
	"github.com/setavenger/blindbit-lib/types"
)

// DefaultDataDir returns the default data dir "~/.blindbit-desktop/"
// if homedir is not found falls back to current directory "."
func DefaultDataDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		logging.L.Err(err).Msg("error getting home directory")
		logging.L.Info().Msg("falling back to current directory")
		homeDir = "."
	}
	dataDir := filepath.Join(homeDir, ".blindbit-desktop")
	logging.L.Trace().Str("data_dir", dataDir).Msg("data directory")
	return dataDir
}

const (
	DefaultOracleAddressMainnet = "oracle.setor.dev"
	DefaultOracleAddressSignet  = "signet.oracle.setor.dev"
	DefaultNetwork              = "signet"
	DefaultMinimumAmount        = 546
	DefaultLabelCount           = 0
)

// DefaultOracleAddressForNetwork returns the default oracle address for a given network.
func DefaultOracleAddressForNetwork(n types.Network) string {
	switch n {
	case types.NetworkMainnet:
		return DefaultOracleAddressMainnet
	case types.NetworkSignet:
		return DefaultOracleAddressSignet
	default:
		return ""
	}
}

// GetCurrentBlockHeight fetches the current block height for a given network from mempool.space.
// Returns 0 if the request fails or the height cannot be parsed.
func GetCurrentBlockHeight(network types.Network) (uint64, error) {
	var url string
	switch network {
	case types.NetworkMainnet:
		url = "https://mempool.space/api/blocks/tip/height"
	case types.NetworkSignet:
		url = "https://mempool.space/signet/api/blocks/tip/height"
	case types.NetworkTestnet:
		url = "https://mempool.space/testnet/api/blocks/tip/height"
	default:
		return 0, fmt.Errorf("unsupported network")
	}

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return 0, fmt.Errorf("failed to fetch block height: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-200 status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response body: %w", err)
	}

	height, err := strconv.ParseUint(string(body), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse block height: %w", err)
	}

	return height, nil
}
