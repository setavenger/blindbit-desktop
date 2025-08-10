package wallet

import (
	"bytes"
	"fmt"

	"github.com/setavenger/blindbit-lib/wallet"
)

func (s *Scanner) GetAllOwnedUTXOs() []*wallet.OwnedUTXO {
	return s.allOwnedUTXOs
}

// ClearUTXOs clears all found UTXOs (useful for new wallets or manual reset)
func (s *Scanner) ClearUTXOs() {
	s.allOwnedUTXOs = nil
	s.logger.Info().Msg("cleared all UTXOs")
}

// LoadExistingUTXOs loads existing UTXOs into the scanner
func (s *Scanner) LoadExistingUTXOs(existingUTXOs []*wallet.OwnedUTXO) {
	s.allOwnedUTXOs = existingUTXOs
	s.logger.Info().
		Int("utxos_loaded", len(existingUTXOs)).
		Msg("loaded existing UTXOs into scanner")
}

// GetUTXOStats returns statistics about the found UTXOs
func (s *Scanner) GetUTXOStats() (total int, unspent int, spent int) {
	total = len(s.allOwnedUTXOs)
	for _, utxo := range s.allOwnedUTXOs {
		switch utxo.State {
		case wallet.StateUnspent:
			unspent++
		case wallet.StateSpent:
			spent++
		}
	}
	return total, unspent, spent
}

// GetUTXOCount returns the number of found UTXOs
func (s *Scanner) GetUTXOCount() int {
	return len(s.allOwnedUTXOs)
}

// isDuplicateUTXO checks if a UTXO already exists in the scanner's UTXO list
func (s *Scanner) isDuplicateUTXO(utxo *wallet.OwnedUTXO) bool {
	for _, existing := range s.allOwnedUTXOs {
		if bytes.Equal(existing.Txid[:], utxo.Txid[:]) && existing.Vout == utxo.Vout {
			return true
		}
	}
	return false
}

// addUTXOsSafely adds UTXOs to the scanner's list, preventing duplicates
func (s *Scanner) addUTXOsSafely(newUTXOs []*wallet.OwnedUTXO) int {
	added := 0
	for _, utxo := range newUTXOs {
		if !s.isDuplicateUTXO(utxo) {
			s.allOwnedUTXOs = append(s.allOwnedUTXOs, utxo)
			added++
		} else {
			s.logger.Debug().
				Str("txid", fmt.Sprintf("%x", utxo.Txid[:])).
				Uint32("vout", utxo.Vout).
				Msg("skipping duplicate UTXO")
		}
	}
	return added
}
