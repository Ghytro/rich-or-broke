package handler

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
	"time"

	"github.com/Ghytro/ab_interview/openexchange"
	"github.com/Ghytro/ab_interview/tenor"

	"github.com/gorilla/mux"
)

var errIncorrectCurrencyCode = errors.New("incorrect currency code")

func init() {
	rand.Seed(time.Now().UnixNano())
}

func DiffHandler(w http.ResponseWriter, r *http.Request) {
	today := time.Now()
	yesterday := today.Add(-24 * time.Hour)
	chanYesterdayCourse := make(chan float64)
	chanTodayCourse := make(chan float64)
	chanError := make(chan error)
	currency := mux.Vars(r)["currency_id"]
	f := func(t time.Time, c chan float64) {
		m, err := openexchange.HistoricalRates(t, "USD")
		if err != nil {
			log.Println(err)
			chanError <- err
			return
		}
		val, ok := m[currency]
		if !ok {
			log.Println(errIncorrectCurrencyCode)
			chanError <- errIncorrectCurrencyCode
			return
		}
		chanError <- nil
		c <- val
	}
	go f(today, chanTodayCourse)
	go f(yesterday, chanYesterdayCourse)
	for i := 0; i < 2; i++ {
		err := <-chanError
		if err != nil {
			if err == errIncorrectCurrencyCode {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			return
		}
	}
	todayCourse, yesterdayCourse := <-chanTodayCourse, <-chanYesterdayCourse

	var (
		gif *tenor.Gif
		err error
	)
	if todayCourse > yesterdayCourse {
		gif, err = tenor.GetRandomGif("rich", 100)
	} else {
		gif, err = tenor.GetRandomGif("broke", 100)
	}
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(gif.BinaryContent)
}