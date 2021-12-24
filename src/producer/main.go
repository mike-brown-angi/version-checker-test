package main

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
	"time"
)
const (
	topic         = "erboh-ratings"
	brokerAddress = "kafka-kafka-0.kafka-kafka-brokers.queuing.svc:9092"
)

func main() {
	var h http.Handler = http.HandlerFunc(notfound)
	r := mux.NewRouter()
	r.NotFoundHandler = h
	r.HandleFunc("/", home)
	r.HandleFunc("/health", health)
	err := http.ListenAndServe(":3000", r)
	if err != nil {
		return
	}
}

func closeProducer(w *kafka.Writer) {
	err := w.Close()
	if err != nil {
		panic(err)
	}
}
func produce(ctx context.Context, ratings []string) {

	l := log.New(os.Stdout, "kafka writer: ", 0)
	// initialize the writer with the broker addresses, and the topic
	w := kafka.NewWriter(kafka.WriterConfig{
		Brokers: []string{brokerAddress},
		Topic:   topic,
		Logger: l,
	})
    defer closeProducer(w)

	for i := 0; i < len(ratings); i++ {
		// each kafka message has a key and value. The key is used
		// to decide which partition (and consequently, which broker)
		// the message gets published on
		// The value in this case is the rating string.
		err := w.WriteMessages(ctx, kafka.Message{
			Key: []byte(strconv.Itoa(i)),
			Value: []byte(ratings[i]),
		})
		if err != nil {
			panic("could not write message " + err.Error())
		}
		// log a confirmation once the message is written
		fmt.Println("writes:", i)
		// sleep for a second
		time.Sleep(1 * time.Millisecond)
	}
}

func home(w http.ResponseWriter, r *http.Request) {
	// each time the page is refreshed on / the produce function fires again
	// adding ratings to the topic
	// create a new context to be used by the kafka writer
	ctx := context.Background()
	w.Header().Set("Content-Type", "text/html")
	h, ratings := ranking.GenRatings()
	_, err := fmt.Fprint(w, "<body><h1>Voice Simulation Ratings</h1><table>")
	if err != nil {
		return
	}
	_, err = fmt.Fprint(w, "<tr><th>Battle</th><th>Voice 1</th><th>Voice 2</th><th>Rating</th></tr>")
	if err != nil {
		return
	}
	for _, hl := range h {
		_, err2 := fmt.Fprintf(w, "%s\n", hl)
		if err2 != nil {
			return
		}
	}
	_, err = fmt.Fprint(w, "</table><br><br>")
	if err != nil {
		return
	}
	_, err = fmt.Fprint(w, "</body>")
	if err != nil {
		return
	}
	produce(ctx, ratings)
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := w.Write([]byte(fmt.Sprintf("[{\"response\": %d}]", http.StatusOK)))
	if err != nil {
		return
	}
}

func notfound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/html")
	_, err := fmt.Fprint(w, "404 Error: We couldn't find that.")
	if err != nil {
		return
	}
}
