package krakenalerter

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	kraken "github.com/Beldur/kraken-go-api-client"
	"github.com/subosito/twilio"
)

type KrakenAlerter struct {
	KrakenAPI       *kraken.KrakenApi
	TwilioAPI       *twilio.Client
	Phone           string
	mu              sync.Mutex
	LowerPriceBound float64
	UpperPriceBound float64
	AlertsEnabled   bool
	LastPrice       float64
	QueryFrequency  time.Duration
}

func NewKrakenAlerter(twilioAPI *twilio.Client, phone string, lowerPriceBound, upperPriceBound float64, alertsEnabled bool) *KrakenAlerter {
	// Only using public methods (therefore no keys)
	krakenApi := kraken.NewKrakenApi("", "")

	return &KrakenAlerter{
		KrakenAPI:       krakenApi,
		TwilioAPI:       twilioAPI,
		Phone:           phone,
		LowerPriceBound: lowerPriceBound,
		UpperPriceBound: upperPriceBound,
		AlertsEnabled:   alertsEnabled,
		QueryFrequency:  10 * time.Second,
	}
}

func (ka *KrakenAlerter) SetPriceBounds(lowerPriceBound, upperPriceBound float64) {
	ka.mu.Lock()
	defer ka.mu.Unlock()

	ka.LowerPriceBound = lowerPriceBound
	ka.UpperPriceBound = upperPriceBound
}

func (ka *KrakenAlerter) SetLowerPriceBound(lowerPriceBound float64) {
	ka.SetPriceBounds(lowerPriceBound, ka.UpperPriceBound)
}

func (ka *KrakenAlerter) SetUpperPriceBound(upperPriceBound float64) {
	ka.SetPriceBounds(ka.LowerPriceBound, upperPriceBound)
}

func (ka *KrakenAlerter) SetAlertsEnabled(on bool) {
	ka.mu.Lock()
	defer ka.mu.Unlock()

	ka.AlertsEnabled = on
}

func (ka *KrakenAlerter) DisableAlerts() {
	ka.SetAlertsEnabled(false)
}

func (ka *KrakenAlerter) EnableAlerts() {
	ka.SetAlertsEnabled(true)
}

func (ka *KrakenAlerter) adjustBounds() {
	// 5%
	ka.SetPriceBounds(0.95*ka.LastPrice, 1.05*ka.LastPrice)
}

func (ka *KrakenAlerter) StartKrakenAlerter() {
	ka.queryPriceAndSendAlerts()
}

func (ka *KrakenAlerter) queryPriceAndSendAlerts() {
	for {
		tpi, err := ka.getPairTickerInfo()
		if err != nil {
			fmt.Println("Failed to get pair ticker info:", err)
			return
		}

		lastPrice, err := strconv.ParseFloat(tpi.Last[0], 64)
		if err != nil {
			fmt.Println("Failed to extract float value from last price:", err)
			return
		}
		ka.LastPrice = lastPrice

		fmt.Printf("%s Last Price for %s: %f\n", currTime(), "XETHXXBT", ka.LastPrice)

		if ka.LowerPriceBound < ka.LastPrice && ka.LastPrice < ka.UpperPriceBound {
			fmt.Printf("Price is within desired bounds: %f < %f < %f\n", ka.LowerPriceBound, ka.LastPrice, ka.UpperPriceBound)
			time.Sleep(ka.QueryFrequency)
			continue
		}

		// Send SMS
		fmt.Printf("Price is outside of desired bounds: %f < %f < %f. Sending alert\n", ka.LowerPriceBound, ka.LastPrice, ka.UpperPriceBound)
		if ka.AlertsEnabled {
			ka.priceSMSAlert()
		} else {
			fmt.Println("Alerts are disabled")
		}

		// If lastPrice was out of bounds, adjust the bounds
		ka.adjustBounds()

		time.Sleep(ka.QueryFrequency)
	}
}

func (ka *KrakenAlerter) getPairTickerInfo() (TradingPairTickerInfo, error) {
	var tpi TradingPairTickerInfo

	result, err := ka.KrakenAPI.Query("Ticker", map[string]string{
		"pair": "XETHXXBT",
	})

	if err != nil {
		fmt.Println("Error:", err)
		return tpi, fmt.Errorf("API Query failed: %s", err)
	}

	tmp1 := result.(map[string]interface{})["XETHXXBT"]
	tpi, err = unmarshalInterfaceToTradingPairTickerInfo(tmp1)
	if err != nil {
		fmt.Println("Error:", err)
		return tpi, fmt.Errorf("Failed to unmarshal API result: %s", err)
	}

	return tpi, nil
}

func (ka *KrakenAlerter) priceSMSAlert() {
	From := "+12033496456"
	To := ka.Phone

	// TODO: Add lower bound and say UP/DOWN
	msg := fmt.Sprintf("Last Price for %s: %f < %f < %f\n", "XETHXXBT", ka.LowerPriceBound, ka.LastPrice, ka.UpperPriceBound)
	_, _, err := ka.TwilioAPI.Messages.SendSMS(From, To, msg)
	if err != nil {
		fmt.Println("Failed to send SMS message:", err)
	}
}
