package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/rikonor/kraken-alerter"
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

var (
	phoneToAlert    = flag.String("phone", "", "Phone to alert using SMS")
	lowerPriceBound = flag.Float64("lower", 0.0, "Lower price bound")
	upperPriceBound = flag.Float64("upper", 10.0, "Upper price bound")
	enableAlerts    = flag.Bool("enable-alerts", false, "Enable alerts")

	// Twilio API Credentials
	twilioAccountSid = flag.String("twilio-sid", "", "Twilio Account SID")
	twilioAuthToken  = flag.String("twilio-token", "", "Twilio Authentication Token")
)

func main() {
	flag.Parse()

	// Phone is mandatory
	if *phoneToAlert == "" {
		fmt.Println("Please provide a phone number to alert")
		os.Exit(1)
	}

	fmt.Println("Starting Kraken-Alerter")

	// Create Twilio API
	twilioAPI := twilio.NewClient(*twilioAccountSid, *twilioAuthToken, nil)

	// Create Kraken Alerter
	krakenAlerter := krakenalerter.NewKrakenAlerter(twilioAPI, *phoneToAlert, *lowerPriceBound, *upperPriceBound, *enableAlerts)
	go krakenAlerter.StartKrakenAlerter()

	// Create Kraken Alerter API
	krakenAlerterAPI := krakenalerter.NewKrakenAlerterAPI("8080", krakenAlerter)
	go krakenAlerterAPI.StartKrakenAlerterAPI()

	fmt.Scanln()
}
