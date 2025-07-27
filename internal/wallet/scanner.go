package wallet

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil/gcs"
	"github.com/btcsuite/btcd/btcutil/gcs/builder"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/rs/zerolog"
	"github.com/setavenger/blindbit-scan/pkg/networking"
	"github.com/setavenger/blindbit-scan/pkg/scan"
	scanwallet "github.com/setavenger/blindbit-scan/pkg/wallet"
	"github.com/setavenger/go-bip352"
	"github.com/setavenger/go-electrum/electrum"
)

// UTXOState represents the state of a UTXO
type UTXOState int8

const (
	StateUnconfirmed UTXOState = iota + 1
	StateUnspent
	StateUnconfirmedSpent
	StateSpent
)

func (u UTXOState) String() string {
	return [...]string{"unconfirmed", "unspent", "unconfirmed_spent", "spent"}[u-1]
}

// OwnedUTXO represents a UTXO owned by the wallet
type OwnedUTXO struct {
	Txid         [32]byte      `json:"txid"`
	Vout         uint32        `json:"vout"`
	Amount       uint64        `json:"amount"`
	PrivKeyTweak [32]byte      `json:"priv_key_tweak"`
	PubKey       [32]byte      `json:"pub_key"`
	Timestamp    uint64        `json:"timestamp"`
	State        UTXOState     `json:"utxo_state"`
	Label        *bip352.Label `json:"label"`
}

// Scanner handles the scanning process
type Scanner struct {
	client         *networking.ClientBlindBit
	electrum       *electrum.Client
	logger         *zerolog.Logger
	wallet         *Wallet
	scanSecret     [32]byte
	spendSecret    [32]byte
	labels         []*bip352.Label
	birthHeight    uint64
	lastScanHeight uint64

	allOwnedUTXOs    []*OwnedUTXO
	progressCallback func(uint64)

	// Scanning control
	scanning bool
	stopChan chan struct{}
	scanMu   sync.Mutex
	doneChan chan struct{} // Added for StopSync
}

// Implement the scan.Scanner interface
func (s *Scanner) SpendPubKey() [33]byte {
	return s.pubKeySpend33()
}

func (s *Scanner) ScanSecretKey() [32]byte {
	return s.scanSecret
}

func (s *Scanner) Labels() []*bip352.Label {
	return s.labels
}

// NewScanner creates a new scanner
func NewScanner(
	oracleURL string,
	electrumURL string,
	wallet *Wallet,
	logger *zerolog.Logger,
	labelCount int, // Add label count parameter
) (*Scanner, error) {
	// Create BlindBit client
	blindBitClient := &networking.ClientBlindBit{BaseUrl: oracleURL}

	// Create Electrum client - disabled for now
	var electrumClient *electrum.Client = nil

	// Convert secrets to fixed length arrays
	var scanSecret [32]byte
	var spendSecret [32]byte
	copy(scanSecret[:], wallet.ScanSecret)
	copy(spendSecret[:], wallet.SpendSecret)

	// Generate labels properly with the specified count
	labels, err := generateLabels(wallet, labelCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate labels: %w", err)
	}

	// Use the logger directly (it already has caller information)
	scanner := &Scanner{
		client:         blindBitClient,
		electrum:       electrumClient,
		logger:         logger,
		wallet:         wallet,
		scanSecret:     scanSecret,
		spendSecret:    spendSecret,
		labels:         labels,
		birthHeight:    840000, // Default birth height
		lastScanHeight: 0,
		stopChan:       make(chan struct{}),
	}

	return scanner, nil
}

// generateLabels creates labels for the wallet
func generateLabels(wallet *Wallet, labelCount int) ([]*bip352.Label, error) {

	// we always need the change label m=0
	// if 21 labels are requested we count 21 desired labels additionally to the change label
	// change label is not intended to be "usable" for the user
	labelCount++

	// Generate labels based on the specified count
	labels := make([]*bip352.Label, 0, labelCount)

	// Convert spend secret to fixed length array
	var scanSecret [32]byte
	copy(scanSecret[:], wallet.ScanSecret)

	// Generate the specified number of labels
	for i := range labelCount {
		label, err := bip352.CreateLabel(scanSecret, uint32(i))
		if err != nil {
			return nil, err
		}
		labels = append(labels, &label)
	}

	return labels, nil
}

// SetProgressCallback sets a callback function for real-time progress updates
func (s *Scanner) SetProgressCallback(callback func(uint64)) {
	s.progressCallback = callback
}

// SetBirthHeight sets the birth height for scanning
func (s *Scanner) SetBirthHeight(height uint64) {
	s.birthHeight = height
}

// SetLastScanHeight sets the last scan height
func (s *Scanner) SetLastScanHeight(height uint64) {
	s.lastScanHeight = height
}

