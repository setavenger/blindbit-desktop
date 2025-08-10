package wallet

import (
	"fmt"
	"sync"
	"time"

	"github.com/setavenger/blindbit-lib/networking"
	"github.com/setavenger/blindbit-lib/wallet"
	netscan "github.com/setavenger/blindbit-scan/pkg/networking"
	"github.com/setavenger/blindbit-scan/pkg/scan"
)

func (s *Scanner) FinishBlock(data *BlockData) error {
	height := data.Height
	ownedUTXOs := data.OwnedUTXOs

	err := s.MarkSpentUTXOs(data)
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
	OwnedUTXOs  []*wallet.OwnedUTXO
}

func (s *Scanner) BlockFetcher(height uint64) (*BlockData, error) {
	fetchStart := time.Now()
	defer func() {
		s.logger.Debug().Uint64("height", height).
			Dur("fetch_duration", time.Since(fetchStart)).
			Msg("data fetched")
	}()

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
	processStart := time.Now()
	defer func() {
		s.logger.Debug().Uint64("height", data.Height).
			Dur("block_process_duration", time.Since(processStart)).
			Msg("block processed")
	}()

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
