package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"strconv"
	"time"

	kraken "github.com/Beldur/kraken-go-api-client"
	"github.com/subosito/twilio"
)

/*
Kraken Alerter

Goals:
1. Retreive XBT -> ETH exchange rate
2. Set checkpoints to alert me at
3. Send me an SMS when the alerts are raised

*/

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

type TradingPairTickerInfoGroup map[string]TradingPairTickerInfo

func UnmarshalInterfaceToTradingPairTickerInfo(in interface{}) (TradingPairTickerInfo, error) {
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

func getPairTickerInfo(api *kraken.KrakenApi, pair TradingPairName) (TradingPairTickerInfo, error) {
	var tpi TradingPairTickerInfo

	result, err := api.Query("Ticker", map[string]string{
		"pair": string(pair),
	})

	if err != nil {
		fmt.Println("Error:", err)
		return tpi, fmt.Errorf("API Query failed: %s", err)
	}

	tmp1 := result.(map[string]interface{})["XETHXXBT"]
	tpi, err = UnmarshalInterfaceToTradingPairTickerInfo(tmp1)
	if err != nil {
		fmt.Println("Error:", err)
		return tpi, fmt.Errorf("Failed to unmarshal API result: %s", err)
	}

	return tpi, nil
}

func priceSMSAlert(twilioAPI *twilio.Client, pairName TradingPairName, lastPrice string) {
	From := "+12033496456"
	To := "(203) 451-1578"

	msg := fmt.Sprintf("Last Price for %s: %s\n", pairName, lastPrice)
	_, _, err := twilioAPI.Messages.SendSMS(From, To, msg)
	if err != nil {
		fmt.Println("Failed to send SMS message:", err)
	}
}

func queryPriceAndSendAlerts(krakenAPI *kraken.KrakenApi, twilioAPI *twilio.Client) {
	queryFrequency := 10 * time.Second

	for {
		tpi, err := getPairTickerInfo(krakenAPI, TradingPairETHXBT)
		if err != nil {
			fmt.Println("Failed to get pair ticker info:", err)
			return
		}

		lastPrice := tpi.Last[0]

		fmt.Printf("Last Price for %s: %s\n", TradingPairETHXBT, lastPrice)

		// Check if the price should trigger an alert
		lowerPriceBound := 0.018
		upperPriceBound := 0.0181

		fLastPrice, err := strconv.ParseFloat(lastPrice, 64)
		if err != nil {
			fmt.Println("Failed to convert lastPrice to float64:", err)
			return
		}

		if lowerPriceBound < fLastPrice && fLastPrice < upperPriceBound {
			fmt.Printf("Price is within desired bounds: %f < %f < %f\n", lowerPriceBound, fLastPrice, upperPriceBound)
			time.Sleep(queryFrequency)
			continue
		}

		// Send SMS
		fmt.Printf("Price is outside of desired bounds (%f). Sending alert\n", fLastPrice)
		priceSMSAlert(twilioAPI, TradingPairETHXBT, lastPrice)

		time.Sleep(queryFrequency)
	}
}

// Kraken API Credentials
var krakenAPIKey = flag.String("kraken-key", "", "Kraken API key")
var krakenAPISecret = flag.String("kraken-secret", "", "Kraken API secret")

// Twilio API Credentials
var twilioAccountSid = flag.String("twilio-sid", "", "Twilio Account SID")
var twilioAuthToken = flag.String("twilio-token", "", "Twilio Authentication Token")

func main() {
	fmt.Println("Starting Kraken-Alerter")
	flag.Parse()

	// Create Kraken API
	krakenAPI := kraken.NewKrakenApi(*krakenAPIKey, *krakenAPISecret)

	// Create Twilio API
	twilioAPI := twilio.NewClient(*twilioAccountSid, *twilioAuthToken, nil)

	queryPriceAndSendAlerts(krakenAPI, twilioAPI)
}
