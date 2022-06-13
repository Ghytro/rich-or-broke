package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Ghytro/ab_interview/config"
	"github.com/Ghytro/ab_interview/handler"
	"github.com/gorilla/mux"
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/api/diff/{currency_id}", handler.DiffHandler).Methods("GET")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", config.Config.Port), router))
}