// GetLastScanHeight returns the last scan height
func (s *Scanner) GetLastScanHeight() uint64 {
	return s.lastScanHeight
}

// pubKeySpend33 derives the public key from SpendSecret
func (s *Scanner) pubKeySpend33() [33]byte {
	priv, pub := btcec.PrivKeyFromBytes(s.wallet.SpendSecret)
	_ = priv // not used
	return bip352.ConvertToFixedLength33(pub.SerializeCompressed())
}

// ScanBlock scans a single block for UTXOs using the scan package
func (s *Scanner) ScanBlock(blockHeight uint64) ([]*OwnedUTXO, error) {
	s.logger.Info().Uint64("height", blockHeight).Msg("scanning block")

	// Get tweaks for this block
	tweaks, err := s.client.GetTweaks(blockHeight, 1000) // Default dust limit
	if err != nil {
		return nil, fmt.Errorf("failed to get tweaks: %w", err)
	}

	if len(tweaks) == 0 {
		return nil, nil
	}

	// Get UTXOs for this block
	utxos, err := s.client.GetUTXOs(blockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get UTXOs: %w", err)
	}

	// Use the scan package to identify owned UTXOs
	ownedUTXOs, err := scan.ScanDataOptimized(s, utxos, tweaks)
	if err != nil {
		return nil, fmt.Errorf("failed to scan data: %w", err)
	}
	if len(ownedUTXOs) == 0 {
		// early function exit
		return nil, nil
	}

	// Convert from scan package format to our format
	var result []*OwnedUTXO
	for _, utxo := range ownedUTXOs {
		state := StateUnspent
		if utxo.State == scanwallet.StateSpent {
			state = StateSpent
		} else if utxo.State == scanwallet.StateUnconfirmedSpent {
			state = StateUnconfirmedSpent
		} else if utxo.State == scanwallet.StateUnconfirmed {
			state = StateUnconfirmed
		}

		result = append(result, &OwnedUTXO{
			Txid:         utxo.Txid,
			Vout:         utxo.Vout,
			Amount:       utxo.Amount,
			PrivKeyTweak: utxo.PrivKeyTweak,
			PubKey:       utxo.PubKey,
			Timestamp:    utxo.Timestamp,
			State:        state,
			Label:        utxo.Label,
		})
	}

	return result, nil
}

// SyncToTipWithProgress syncs from the last scan height to the current chain tip with progress callback
func (s *Scanner) SyncToTipWithProgress(progressCallback func(uint64)) error {
	chainTip, err := s.client.GetChainTip()
	if err != nil {
		return fmt.Errorf("failed to get chain tip: %w", err)
	}

	s.logger.Info().Uint64("chain_tip", chainTip).Msg("syncing to tip")

	startHeight := s.birthHeight
	if s.lastScanHeight >= startHeight {
		startHeight = s.lastScanHeight + 1
	}

	if startHeight > chainTip {
		return nil
	}

	if startHeight == 0 {
		startHeight = 1
	}

	// Don't reset allOwnedUTXOs - accumulate UTXOs across scan cycles
	for i := startHeight; i <= chainTip; i++ {
		// Check for stop signal before scanning each block
		select {
		case <-s.stopChan:
			s.logger.Info().Msg("scanning stopped during block scan")
			return nil
		default:
		}

		s.logger.Debug().Uint64("height", i).Msg("scanning block")

		// Mark spent UTXOs first
		err = s.MarkSpentUTXOs(i)
		if err != nil {
			s.logger.Err(err).Uint64("height", i).Msg("error marking spent UTXOs")
			// Continue scanning even if marking spent UTXOs fails
		}

		ownedUTXOs, err := s.ScanBlock(i)
		if err != nil {
			return fmt.Errorf("failed to scan block %d: %w", i, err)
		}

		if len(ownedUTXOs) > 0 {
			s.logger.Info().Uint64("height", i).Int("utxos_found", len(ownedUTXOs)).Msg("found UTXOs")
			added := s.addUTXOsSafely(ownedUTXOs)
			if added > 0 {
				s.logger.Info().Uint64("height", i).Int("utxos_added", added).Int("utxos_skipped", len(ownedUTXOs)-added).Msg("processed UTXOs")
			}
		}

		s.lastScanHeight = i

		// Report progress via callback
		progressCallback(i)

		// Save progress every 100 blocks
		if i%100 == 0 {
			s.logger.Debug().Uint64("height", i).Msg("saving progress")
		}
	}

	return nil
}

