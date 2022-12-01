package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MixinNetwork/mixin/domains/ethereum"
	"github.com/MixinNetwork/mixin/logger"
	"github.com/dimfeld/httptreemux"
	"github.com/unrolled/render"
)

var (
	ctx context.Context
)

type TxRecord struct {
	Type           string    `json:"type"`
	Amount         string    `json:"amount"`
	Asset          string    `json:"asset"`
	AssetPriceBack string    `json:"asset_price_back"`
	Timestamp      time.Time `json:"timestamp"`

	SwapSource         string `json:"swap_source,omitempty"`
	SwapAsset          string `json:"swap_asset,omitempty"`
	SwapAmount         string `json:"swap_amount,omitempty"`
	SwapAssetPriceBack string `json:"swap_asset_price_back,omitempty"`
}

func StartHTTP(c context.Context) error {
	ctx = c

	router := httptreemux.New()
	router.GET("/txs/:address", getTxs)
	handler := handleCORS(router)
	handler = handleLog(handler)
	return http.ListenAndServe(fmt.Sprintf(":%d", HTTPPort), handler)
}

func handleCORS(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			handler.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Add("Access-Control-Allow-Headers", "Content-Type,X-Request-ID")
		w.Header().Set("Access-Control-Allow-Methods", "OPTIONS,GET,POST,DELETE")
		w.Header().Set("Access-Control-Max-Age", "600")
		if r.Method == "OPTIONS" {
			render.New().JSON(w, http.StatusOK, map[string]interface{}{})
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

func handleLog(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger.Verbosef("ServeHTTP(%v)", *r)
		handler.ServeHTTP(w, r)
	})
}

func getTxs(w http.ResponseWriter, r *http.Request, params map[string]string) {
	address := params["address"]
	logger.Verbosef("getTxs for %s", address)
	err := ethereum.VerifyAddress(address)
	if err != nil {
		logger.Verbosef("VerifyAddress(%s) failed, error: %v", address, err)
		render.New().JSON(w, http.StatusOK, map[string]interface{}{
			"error": "Invalid ethereum address",
		})
		return
	}

	user, err := fetchMvmUser(address)
	if err != nil {
		logger.Verbosef("register (%s) failed, error: %v", address, err)
		render.New().JSON(w, http.StatusOK, map[string]interface{}{"error": err})
		return
	}

	snapshots := fetchTotalSnapshots(ctx, user)
	txs := filterSnapshots(snapshots)

	var records = make([]*TxRecord, len(txs))
	wg := sync.WaitGroup{}

	for i, s := range txs {
		wg.Add(1)

		r := TxRecord{
			Type:      s.Type,
			Amount:    s.Amount,
			Asset:     s.AssetId,
			Timestamp: s.CreatedAt,
		}
		if s.Type == "swap" {
			arr := strings.Split(s.Memo, "|")
			r.SwapSource = arr[0]
			r.SwapAsset = arr[1]
			r.SwapAmount = arr[2]
		}

		i := i
		go func(r TxRecord, i int) {
			record, err := fetchHistoricalPriceForTx(&r)
			if err != nil {
				return
			}

			records[i] = record
			wg.Done()
		}(r, i)
	}

	wg.Wait()

	render.New().JSON(w, http.StatusOK, map[string]interface{}{
		"data": records,
	})
}
