package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sh0rez/luxtronik2-exporter/pkg/luxtronik"
	log "github.com/sirupsen/logrus"
	yaml "gopkg.in/yaml.v2"
)

var gauges = make(map[string]*prometheus.GaugeVec)

func main() {
	// log.SetLevel(log.DebugLevel)
	filterSpec, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	var filters luxtronik.Filters
	yaml.Unmarshal([]byte(filterSpec), &filters)

	lux := luxtronik.Connect("172.21.20.103", filters)

	for name := range lux.Domains() {
		gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "luxtronik",
			Name:      name,
		},
			[]string{
				"attr",
			},
		)
		prometheus.MustRegister(gauge)
		gauges[name] = gauge
	}

	lux.OnUpdate = func(new []luxtronik.Location) {
		for _, loc := range new {
			domain := loc.Domain
			field := loc.Field
			value := lux.Value(domain, field)

			setMetric(domain, field, value)
		}
	}

	for domainName, domains := range lux.Domains() {
		for field, value := range domains {
			setMetric(domainName, field, value)
		}
	}

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}

type jsonMetric struct {
	Unit  string `json:"unit"`
	Value string `json:"value"`
}

func setMetric(domain, field, value string) {
	gauge := gauges[domain]

	var jv jsonMetric
	err := json.Unmarshal([]byte(value), &jv)

	v, err := strconv.ParseFloat(jv.Value, 64)
	if err != nil {
		log.WithFields(
			log.Fields{
				"domain": domain,
				"field":  field,
				"value":  value,
			}).Warn("metric value parse failure")
		return
	}

	id := field
	if jv.Unit != "" {
		id = id + "_" + jv.Unit
	}

	gauge.WithLabelValues(id).Set(v)
	log.WithFields(log.Fields{
		"id":    id,
		"value": v,
	}).Info("updated metric")
}
