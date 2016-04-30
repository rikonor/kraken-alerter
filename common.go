package krakenalerter

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const (
	defaultLowerPriceBound = 0.0
	defaultUpperPriceBound = 10.0
)

var priceBoundsMux = sync.Mutex{}
var lowerPriceBound = defaultLowerPriceBound
var upperPriceBound = defaultUpperPriceBound

var alertsEnabledMux = sync.Mutex{}
var alertsEnabled bool

type TradingPairName string

const (
	TradingPairETHXBT TradingPairName = "XETHXXBT"
)

type TradingPairTickerInfo struct {
	Open   string   `json:"o"`
	Last   []string `json:"c"`
	Low    []string `json:"l"`
	High   []string `json:"h"`
	Ask    []string `json:"a"`
	Bid    []string `json:"b"`
	Volume []string `json:"v"`
}

func currTime() string {
	return time.Now().Format(time.RFC3339)
}

func unmarshalInterfaceToTradingPairTickerInfo(in interface{}) (TradingPairTickerInfo, error) {
	var tpi TradingPairTickerInfo

	// Remarshal the interface{}
	b, err := json.Marshal(in)
	if err != nil {
		fmt.Println("Error:", err)
		return tpi, err
	}

	// Unmarshal the json into the struct
	err = json.Unmarshal(b, &tpi)
	if err != nil {
		fmt.Println("Error:", err)
		return tpi, err
	}

	return tpi, nil
}
