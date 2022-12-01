package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/MixinNetwork/bot-api-go-client"
	"strings"
)

var Mtg4SwapGroup = [5]string{
	"a753e0eb-3010-4c4a-a7b2-a7bda4063f62",
	"099627f8-4031-42e3-a846-006ee598c56e",
	"aefbfd62-727d-4424-89db-ae41f75d2e04",
	"d68ca71f-0e2c-458a-bb9c-1d6c2eed2497",
	"e4bc0740-f8fe-418c-ae1b-32d9926f5863",
}

func filterSnapshots(totalSnapshots []*bot.Snapshot) []*bot.Snapshot {
	var filteredSnapshots []*bot.Snapshot

	for i, s := range totalSnapshots {
		if s.Type == "deposit" {
			s.Type = "Deposit"
			filteredSnapshots = append(filteredSnapshots, s)
			continue
		}

		if s.Type == "transfer" && s.OpponentId == WithdrawalBot && s.Memo != "" {
			memo, err := base64.RawURLEncoding.DecodeString(s.Memo)
			if err != nil {
				continue
			}

			arr := strings.Split(string(memo), "~~")
			if arr[0] != "" && arr[2] != "" && (s.AssetId != EosAssetId || arr[1] != "") {
				s.Type = "Withdraw"
				s.Amount = s.Amount[1:]
				filteredSnapshots = append(filteredSnapshots, s)
				continue
			}
		}

		if s.Type == "transfer" && s.OpponentId == MixPayBot {
			memo, err := base64.RawURLEncoding.DecodeString(s.Memo)
			if err != nil {
				continue
			}

			arr := strings.Split(string(memo), "|")
			if arr[0] == "Swap" {
				next := totalSnapshots[i+1]
				nextMemo, err := base64.RawURLEncoding.DecodeString(next.Memo)
				if err != nil {
					continue
				}
				nextArr := strings.Split(string(nextMemo), "|")

				if nextArr[0] == "PM" && next.CreatedAt.Sub(s.CreatedAt).Minutes() < 1 {
					s.Type = "swap"
					s.Amount = s.Amount[1:]
					s.Memo = "mixpay|" + next.AssetId + "|" + next.Amount
					filteredSnapshots = append(filteredSnapshots, s)
				}
			}
		}

		if s.Type == "raw" && s.OpponentMultisigThreshold == 3 {
			next := totalSnapshots[i+1]

			var receivers [5]string
			copy(receivers[:], s.OpponentMultisigReceivers[:5])

			if receivers == Mtg4SwapGroup && next.CreatedAt.Sub(s.CreatedAt).Minutes() < 1 && next.OpponentMultisigThreshold == 0 && next.OpponentMultisigReceivers == nil {
				outMemo, err := base64.StdEncoding.DecodeString(next.Memo)
				if err != nil {
					continue
				}

				var memo struct {
					S string `json:"s"`
					T string `json:"t"`
				}
				err = json.Unmarshal(outMemo, &memo)
				if err != nil {
					continue
				}

				if memo.S == "4swapTrade" {
					s.Type = "Swap"
					s.Amount = s.Amount[1:]
					s.Memo = "4swap|" + next.AssetId + "|" + next.Amount
					filteredSnapshots = append(filteredSnapshots, s)
				}
			}
		}
	}

	// reverse
	for i := 0; i < len(filteredSnapshots); i++ {
		j := len(filteredSnapshots) - 1 - i
		if i >= j {
			break
		}

		tmp := filteredSnapshots[j]
		filteredSnapshots[j] = filteredSnapshots[i]
		filteredSnapshots[i] = tmp
	}
	return filteredSnapshots
}

func fetchHistoricalPriceForTx(record *TxRecord) (*TxRecord, error) {
	price, err := fetchHistoricalPrice(record.Asset, record.Timestamp)
	if err != nil {
		return nil, err
	}
	record.AssetPriceBack = price.PriceUSD

	if record.Type == "swap" {
		swapAssetPrice, err := fetchHistoricalPrice(record.SwapAsset, record.Timestamp)
		if err != nil {
			return nil, err
		}
		record.SwapAssetPriceBack = swapAssetPrice.PriceUSD
	}

	return record, nil
}
