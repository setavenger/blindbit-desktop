package scanner

import (
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil/gcs"
	"github.com/btcsuite/btcd/btcutil/gcs/builder"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/rs/zerolog"
	"github.com/setavenger/blindbit-lib/networking"
	"github.com/setavenger/blindbit-lib/wallet"
	netscan "github.com/setavenger/blindbit-scan/pkg/networking"
	"github.com/setavenger/blindbit-scan/pkg/scan"
	"github.com/setavenger/go-bip352"
	"github.com/setavenger/go-electrum/electrum"
)

// Scanner handles the scanning process
type Scanner struct {
	Client         networking.BlindBitConnector
	electrum       *electrum.Client
	logger         *zerolog.Logger
	wallet         *wallet.Wallet
	scanSecret     [32]byte
	spendSecret    [32]byte
	labels         []*bip352.Label
	birthHeight    uint64
	lastScanHeight uint64

	allOwnedUTXOs      []*wallet.OwnedUTXO
	progressCallback   func(uint64)
	utxoUpdateCallback func() // New callback for UTXO updates

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
	wallet *wallet.Wallet,
	logger *zerolog.Logger,
	labelCount int, // Add label count parameter
) (*Scanner, error) {
	// Create BlindBit client
	blindBitClient := &networking.ClientBlindBit{BaseURL: oracleURL}

	// Create Electrum client - disabled for now
	var electrumClient *electrum.Client = nil

	// Generate labels properly with the specified count
	labels, err := generateLabels(wallet, labelCount)
	if err != nil {
		return nil, fmt.Errorf("failed to generate labels: %w", err)
	}

	// Use the logger directly (it already has caller information)
	scanner := &Scanner{
		Client:         blindBitClient,
		electrum:       electrumClient,
		logger:         logger,
		wallet:         wallet,
		scanSecret:     wallet.SecretKeyScan,
		spendSecret:    wallet.SecretKeySpend,
		labels:         labels,
		birthHeight:    840000, // Default birth height
		lastScanHeight: 0,
		stopChan:       make(chan struct{}),
	}

	return scanner, nil
}

func NewScannerFull(
	bbClient networking.BlindBitConnector,
	electrumClient *electrum.Client,
	wallet *wallet.Wallet,
	logger *zerolog.Logger,
	labels []*bip352.Label,
	birthHeight uint64,
	lastScanHeight uint64,
	stopChan chan struct{},
) (*Scanner, error) {
	// Use the logger directly (it already has caller information)
	scanner := &Scanner{
		Client:         bbClient,
		electrum:       electrumClient,
		logger:         logger,
		wallet:         wallet,
		scanSecret:     wallet.SecretKeyScan,
		spendSecret:    wallet.SecretKeySpend,
		labels:         labels,
		birthHeight:    birthHeight, // Default birth height
		lastScanHeight: lastScanHeight,
		stopChan:       stopChan,
	}

	return scanner, nil
}

