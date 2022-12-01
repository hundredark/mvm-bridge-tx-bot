package main

import (
	"context"
	"encoding/json"
	"net/url"
	"time"

	"github.com/MixinNetwork/bot-api-go-client"
)

type UserKey struct {
	ClientID   string `json:"client_id"`
	SessionID  string `json:"session_id"`
	PrivateKey string `json:"private_key"`
}

type RegisteredUser struct {
	UserID    string  `json:"user_id"`
	SessionID string  `json:"session_id"`
	FullName  string  `json:"full_name"`
	CreatedAt string  `json:"created_at"`
	Key       UserKey `json:"key"`
	Contract  string  `json:"contract"`
}

type HistoricalPrice struct {
	Type     string `json:"type"`
	PriceBTC string `json:"price_btc"`
	PriceUSD string `json:"price_usd"`
}

type HistoricalPriceResponse struct {
	Data  *HistoricalPrice `json:"data"`
	Error *bot.Error       `json:"error"`
}

func fetchMvmUser(address string) (*RegisteredUser, error) {
	var path = BridgeApi + "/users"
	data, err := json.Marshal(map[string]interface{}{
		"public_key": address,
	})
	body, err := Request("POST", path, data)
	if err != nil {
		return nil, err
	}

	var resp struct {
		User RegisteredUser `json:"user"`
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp.User, nil
}

func fetchTotalSnapshots(ctx context.Context, user *RegisteredUser) []*bot.Snapshot {
	var totalSnapshots []*bot.Snapshot
	var offset = ""

	for {
		snapshots, err := bot.Snapshots(ctx, 500, offset, "", "asc", user.UserID, user.SessionID, user.Key.PrivateKey)
		if err != nil {
			break
		}

		totalSnapshots = append(totalSnapshots, snapshots...)

		if len(snapshots) == 500 {
			offset = snapshots[499].CreatedAt.Format(time.RFC3339)
		} else {
			break
		}
	}

	return totalSnapshots
}

func fetchHistoricalPrice(asset string, offset time.Time) (*HistoricalPrice, error) {
	values := url.Values{}
	values.Add("asset", asset)
	values.Add("offset", offset.Format(time.RFC3339))

	endpoint := MixinApi + "/network/ticker?" + values.Encode()
	body, err := Request("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}

	var resp HistoricalPriceResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, err
	}
	if resp.Error != nil {
		return nil, err
	}
	return resp.Data, nil
}
