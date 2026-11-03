package main

import (
	"log"
	"net/http"

	rocketsapi "rockets-api"
)

func main() {
	store := rocketsapi.NewRocketStore()
	serv := rocketsapi.NewServer(store)

	log.Println("rockets API listening on :8088")
	log.Fatal(http.ListenAndServe(":8088", serv.Handler()))
}