// matchFilter checks if any values match the GCS filter
func matchFilter(nBytes []byte, blockHash [32]byte, values [][]byte) (bool, error) {
	c := chainhash.Hash{}
	err := c.SetBytes(bip352.ReverseBytesCopy(blockHash[:]))
	if err != nil {
		return false, fmt.Errorf("failed to set hash bytes: %w", err)
	}

	filter, err := gcs.FromNBytes(builder.DefaultP, builder.DefaultM, nBytes)
	if err != nil {
		return false, fmt.Errorf("failed to create filter: %w", err)
	}

	key := builder.DeriveKey(&c)
	isMatch, err := filter.HashMatchAny(key, values)
	if err != nil {
		return false, fmt.Errorf("failed to match filter: %w", err)
	}

	return isMatch, nil
}

func (s *Scanner) GetAllOwnedUTXOs() []*OwnedUTXO {
	return s.allOwnedUTXOs
}

// ClearUTXOs clears all found UTXOs (useful for new wallets or manual reset)
func (s *Scanner) ClearUTXOs() {
	s.allOwnedUTXOs = nil
	s.logger.Info().Msg("cleared all UTXOs")
}

// LoadExistingUTXOs loads existing UTXOs into the scanner
func (s *Scanner) LoadExistingUTXOs(existingUTXOs []*OwnedUTXO) {
	s.allOwnedUTXOs = existingUTXOs
	s.logger.Info().Int("utxos_loaded", len(existingUTXOs)).Msg("loaded existing UTXOs into scanner")
}

