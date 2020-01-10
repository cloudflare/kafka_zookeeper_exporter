package main

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	zkLog "log"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	timeout = 10 * time.Second
	addr = "0.0.0.0:6900"
)

func healthCheck(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func main() {
	log.SetFormatter(&log.JSONFormatter{
		FieldMap: log.FieldMap{
			log.FieldKeyMsg: "message",
		},
	})
	broker := os.Getenv("ZK_BROKER")
	chroot := os.Getenv("ZK_CHROOT")
	if len(broker) == 0 {
		log.Fatal("Required list of ZooKeeper brokers not available.")
	} else if len(chroot) == 0 {
		log.Fatal("Required chroot for ZooKeeper is not specified.")
	}
	zkLog.SetOutput(ioutil.Discard)

	prometheus.MustRegister(
		newCollector(strings.Split(broker, ","), chroot, timeout))
	s := &http.Server{
		Addr:         addr,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	}
	p := mux.NewRouter()
	p.Handle("/metrics",
		promhttp.Handler()).Methods("GET")
	p.Handle("/health",
		http.HandlerFunc(healthCheck)).Methods("GET")
	s.Handler = p
	log.Fatal(s.ListenAndServe())
}