//go:build integration
// +build integration

package ledger

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/base"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/rpc"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/types"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
)

func TestGetStatementFromLog(t *testing.T) {
	bn := uint64(9279453)
	txid := uint64(208)
	log := types.SimpleLog{
		Address: base.HexToAddress("0x6b175474e89094c44da98b954eedeac495271d0f"),
		Topics: []base.Hash{
			transferTopic,
			base.HexToHash("0xf503017d7baf7fbc0fff7492b751025c6a78179b"),
			base.HexToHash("0x1212121212121212121212121212121212121212"),
		},
		Data:             "0xa",
		BlockNumber:      bn,
		TransactionIndex: txid,
	}
	conn := rpc.TempConnection(utils.GetTestChain())
	l := NewLedger(
		conn,
		base.HexToAddress("0xf503017d7baf7fbc0fff7492b751025c6a78179b"),
		0,
		utils.NOPOS,
		true,
		false,
		false,
		false,
		false,
		nil,
	)
	tx := types.SimpleTransaction{
		BlockNumber:      bn,
		TransactionIndex: txid,
		Receipt:          &types.SimpleReceipt{},
	}
	l.theTx = &tx
	apps := make([]types.SimpleAppearance, 0, 100)
	apps = append(apps, types.SimpleAppearance{
		BlockNumber:      uint32(bn),
		TransactionIndex: uint32(txid),
	})
	l.SetContexts("mainnet", apps)
	s, _ := l.getStatementsFromLog(conn, &log)
	b, _ := json.MarshalIndent(s, "", "  ")
	fmt.Println(string(b))
	fmt.Println("reconciled:", s.Reconciled())
}
