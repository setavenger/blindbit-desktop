package wallet

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcutil/gcs"
	"github.com/btcsuite/btcd/btcutil/gcs/builder"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/rs/zerolog"
	"github.com/setavenger/blindbit-lib/networking"
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/setavenger/blindbit-lib/wallet"
	netscan "github.com/setavenger/blindbit-scan/pkg/networking"
	"github.com/setavenger/blindbit-scan/pkg/scan"
	"github.com/setavenger/go-bip352"
	"github.com/setavenger/go-electrum/electrum"
)

// Scanner handles the scanning process
type Scanner struct {
	client         networking.BlindBitConnector
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
		client:         blindBitClient,
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
		client:         bbClient,
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
	tweaks, err := s.client.GetTweaks(blockHeight, 1000) // Default dust limit
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
		filterData, err := s.client.GetFilter(blockHeight, networking.NewUTXOFilterType)
		if err != nil {
			s.logger.Err(err).Msg("failed to get UTXO filter")
			// Continue without filter optimization
		} else {
			isMatch, err := s.matchFilter(filterData.Data, filterData.BlockHash, potentialOutputs)
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
	utxos, err := s.client.GetUTXOs(blockHeight)
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
	var potentialOutputs [][]byte

	// Use a mutex to protect the shared slice
	var mu sync.Mutex

	// Process tweaks in parallel
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 13) // Limit concurrent goroutines

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

// processTweak processes a single tweak and returns all possible output pubkeys
func (s *Scanner) processTweak(tweak [33]byte) [][]byte {
	var outputs [][]byte

	var scanSecret [32]byte
	copy(scanSecret[:], s.scanSecret[:])

	var tweakBytes [33]byte
	copy(tweakBytes[:], tweak[:])

	sharedSecret, err := bip352.CreateSharedSecret(&tweak, &s.scanSecret, nil)
	if err != nil {
		return outputs // Return empty slice if there's an error
	}

	outputPubKey, err := bip352.CreateOutputPubKey(*sharedSecret, s.pubKeySpend33(), 0)
	if err != nil {
		return outputs // Return empty slice if there's an error
	}

	// Add base output
	outputs = append(outputs, outputPubKey[:])

	// Add label combinations
	for _, label := range s.labels {
		outputPubKey33 := utils.ConvertToFixedLength33(append([]byte{0x02}, outputPubKey[:]...))
		labelPotentialOutputPrep, err := bip352.AddPublicKeys(&outputPubKey33, &label.PubKey)
		if err != nil {
			continue
		}

		outputs = append(outputs, labelPotentialOutputPrep[1:])

		var negatedLabelPubKey [33]byte
		copy(negatedLabelPubKey[:], label.PubKey[:])
		err = bip352.NegatePublicKey(&negatedLabelPubKey)
		if err != nil {
			continue
		}

		labelPotentialOutputPrepNegated, err := bip352.AddPublicKeys(&outputPubKey33, &negatedLabelPubKey)
		if err != nil {
			continue
		}

		outputs = append(outputs, labelPotentialOutputPrepNegated[1:])
	}

	return outputs
}

// matchFilter checks if any values match the GCS filter (for UTXO filtering)
func (s *Scanner) matchFilter(nBytes []byte, blockHash [32]byte, values [][]byte) (bool, error) {
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

	var stopFlag bool

	// Don't reset allOwnedUTXOs - accumulate UTXOs across scan cycles
	// for i := startHeight; i <= chainTip; i++ {
	// 	// Check for stop signal before scanning each block
	// 	select {
	// 	case <-s.stopChan:
	// 		stopFlag = true
	// 		s.logger.Info().Msg("scanning stopped during block scan")
	// 		return nil
	// 	default:
	// 	}
	//
	// 	s.ProcessHeight(i, progressCallback)
	// }

	dataCollector := make(map[uint64]*BlockData, 10) // backlog to be collected
	errChan := make(chan error)
	dataChan := make(chan *BlockData, 4) // 4 might be random and unnecessary

	var mu sync.Mutex

	// fetch Routine
	go func() {
		semaphore := make(chan struct{}, 24) // Limit concurrent goroutines

		for i := startHeight; i <= chainTip; i++ {
			semaphore <- struct{}{} // Acquire semaphore
			select {
			case <-s.stopChan:
				stopFlag = true
				s.logger.Info().Msg("scanning stopped during block scan")
				return
			default:
			}

			// Check for stop signal before scanning each block
			if stopFlag {
				s.logger.Info().Msg("stop flag called")
				return
			}
			go func(height uint64) {
				defer func() { <-semaphore }() // Release semaphore
				data, err := s.BlockFetcher(height)
				if err != nil {
					s.logger.Err(err).Uint64("height", height).Msg("failed fetching data")
					errChan <- err
					return
				}
				dataChan <- data
			}(i)
		}
	}()

	// process routine
	for !stopFlag {
		// s.logger.Debug().Msg("waiting for fetched blocks")
		select {
		case blockData := <-dataChan:
			height := blockData.Height
			if height > s.lastScanHeight+1 {
				// store away as we need to process in order
				mu.Lock()
				dataCollector[height] = blockData
				mu.Unlock()
			}

			var ownedUTXOs []*wallet.OwnedUTXO
			ownedUTXOs, err = s.ProcessBlockData(blockData)
			if err != nil {
				s.logger.Err(err).Uint64("height", height).Msg("failed to process block data")
				return err
			}

			err = s.FinishBlock(height, ownedUTXOs)
			if err != nil {
				s.logger.Err(err).Uint64("height", height).Msg("failed to finish block")
				return err
			}

			s.lastScanHeight = height

			// Report progress via callback
			progressCallback(height)

			// Save progress every 100 blocks
			if height%100 == 0 {
				s.logger.Debug().Uint64("height", height).Msg("saving progress")
			}

			// we check if the next block is in the collector
			// if so we pull and process and try for the next height up again until one does not exist
			var foundNextBlock bool = true
			for foundNextBlock {
				height++
				if blockData, foundNextBlock = dataCollector[height]; !foundNextBlock {
					continue
				}
				var ownedUTXOs []*wallet.OwnedUTXO
				ownedUTXOs, err = s.ProcessBlockData(blockData)
				if err != nil {
					s.logger.Err(err).Uint64("height", height).Msg("failed to process block data")
					return err
				}

				err = s.FinishBlock(height, ownedUTXOs)
				if err != nil {
					s.logger.Err(err).Uint64("height", height).Msg("failed to finish block")
					return err
				}

				s.lastScanHeight = height

				// Report progress via callback
				progressCallback(height)

				// Save progress every 100 blocks
				if height%100 == 0 {
					s.logger.Debug().Uint64("height", height).Msg("saving progress")
				}
			}
		}
	}

	return nil
}

func (s *Scanner) ProcessHeight(i uint64, progressCallback func(uint64)) (err error) {
	blockStart := time.Now()
	s.logger.Debug().Uint64("height", i).Msg("scanning block")
	// Mark spent UTXOs first
	markStart := time.Now()
	err = s.MarkSpentUTXOs(i)
	if err != nil {
		s.logger.Err(err).Uint64("height", i).Msg("error marking spent UTXOs")
		// Continue scanning even if marking spent UTXOs fails
	}
	markDuration := time.Since(markStart)

	// Scan block for new UTXOs
	scanStart := time.Now()
	ownedUTXOs, err := s.ScanBlock(i)
	if err != nil {
		return fmt.Errorf("failed to scan block %d: %w", i, err)
	}
	scanDuration := time.Since(scanStart)

	// Process found UTXOs
	processStart := time.Now()
	if len(ownedUTXOs) > 0 {
		s.logger.Info().
			Uint64("height", i).
			Int("utxos_found", len(ownedUTXOs)).
			Msg("found UTXOs")
		added := s.addUTXOsSafely(ownedUTXOs)
		if added > 0 {
			s.logger.Info().Uint64("height", i).
				Int("utxos_added", added).
				Int("utxos_skipped", len(ownedUTXOs)-added).
				Msg("processed UTXOs")
			// Call UTXO update callback if new UTXOs were added
			if s.utxoUpdateCallback != nil {
				s.utxoUpdateCallback()
			}
		}
	}
	processDuration := time.Since(processStart)

	s.lastScanHeight = i

	// Report progress via callback
	progressCallback(i)

	// Save progress every 100 blocks
	if i%100 == 0 {
		s.logger.Debug().Uint64("height", i).Msg("saving progress")
	}

	// Log block timing summary
	totalBlockDuration := time.Since(blockStart)
	s.logger.Info().Uint64("height", i).
		Dur("total_block", totalBlockDuration).
		Dur("mark_spent", markDuration).
		Dur("scan_block", scanDuration).
		Dur("process_utxos", processDuration).
		Int("found_utxos", len(ownedUTXOs)).
		Msg("block processing timing summary")

	return
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

func (s *Scanner) FinishBlock(height uint64, ownedUTXOs []*wallet.OwnedUTXO) error {
	err := s.MarkSpentUTXOs(height)
	if err != nil {
		s.logger.Err(err).Msg("failed to mark utxos as spent")
		return err
	}
	if len(ownedUTXOs) > 0 {
		s.logger.Info().
			Uint64("height", height).
			Int("utxos_found", len(ownedUTXOs)).
			Msg("found UTXOs")
		added := s.addUTXOsSafely(ownedUTXOs)
		if added > 0 {
			s.logger.Info().Uint64("height", height).
				Int("utxos_added", added).
				Int("utxos_skipped", len(ownedUTXOs)-added).
				Msg("processed UTXOs")
			// Call UTXO update callback if new UTXOs were added
			if s.utxoUpdateCallback != nil {
				s.utxoUpdateCallback()
			}
		}
	}

	return nil
}

type BlockData struct {
	Height      uint64
	FilterNew   *networking.Filter
	FilterSpent *networking.Filter
	Tweaks      [][33]byte
}

func (s *Scanner) BlockFetcher(height uint64) (*BlockData, error) {
	s.logger.Debug().Uint64("height", height).Msg("fetching data")
	var wg sync.WaitGroup
	wg.Add(3)

	// can this end in a deadlock if the waitgroup is waiting but more than one ends in an error branch?
	errChan := make(chan error, 4)

	var filterNew, filterSpent *networking.Filter
	var tweaks [][33]byte

	go func() {
		defer wg.Done()
		var err error
		filterNew, err = s.client.GetFilter(height, networking.NewUTXOFilterType)
		if err != nil {
			s.logger.Err(err).Msg("failed to get new utxos filter")
			errChan <- err
		}
	}()
	go func() {
		defer wg.Done()
		var err error
		filterSpent, err = s.client.GetFilter(height, networking.SpentOutpointsFilterType)
		if err != nil {
			s.logger.Err(err).Msg("failed to get spent outpoints filter")
			errChan <- err
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		tweaks, err = s.client.GetTweaks(height, 0)
		if err != nil {
			s.logger.Err(err).Msg("failed to pull tweaks")
			errChan <- err
		}
	}()

	wg.Wait()

	select {
	case err := <-errChan:
		return nil, err
	default:
		// do nothing
	}

	var out BlockData
	out.Height = height
	out.FilterNew = filterNew
	out.FilterSpent = filterSpent
	out.Tweaks = tweaks

	return &out, nil
}

func (s *Scanner) ProcessBlockData(data *BlockData) ([]*wallet.OwnedUTXO, error) {
	blockHeight := data.Height
	s.logger.Info().Uint64("height", blockHeight).Msg("processing block")

	// Time the entire block scan
	blockStart := time.Now()

	tweaks := data.Tweaks

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
		filterData := data.FilterNew
		filterCheckStart := time.Now()
		isMatch, err := s.matchFilter(filterData.Data, filterData.BlockHash, potentialOutputs)
		if err != nil {
			s.logger.Err(err).Msg("failed to match filter")
			// Continue without filter optimization
		} else if !isMatch {
			// No potential outputs in this block, skip expensive operations
			filterCheckDuration := time.Since(filterCheckStart)
			totalDuration := time.Since(blockStart)
			s.logger.Info().Uint64("height", blockHeight).
				Dur("total", totalDuration).
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

	// Time GetUTXOs
	utxoStart := time.Now()
	// keep in here, only called if we think we have a match
	utxos, err := s.client.GetUTXOs(blockHeight)
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

	// todo: we are doing computations several times over.
	// If we have a match we are doing the same step as in precomputtion
	ownedUTXOsScan, err := scan.ScanDataOptimized(s, utxosTransform, tweaks)
	if err != nil {
		return nil, fmt.Errorf("failed to scan data: %w", err)
	}

	ownedUTXOs := make([]*wallet.OwnedUTXO, len(ownedUTXOsScan))
	for i := range ownedUTXOsScan {
		v := ownedUTXOsScan[i]
		ownedUTXOs[i] = &wallet.OwnedUTXO{
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
		Dur("precompute_outputs", filterDuration).
		Dur("get_utxos", utxoDuration).
		Dur("scan_data", scanDuration).
		Dur("convert", convertDuration).
		Int("tweaks", len(tweaks)).
		Int("utxos", len(utxos)).
		Int("found", len(result)).
		Msg("block scan timing breakdown")

	return ownedUTXOs, err
}

// MarkSpentUTXOs marks UTXOs as spent based on the spent outpoints filter
func (s *Scanner) MarkSpentUTXOs(blockHeight uint64) error {
	markStart := time.Now()

	// Get the spent outpoints filter
	filterStart := time.Now()
	filter, err := s.client.GetFilter(blockHeight, networking.SpentOutpointsFilterType)
	if err != nil {
		s.logger.Err(err).Msg("failed to get spent outpoints filter")
		return err
	}
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
	index, err := s.client.GetSpentOutpointsIndex(blockHeight)
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
		Int("utxo_count", len(s.allOwnedUTXOs)).
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
			s.logger.Info().Msg("scanner stopped wow")

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