// generateLabels creates labels for the wallet
// Will always create the chnage label labelCount is for the labels to be created in addtion
func generateLabels(wallet *wallet.Wallet, labelCount int) ([]*bip352.Label, error) {

	// we always need the change label m=0
	// if 21 labels are requested we count 21 desired labels additionally to the change label
	// change label is not intended to be "usable" for the user
	labelCount++

	// Generate labels based on the specified count
	labels := make([]*bip352.Label, 0, labelCount)

	// Generate the specified number of labels
	for i := range labelCount {
		label, err := bip352.CreateLabel(wallet.SecretKeyScan.ToArrayPtr(), uint32(i))
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

// SetUTXOUpdateCallback sets a callback function for UTXO updates
func (s *Scanner) SetUTXOUpdateCallback(callback func()) {
	s.utxoUpdateCallback = callback
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
	return s.wallet.PubKeySpend
}

// ScanBlock scans a single block for UTXOs using the scan package
func (s *Scanner) ScanBlock(blockHeight uint64) ([]*wallet.OwnedUTXO, error) {
	s.logger.Info().Uint64("height", blockHeight).Msg("scanning block")

	// Time the entire block scan
	blockStart := time.Now()

	// Time GetTweaks
	tweakStart := time.Now()
	tweaks, err := s.Client.GetTweaks(blockHeight, 1000) // Default dust limit
	if err != nil {
		return nil, fmt.Errorf("failed to get tweaks: %w", err)
	}
	tweakDuration := time.Since(tweakStart)
	s.logger.Debug().
		Uint64("height", blockHeight).
		Dur("get_tweaks", tweakDuration).
		Int("tweak_count", len(tweaks)).
		Msg("timing")

	if len(tweaks) == 0 {
		return nil, nil
	}

	// OPTIMIZATION: Precompute potential outputs and check against filter first
	filterStart := time.Now()
	potentialOutputs := s.precomputePotentialOutputs(tweaks)
	filterDuration := time.Since(filterStart)
	s.logger.Debug().
		Uint64("height", blockHeight).
		Dur("precompute_outputs", filterDuration).
		Int("potential_outputs", len(potentialOutputs)).
		Msg("timing")

	// Check filter to see if any of our outputs might be in this block
	if len(potentialOutputs) > 0 {
		// if len(potentialOutputs) > 0 {
		filterCheckStart := time.Now()
		filterData, err := s.Client.GetFilter(blockHeight, networking.NewUTXOFilterType)
		if err != nil {
			s.logger.Err(err).Msg("failed to get UTXO filter")
			// Continue without filter optimization
		} else {
			isMatch, err := matchFilter(filterData.Data, filterData.BlockHash, potentialOutputs)
			if err != nil {
				s.logger.Err(err).Msg("failed to match filter")
				// Continue without filter optimization
			} else if !isMatch {
				// No potential outputs in this block, skip expensive operations
				filterCheckDuration := time.Since(filterCheckStart)
				totalDuration := time.Since(blockStart)
				s.logger.Info().Uint64("height", blockHeight).
					Dur("total", totalDuration).
					Dur("get_tweaks", tweakDuration).
					Dur("precompute_outputs", filterDuration).
					Dur("filter_check", filterCheckDuration).
					Int("tweaks", len(tweaks)).
					Int("potential_outputs", len(potentialOutputs)).
					Msg("block skipped - no potential outputs found")
				return nil, nil
			}
			filterCheckDuration := time.Since(filterCheckStart)
			s.logger.Debug().
				Uint64("height", blockHeight).
				Dur("filter_check", filterCheckDuration).
				Msg("filter matched, continuing with scan")
		}
	}

	// Time GetUTXOs
	utxoStart := time.Now()
	utxos, err := s.Client.GetUTXOs(blockHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to get UTXOs: %w", err)
	}
	utxoDuration := time.Since(utxoStart)
	s.logger.Debug().
		Uint64("height", blockHeight).
		Dur("get_utxos", utxoDuration).
		Int("utxo_count", len(utxos)).
		Msg("timing")

	// Time ScanDataOptimized
	scanStart := time.Now()
	// todo: cleanup type conversions

	utxosTransform := make([]*netscan.UTXOServed, len(utxos))
	for i := range utxos {
		v := utxos[i]
		utxosTransform[i] = &netscan.UTXOServed{
			Txid:         v.Txid,
			Vout:         v.Vout,
			Amount:       v.Amount,
			ScriptPubKey: v.ScriptPubKey,
			BlockHeight:  v.BlockHeight,
			BlockHash:    v.BlockHash,
			Timestamp:    v.Timestamp,
			Spent:        v.Spent,
		}
	}

	// todo: we are doing comptations several times over.
	// If we have a match we are doing the same step as in precomputtion
	ownedUTXOsScan, err := scan.ScanDataOptimized(s, utxosTransform, tweaks)
	if err != nil {
		return nil, fmt.Errorf("failed to scan data: %w", err)
	}

	ownedUTXOs := make([]wallet.OwnedUTXO, len(ownedUTXOsScan))
	for i := range ownedUTXOsScan {
		v := ownedUTXOsScan[i]
		ownedUTXOs[i] = wallet.OwnedUTXO{
			Txid:         v.Txid,
			Vout:         v.Vout,
			Amount:       v.Amount,
			PrivKeyTweak: v.PrivKeyTweak,
			PubKey:       v.PubKey,
			Timestamp:    v.Timestamp,
			State:        wallet.UTXOState(v.State),
			Label:        v.Label,
		}
	}

	scanDuration := time.Since(scanStart)
	s.logger.Debug().
		Uint64("height", blockHeight).
		Dur("scan_data", scanDuration).
		Int("found_utxos", len(ownedUTXOs)).
		Msg("timing")

	if len(ownedUTXOs) == 0 {
		// early function exit
		return nil, nil
	}

	// Time conversion
	convertStart := time.Now()
	// Convert from scan package format to our format
	var result []*wallet.OwnedUTXO
	for _, utxo := range ownedUTXOs {
		state := wallet.StateUnspent
		if utxo.State == wallet.StateSpent {
			state = wallet.StateSpent
		} else if utxo.State == wallet.StateUnconfirmedSpent {
			state = wallet.StateUnconfirmedSpent
		} else if utxo.State == wallet.StateUnconfirmed {
			state = wallet.StateUnconfirmed
		}

		result = append(result, &wallet.OwnedUTXO{
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
	convertDuration := time.Since(convertStart)

	// Log total block scan time and breakdown
	totalDuration := time.Since(blockStart)
	s.logger.Info().Uint64("height", blockHeight).
		Dur("total", totalDuration).
		Dur("get_tweaks", tweakDuration).
		Dur("precompute_outputs", filterDuration).
		Dur("get_utxos", utxoDuration).
		Dur("scan_data", scanDuration).
		Dur("convert", convertDuration).
		Int("tweaks", len(tweaks)).
		Int("utxos", len(utxos)).
		Int("found", len(result)).
		Msg("block scan timing breakdown")

	return result, nil
}

// precomputePotentialOutputs computes all possible output pubkeys for the given tweaks
func (s *Scanner) precomputePotentialOutputs(tweaks [][33]byte) [][]byte {
	precomputeStart := time.Now()
	defer func() {
		s.logger.Debug().Int("tweak_count", len(tweaks)).
			Dur("precompute_duration", time.Since(precomputeStart)).
			Msg("precomputation completed")
	}()

	var potentialOutputs [][]byte

	// Use a mutex to protect the shared slice
	var mu sync.Mutex

	// Process tweaks in parallel
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 24) // Limit concurrent goroutines

	for _, tweak := range tweaks {
		wg.Add(1)
		go func(tweak [33]byte) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			// Process this tweak
			tweakOutputs := s.processTweak(tweak)

			// Add results to shared slice
			mu.Lock()
			potentialOutputs = append(potentialOutputs, tweakOutputs...)
			mu.Unlock()
		}(tweak)
	}

	wg.Wait()

	return potentialOutputs
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
