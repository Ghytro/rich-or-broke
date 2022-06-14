package openexchange

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/Ghytro/ab_interview/config"
)

func TestHistoricalRates(t *testing.T) {
	dates := [...]string{
		"2003-04-23",
		"2010-03-18",
		"2015-07-24",
		"2007-09-15",
		"2020-01-02",
	}
	timestamps := [len(dates)]time.Time{}
	for i, d := range dates {
		var err error
		timestamps[i], err = time.Parse("2006-01-02", d)
		if err != nil {
			t.Fatal(err)
		}
	}
	errs := make(chan error, len(dates)*2)
	correctRates, testedRates := [len(dates)]map[string]float64{}, [len(dates)]map[string]float64{}
	var wg sync.WaitGroup
	wg.Add(len(dates) * 2)
	for i, d := range dates {
		go func(idx int, date string) {
			defer wg.Done()
			resp, err := http.Get(
				fmt.Sprintf(
					"%shistorical/%s.json?app_id=%s&base=%s",
					config.Config.OpenExchangeBaseUrl,
					date,
					config.Config.OpenExchangeApiToken,
					config.Config.BaseCurrencyId,
				),
			)
			if err != nil {
				errs <- err
				return
			}
			switch resp.StatusCode {
			case http.StatusForbidden:
				errs <- ErrIncorrectBaseCurrency
				return
			case http.StatusBadRequest:
				errs <- ErrIncorrectDate
				return
			case http.StatusUnauthorized:
				errs <- ErrIncorrectOpenExchangeToken
				return
			}
			respBody, err := io.ReadAll(resp.Body)
			if err != nil {
				errs <- err
				return
			}
			resp.Body.Close()

			unmarshaled := new(
				struct {
					Rates map[string]interface{} `json:"rates"`
				},
			)
			if err := json.Unmarshal(respBody, unmarshaled); err != nil {
				errs <- err
				return
			}
			result := make(map[string]float64)
			for currency, rate := range unmarshaled.Rates {
				result[currency] = rate.(float64)
			}
			correctRates[idx] = result
		}(i, d)
		go func(idx int, t time.Time) {
			defer wg.Done()
			r, err := HistoricalRates(t)
			if err != nil {
				errs <- err
				return
			}
			testedRates[idx] = r
		}(i, timestamps[i])
	}
	wg.Wait()
	for i, trs := range testedRates {
		for base, tr := range trs {
			if cr, ok := correctRates[i][base]; !ok || cr != tr {
				t.Fatalf(
					"incorrect rate got for currency %s: expected %f, but got %f",
					base,
					cr,
					tr,
				)
			}
		}
	}
}
