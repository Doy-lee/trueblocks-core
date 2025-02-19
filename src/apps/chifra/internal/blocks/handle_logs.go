// Copyright 2021 The TrueBlocks Authors. All rights reserved.
// Use of this source code is governed by a license that can
// be found in the LICENSE file.

package blocksPkg

import (
	"context"
	"fmt"
	"sort"

	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/articulate"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/identifiers"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/logger"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/output"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/types"
	"github.com/TrueBlocks/trueblocks-core/src/apps/chifra/pkg/utils"
)

func (opts *BlocksOptions) HandleLogs() error {
	chain := opts.Globals.Chain
	testMode := opts.Globals.TestMode
	nErrors := 0

	abiCache := articulate.NewAbiCache(opts.Conn, opts.Articulate)
	logFilter := types.NewLogFilter(opts.Emitter, opts.Topic)

	ctx, cancel := context.WithCancel(context.Background())
	fetchData := func(modelChan chan types.Modeler[types.RawLog], errorChan chan error) {
		apps, _, err := identifiers.IdsToApps(chain, opts.BlockIds)
		if err != nil {
			errorChan <- err
			cancel()
		}

		if sliceOfMaps, cnt, err := types.AsSliceOfMaps[types.SimpleTransaction](apps, false); err != nil {
			errorChan <- err
			cancel()

		} else if cnt == 0 {
			errorChan <- fmt.Errorf("no blocks found for the query")
			cancel()

		} else {
			bar := logger.NewBar(logger.BarOptions{
				Enabled: !testMode && !utils.IsTerminal(),
				Total:   int64(cnt),
			})

			for _, thisMap := range sliceOfMaps {
				for app := range thisMap {
					thisMap[app] = new(types.SimpleTransaction)
				}

				iterFunc := func(app types.SimpleAppearance, value *types.SimpleTransaction) error {
					if value.Receipt == nil {
						value.Receipt = &types.SimpleReceipt{}
					}

					bn := uint64(app.BlockNumber)
					ts := opts.Conn.GetBlockTimestamp(bn)
					if logs, err := opts.Conn.GetLogsByNumber(bn, ts); err != nil {
						delete(thisMap, app)
						return fmt.Errorf("block at %d returned an error: %w", bn, err)

					} else if len(logs) == 0 {
						delete(thisMap, app)
						return fmt.Errorf("block at %d has no logs", bn)

					} else {
						l := make([]types.SimpleLog, 0, len(logs))
						for index := range logs {
							if opts.Articulate {
								if err = abiCache.ArticulateLog(&logs[index]); err != nil {
									errorChan <- err // continue even with an error
								}
							}
							l = append(l, logs[index])
						}
						value.Receipt.Logs = append(value.Receipt.Logs, l...)
						bar.Tick()
						return nil
					}
				}

				iterErrorChan := make(chan error)
				iterCtx, iterCancel := context.WithCancel(context.Background())
				defer iterCancel()
				go utils.IterateOverMap(iterCtx, iterErrorChan, thisMap, iterFunc)
				for err := range iterErrorChan {
					if !testMode || nErrors == 0 {
						errorChan <- err
						nErrors++
					}
				}

				items := make([]types.SimpleLog, 0, len(thisMap))
				for _, tx := range thisMap {
					items = append(items, tx.Receipt.Logs...)
				}
				sort.Slice(items, func(i, j int) bool {
					if items[i].BlockNumber == items[j].BlockNumber {
						if items[i].TransactionIndex == items[j].TransactionIndex {
							return items[i].LogIndex < items[j].LogIndex
						}
						return items[i].TransactionIndex < items[j].TransactionIndex
					}
					return items[i].BlockNumber < items[j].BlockNumber
				})

				for _, item := range items {
					if !logFilter.PassesFilter(&item) {
						continue
					}
					modelChan <- &item
				}
			}
			bar.Finish(true /* newLine */)
		}
	}

	extra := map[string]interface{}{
		"articulate": opts.Articulate,
	}

	return output.StreamMany(ctx, fetchData, opts.Globals.OutputOptsWithExtra(extra))
}
