package main

import (
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mdbdba/DigitalOceanChallenge2021/src/ranking"
	"net/http"
)

func main() {
	var h http.Handler = http.HandlerFunc(notfound)
	r := mux.NewRouter()
	r.NotFoundHandler = h
	r.HandleFunc("/", home)
	http.ListenAndServe(":3000", r)
}

func home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "<body><h1>Voice Simulation Ratings</h1><table>")
	fmt.Fprint(w, "<tr><th>Battle</th><th>Voice 1</th><th>Voice 2</th><th>Rating</th></tr>")
	for _, line := range ranking.GenRatings() {
		fmt.Fprintf(w, "%s\n", line)
	}
	fmt.Fprint(w, "</table></body>")
}

func notfound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, "404 Error: We couldn't find that.")
}
