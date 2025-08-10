package wallet

import (
	"fmt"
	"sync"

	"github.com/setavenger/blindbit-lib/wallet"
)

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
