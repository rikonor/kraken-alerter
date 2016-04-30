package krakenalerter

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type KrakenAlerterAPI struct {
	port          string
	krakenAlerter *KrakenAlerter
}

func NewKrakenAlerterAPI(port string, krakenAlerter *KrakenAlerter) *KrakenAlerterAPI {
	return &KrakenAlerterAPI{
		port:          port,
		krakenAlerter: krakenAlerter,
	}
}

func (kaa *KrakenAlerterAPI) StartKrakenAlerterAPI() {
	http.HandleFunc("/setUpperPriceBound", kaa.setUpperPriceBoundsAPI)
	http.HandleFunc("/setLowerPriceBound", kaa.setLowerPriceBoundsAPI)
	http.HandleFunc("/setAlertsEnabled", kaa.setAlertsEnabledAPI)
	http.ListenAndServe(":"+kaa.port, nil)
}

// HTTP Handlers
func (kaa *KrakenAlerterAPI) setUpperPriceBoundsAPI(w http.ResponseWriter, r *http.Request) {
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

	kaa.krakenAlerter.SetUpperPriceBound(newUpperBound)

	fmt.Fprint(w, "Success")
}

func (kaa *KrakenAlerterAPI) setLowerPriceBoundsAPI(w http.ResponseWriter, r *http.Request) {
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

	kaa.krakenAlerter.SetLowerPriceBound(newLowerBound)

	fmt.Fprint(w, "Success")
}

func (kaa *KrakenAlerterAPI) setAlertsEnabledAPI(w http.ResponseWriter, r *http.Request) {
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

	kaa.krakenAlerter.SetAlertsEnabled(newAlertsEnabled)

	fmt.Fprint(w, "Success")
}
