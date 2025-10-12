package scanning

import (
	"context"

	"github.com/setavenger/blindbit-desktop/internal/controller"
	"github.com/setavenger/blindbit-lib/networking/v2connect"
	"github.com/setavenger/blindbit-lib/scanning/scannerv2"
	"github.com/setavenger/blindbit-lib/wallet"
	"github.com/setavenger/go-bip352"
)

type ScannerInternal struct {
	// todo: fully replace with logic from blindbit-lib
	libScanner   *scannerv2.ScannerV2
	oracleClient *v2connect.OracleClient
	wallet       *wallet.Wallet
}

func NewInternalScanner(m *controller.Manager) (*ScannerInternal, error) {
	oracleClient, err := v2connect.NewClient(context.TODO(), m.OracleAddress)
	if err != nil {
		return nil, err
	}

	// we only use change labels for now
	labels := []*bip352.Label{m.Wallet.GetLabel(0)}

	libScanner := scannerv2.NewScannerV2(
		oracleClient,
		m.Wallet.SecretKeyScan,
		m.Wallet.PubKeySpend,
		labels,
	)

	scanner := &ScannerInternal{
		libScanner:   libScanner,
		oracleClient: oracleClient,
		wallet:       m.Wallet,
	}

	return scanner, err
}

func (s *ScannerInternal) Scan(ctx context.Context, startHeight, endHeight uint32) error {

	go func() {
		err := s.libScanner.Scan(ctx, startHeight, endHeight)
	}()

	select {}

}
