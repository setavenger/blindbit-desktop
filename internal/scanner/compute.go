package scanner

import (
	"github.com/setavenger/blindbit-lib/utils"
	"github.com/setavenger/go-bip352"
)

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
