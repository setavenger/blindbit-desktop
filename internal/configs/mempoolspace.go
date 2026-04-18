package configs

import "github.com/setavenger/blindbit-lib/types"

func GetMempoolSpaceURL(network types.Network) string {
	switch network {
	case types.NetworkMainnet:
		return "https://mempool.space"
	case types.NetworkTestnet:
		return "https://mempool.space/testnet"
	case types.NetworkSignet:
		return "https://mempool.space/signet"
	}
	return "https://mempool.space"
}