// GetUTXOStats returns statistics about the found UTXOs
func (s *Scanner) GetUTXOStats() (total int, unspent int, spent int) {
	total = len(s.allOwnedUTXOs)
	for _, utxo := range s.allOwnedUTXOs {
		switch utxo.State {
		case StateUnspent:
			unspent++
		case StateSpent:
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
func (s *Scanner) isDuplicateUTXO(utxo *OwnedUTXO) bool {
	for _, existing := range s.allOwnedUTXOs {
		if bytes.Equal(existing.Txid[:], utxo.Txid[:]) && existing.Vout == utxo.Vout {
			return true
		}
	}
	return false
}

// addUTXOsSafely adds UTXOs to the scanner's list, preventing duplicates
func (s *Scanner) addUTXOsSafely(newUTXOs []*OwnedUTXO) int {
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

// MarkSpentUTXOs marks UTXOs as spent based on the spent outpoints filter
func (s *Scanner) MarkSpentUTXOs(blockHeight uint64) error {
	// Get the spent outpoints filter
	filter, err := s.client.GetFilter(blockHeight, networking.SpentOutpointsFilterType)
	if err != nil {
		s.logger.Err(err).Msg("failed to get spent outpoints filter")
		return err
	}

	// Generate local outpoint hashes for our UTXOs
	hashes := s.generateLocalOutpointHashes([32]byte(filter.BlockHash))

	// Convert to byte slice for filter matching
	var hashesForFilter [][]byte
	for hash := range hashes {
		var newHash = make([]byte, 8)
		copy(newHash[:], hash[:])
		hashesForFilter = append(hashesForFilter, newHash[:])
	}

	// Check if any of our UTXOs match the filter
	isMatch, err := matchFilter(filter.Data, filter.BlockHash, hashesForFilter)
	if err != nil {
		s.logger.Err(err).Msg("failed to match filter")
		return err
	}

	if !isMatch {
		return nil
	}

	// Get the spent outpoints index
	index, err := s.client.GetSpentOutpointsIndex(blockHeight)
	if err != nil {
		s.logger.Err(err).Msg("failed to get spent outpoints index")
		return err
	}

	// Mark matching UTXOs as spent
	for _, hash := range index.Data {
		if utxoPtr, ok := hashes[hash]; ok {
			utxoPtr.State = StateSpent
			s.logger.Debug().
				Str("txid", fmt.Sprintf("%x", utxoPtr.Txid[:])).
				Uint32("vout", utxoPtr.Vout).
				Msg("marked UTXO as spent")
		}
	}

	return nil
}

// generateLocalOutpointHashes generates hashes for local UTXOs
func (s *Scanner) generateLocalOutpointHashes(blockHash [32]byte) map[[8]byte]*OwnedUTXO {
	outputs := make(map[[8]byte]*OwnedUTXO, len(s.allOwnedUTXOs))
	blockHashLE := bip352.ReverseBytesCopy(blockHash[:])

	for _, utxo := range s.allOwnedUTXOs {
		if utxo.State == StateSpent {
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

// Start begins the scanning process in a goroutine
func (s *Scanner) Start() error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if s.scanning {
		s.logger.Warn().Msg("scanning already in progress, ignoring start request")
		return fmt.Errorf("scanning already in progress")
	}

	s.scanning = true
	s.stopChan = make(chan struct{}) // Reset stop channel

	s.logger.Info().Msg("starting scanner")

	go func() {
		defer func() {
			s.scanMu.Lock()
			s.scanning = false
			s.scanMu.Unlock()
			s.logger.Info().Msg("scanner stopped")

			// Signal that we're done if there's a done channel
			if s.doneChan != nil {
				close(s.doneChan)
				s.doneChan = nil
			}
		}()

		for {
			// Check if we should stop before starting a new scan cycle
			select {
			case <-s.stopChan:
				s.logger.Info().Msg("received stop signal before scan cycle")
				return
			default:
			}

			err := s.SyncToTipWithProgress(func(height uint64) {
				// Check for stop signal during progress updates
				select {
				case <-s.stopChan:
					return
				default:
				}

				// Call the original progress callback if set
				if s.progressCallback != nil {
					s.progressCallback(height)
				}
			})

			if err != nil {
				s.logger.Error().Err(err).Msg("error during scanning")
			} else {
				// Final UTXO update when scanning completes successfully
				if s.progressCallback != nil {
					s.progressCallback(s.lastScanHeight)
				}
			}

			// Check if we should stop after scan cycle
			select {
			case <-s.stopChan:
				s.logger.Info().Msg("received stop signal after scan cycle")
				return
			default:
				// Wait before next scan cycle
				s.logger.Debug().Msg("waiting before next scan cycle")
				time.Sleep(30 * time.Second)
			}
		}
	}()

	return nil
}

// StopSync signals the scanner to stop and waits for it to actually stop
func (s *Scanner) StopSync() {
	s.scanMu.Lock()

	if !s.scanning {
		s.scanMu.Unlock()
		s.logger.Debug().Msg("scanner not running, nothing to stop")
		return
	}

	s.logger.Info().Msg("stopping scanner synchronously")

	// Create a done channel to signal when the goroutine has stopped
	doneChan := make(chan struct{})
	s.doneChan = doneChan // Store reference to done channel

	// Close the stop channel to signal the scanning goroutine to stop
	select {
	case <-s.stopChan:
		// Channel already closed, do nothing
		s.logger.Debug().Msg("stop channel already closed")
	default:
		// Channel not closed, close it
		close(s.stopChan)
		s.logger.Debug().Msg("stop channel closed")
	}

	s.scanMu.Unlock()

	// Wait for the scanning goroutine to actually stop
	// We'll wait up to 10 seconds for it to stop gracefully
	select {
	case <-doneChan:
		s.logger.Debug().Msg("scanner stopped gracefully")
	case <-time.After(10 * time.Second):
		s.logger.Warn().Msg("scanner stop timeout, forcing stop")
		// Force the scanning state to false
		s.scanMu.Lock()
		s.scanning = false
		s.scanMu.Unlock()
	}
}

// Stop signals the scanner to stop scanning
func (s *Scanner) Stop() {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if !s.scanning {
		s.logger.Debug().Msg("scanner not running, nothing to stop")
		return
	}

	s.logger.Info().Msg("stopping scanner")

	// Close the stop channel to signal the scanning goroutine to stop
	select {
	case <-s.stopChan:
		// Channel already closed, do nothing
		s.logger.Debug().Msg("stop channel already closed")
	default:
		// Channel not closed, close it
		close(s.stopChan)
		s.logger.Debug().Msg("stop channel closed")
	}

	// Don't set scanning to false here - let the goroutine do it
	// This prevents race conditions where Start could be called before the goroutine finishes
}

// IsScanning returns whether the scanner is currently scanning
func (s *Scanner) IsScanning() bool {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()
	return s.scanning
}

// RescanFromHeight resets the last scan height and rescans from the specified height
func (s *Scanner) RescanFromHeight(height uint64) error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if s.scanning {
		s.logger.Warn().Msg("cannot rescan while scanning is in progress")
		return fmt.Errorf("cannot rescan while scanning is in progress")
	}

	oldHeight := s.lastScanHeight
	s.lastScanHeight = height - 1 // Set to height-1 so scanning starts from the specified height

	s.logger.Info().
		Uint64("old_height", oldHeight).
		Uint64("new_height", height).
		Msg("reset scan height for rescan")

	return nil
}

// ForceRescanFromHeight performs a complete rescan from the specified height, clearing all UTXOs
func (s *Scanner) ForceRescanFromHeight(height uint64) error {
	s.scanMu.Lock()
	defer s.scanMu.Unlock()

	if s.scanning {
		s.logger.Warn().Msg("cannot force rescan while scanning is in progress")
		return fmt.Errorf("cannot force rescan while scanning is in progress")
	}

	// Clear all UTXOs
	oldUTXOCount := len(s.allOwnedUTXOs)
	s.allOwnedUTXOs = nil

	oldHeight := s.lastScanHeight
	s.lastScanHeight = height - 1 // Set to height-1 so scanning starts from the specified height

	s.logger.Info().
		Uint64("old_height", oldHeight).
		Uint64("new_height", height).
		Int("cleared_utxos", oldUTXOCount).
		Msg("force rescan: cleared UTXOs and reset scan height")

	return nil
}
