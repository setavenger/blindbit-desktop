package scanner

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/setavenger/blindbit-lib/wallet"
	"github.com/setavenger/go-bip352"
)

// MarkSpentUTXOs marks UTXOs as spent based on the spent outpoints filter
func (s *Scanner) MarkSpentUTXOs(data *BlockData) error {
	markStart := time.Now()

	filterStart := time.Now()
	blockHeight := data.Height
	filter := data.FilterSpent

	// Get the spent outpoints filter
	filterDuration := time.Since(filterStart)

	// Generate local outpoint hashes for our UTXOs
	hashStart := time.Now()
	hashes := s.generateLocalOutpointHashes([32]byte(filter.BlockHash))
	hashDuration := time.Since(hashStart)

	// Convert to byte slice for filter matching
	var hashesForFilter [][]byte
	for hash := range hashes {
		var newHash = make([]byte, 8)
		copy(newHash[:], hash[:])
		hashesForFilter = append(hashesForFilter, newHash[:])
	}

	// Check if any of our UTXOs match the filter
	matchStart := time.Now()
	isMatch, err := matchFilter(filter.Data, filter.BlockHash, hashesForFilter)
	if err != nil {
		s.logger.Err(err).Msg("failed to match filter")
		return err
	}
	matchDuration := time.Since(matchStart)

	if !isMatch {
		// todo: experiment with defer and attaching msg in defer
		totalDuration := time.Since(markStart)
		s.logger.Debug().Uint64("height", blockHeight).
			Dur("total", totalDuration).
			Dur("get_filter", filterDuration).
			Dur("generate_hashes", hashDuration).
			Dur("match_filter", matchDuration).
			Int("utxo_count", len(s.allOwnedUTXOs)).
			Msg("mark spent UTXOs timing (no match)")
		return nil
	}

	// Get the spent outpoints index
	indexStart := time.Now()
	index, err := s.Client.GetSpentOutpointsIndex(blockHeight)
	if err != nil {
		s.logger.Err(err).Msg("failed to get spent outpoints index")
		return err
	}
	indexDuration := time.Since(indexStart)

	// Mark matching UTXOs as spent
	markMatchingStart := time.Now()
	markedCount := 0
	for _, hash := range index.Data {
		if utxoPtr, ok := hashes[hash]; ok {
			utxoPtr.State = wallet.StateSpent
			markedCount++
			s.logger.Debug().
				Str("txid", fmt.Sprintf("%x", utxoPtr.Txid[:])).
				Uint32("vout", utxoPtr.Vout).
				Msg("marked UTXO as spent")
		}
	}
	markMatchingDuration := time.Since(markMatchingStart)

	totalDuration := time.Since(markStart)
	s.logger.Debug().Uint64("height", blockHeight).
		Dur("total", totalDuration).
		Dur("get_filter", filterDuration).
		Dur("generate_hashes", hashDuration).
		Dur("match_filter", matchDuration).
		Dur("get_index", indexDuration).
		Dur("mark_matching", markMatchingDuration).
		Int("all_owned_utxo_count", len(s.allOwnedUTXOs)).
		Int("marked_count", markedCount).
		Msg("mark spent UTXOs timing")

	return nil
}

// generateLocalOutpointHashes generates hashes for local UTXOs
func (s *Scanner) generateLocalOutpointHashes(blockHash [32]byte) map[[8]byte]*wallet.OwnedUTXO {
	outputs := make(map[[8]byte]*wallet.OwnedUTXO, len(s.allOwnedUTXOs))
	blockHashLE := bip352.ReverseBytesCopy(blockHash[:])

	for _, utxo := range s.allOwnedUTXOs {
		if utxo.State == wallet.StateSpent {
			continue
		}

		var buf bytes.Buffer
		// can be optimized with Putting bytes into byte slice/array
		buf.Write(bip352.ReverseBytesCopy(utxo.Txid[:]))
		binary.Write(&buf, binary.LittleEndian, utxo.Vout)

		hashed := sha256.Sum256(append(buf.Bytes(), blockHashLE...))
		var shortHash [8]byte
		copy(shortHash[:], hashed[:])
		outputs[shortHash] = utxo
	}

	return outputs
}
