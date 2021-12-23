package main

import "C"
import (
	"context"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mdbdba/DigitalOceanChallenge2021/src/ranking"
	"github.com/segmentio/kafka-go"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
)
const (
	topic         = "erboh-ratings"
	brokerAddress = "kafka-kafka-0.kafka-kafka-brokers.queuing.svc:9092"
)
var ratings []string

func main() {
	var h http.Handler = http.HandlerFunc(notfound)
	r := mux.NewRouter()
	r.NotFoundHandler = h
	r.HandleFunc("/", home)
	err := http.ListenAndServe(":3001", r)
	if err != nil {
		return
	}
}

func consume(ctx context.Context) {
	// create a new logger that outputs to stdout
	// and has the `kafka reader` prefix
	l := log.New(os.Stdout, "kafka reader: ", 0)
	// initialize a new reader with the brokers and topic
	// the groupID identifies the consumer and prevents
	// it from receiving duplicate messages
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{brokerAddress},
		Topic:   topic,
		GroupID: "voiceRatingConsumer",
		// assign the logger to the reader
		Logger: l,
	})
	for k := 0; k<20 ; k++ {
		// the `ReadMessage` method blocks until we receive the next event
		msg, err := r.ReadMessage(ctx)
		if err != nil {
			// panic("could not read message " + err.Error())
			break
		}
		l.Printf("%d %s\n", k, string(msg.Value))
		// after receiving the message, log its value
		ratings = append(ratings, string(msg.Value))
	}
	err := r.Close()
	if err != nil {
		return 
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	_, err := fmt.Fprint(w, "<body><h1>Voice Simulation Data</h1>")
	if err != nil {
		return
	}
	_, err = fmt.Fprint(w, "</body>")
	if err != nil {
		return
	}

	// create a new context
	ctx := context.Background()
	consume(ctx)
	orderedPairs := ranking.RankByValueCount(*getVoiceRatings())
	fmt.Fprintf(w, "<table><tr><th>Voice</th><th>Rating Sum</th></tr>")
	for o := 0; o < len(orderedPairs); o++ {
		fmt.Fprintf( w,"<tr><td>%s</td><td>%d</td><tr>\n",
			orderedPairs[o].Key, orderedPairs[o].Value)
	}
	fmt.Fprintf(w, "\n")
	return
}

func notfound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html")
	_, err := fmt.Fprint(w, "404 Error: We couldn't find that.")
	if err != nil {
		return
	}
}

func getVoiceRatings()  *map[string]int {
	vRatingHtml := map[string]int{}
	for _, v := range ranking.Voices {
		vRatingHtml[v] = 0
	}
	for _, rating := range ratings {
		ratingSlice := strings.Split(rating,"|")
		//         battle                   Voice 1            Voice 2     Rating
		// format: Steve Jobs vs Bill Gates|Christopher Walken|Darth Vader|2
		ratingValue, err := strconv.Atoi(ratingSlice[3])
		if err != nil {
			panic(err)
		}
		vRatingHtml[ratingSlice[1]] += ratingValue
		vRatingHtml[ratingSlice[2]] += ratingValue
	}
	return &vRatingHtml
}