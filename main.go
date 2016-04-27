package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
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

TODO
1. Setup proper logging
2. Create an additional cli tool to setting up bounds and enabling alerts
3. CLI tool could also add new trading pairs
4. Add alert to SMS about failures

*/

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
	To := *phoneToAlert

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

		fmt.Printf("%s Last Price for %s: %s\n", currTime(), TradingPairETHXBT, lastPrice)

		// Check if the price should trigger an alert
		fLastPrice, err := strconv.ParseFloat(lastPrice, 64)
		if err != nil {
			fmt.Println("Failed to convert lastPrice to float64:", err)
			return
		}

		priceBoundsMux.Lock()
		if lowerPriceBound < fLastPrice && fLastPrice < upperPriceBound {
			priceBoundsMux.Unlock()
			fmt.Printf("Price is within desired bounds: %f < %f < %f\n", lowerPriceBound, fLastPrice, upperPriceBound)
			time.Sleep(queryFrequency)
			continue
		}
		priceBoundsMux.Unlock()

		// Send SMS
		fmt.Printf("Price is outside of desired bounds (%f). Sending alert\n", fLastPrice)
		alertsEnabledMux.Lock()
		if alertsEnabled {
			priceSMSAlert(twilioAPI, TradingPairETHXBT, lastPrice)

			fmt.Println("Disabling alerts until acknoweldged")
			alertsEnabled = false
		} else {
			fmt.Println("Alerts are disabled")
		}
		alertsEnabledMux.Unlock()

		time.Sleep(queryFrequency)
	}
}

// HTTP Handlers
func setUpperPriceBoundsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Error: Only POST accepted")
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	// Extract new bound
	newUpperBound, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Error: setUpperBound requires float64")
		return
	}

	priceBoundsMux.Lock()
	upperPriceBound = newUpperBound
	priceBoundsMux.Unlock()

	fmt.Fprint(w, "Success")
}

func setLowerPriceBoundsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Error: Only POST accepted")
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	// Extract new bound
	newLowerBound, err := strconv.ParseFloat(string(b), 64)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Error: setLowerBound requires float64")
		return
	}

	priceBoundsMux.Lock()
	lowerPriceBound = newLowerBound
	priceBoundsMux.Unlock()

	fmt.Fprint(w, "Success")
}

func setAlertsEnabledAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Error: Only POST accepted")
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, err)
		return
	}

	// Extract new bound
	newAlertsEnabled, err := strconv.ParseBool(string(b))
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Error: setLowerBound requires bool")
		return
	}

	alertsEnabledMux.Lock()
	alertsEnabled = newAlertsEnabled
	alertsEnabledMux.Unlock()

	fmt.Fprint(w, "Success")
}

func currTime() string {
	return time.Now().Format(time.RFC3339)
}

// Kraken API Credentials
var krakenAPIKey = flag.String("kraken-key", "", "Kraken API key")
var krakenAPISecret = flag.String("kraken-secret", "", "Kraken API secret")

// Twilio API Credentials
var twilioAccountSid = flag.String("twilio-sid", "", "Twilio Account SID")
var twilioAuthToken = flag.String("twilio-token", "", "Twilio Authentication Token")

// Phone to alert
var phoneToAlert = flag.String("phone", "", "Phone to alert")

func main() {
	flag.Parse()

	// Phone is mandatory
	if *phoneToAlert == "" {
		fmt.Println("Please provide a phone number to alert")
		os.Exit(1)
	}

	fmt.Println(currTime(), "Starting Kraken-Alerter")

	// Create Kraken API
	krakenAPI := kraken.NewKrakenApi(*krakenAPIKey, *krakenAPISecret)

	// Create Twilio API
	twilioAPI := twilio.NewClient(*twilioAccountSid, *twilioAuthToken, nil)

	// Setup HTTP Server
	http.HandleFunc("/setUpperPriceBound", setUpperPriceBoundsAPI)
	http.HandleFunc("/setLowerPriceBound", setLowerPriceBoundsAPI)
	http.HandleFunc("/setAlertsEnabled", setAlertsEnabledAPI)
	go http.ListenAndServe(":8080", nil)

	queryPriceAndSendAlerts(krakenAPI, twilioAPI)
}
