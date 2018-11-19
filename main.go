package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sh0rez/luxtronik2-exporter/pkg/luxtronik"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	filterSpec, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	var filters luxtronik.Filters
	yaml.Unmarshal([]byte(filterSpec), &filters)

	lux := luxtronik.Connect("172.21.20.103", filters)

	type jsonMetric struct {
		Unit  string `json:"unit"`
		Value string `json:"value"`
	}

	var gauges = make(map[string]*prometheus.GaugeVec)
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

	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	for {
		for name, domain := range lux.Domains() {
			gauge := gauges[name]
			for field, value := range domain {
				var jv jsonMetric
				err := json.Unmarshal([]byte(value), &jv)

				v, err := strconv.ParseFloat(jv.Value, 64)
				if err != nil {
					fmt.Println("Error: failed to parse", name, field, jv.Value)
					continue
				}

				id := field
				if jv.Unit != "" {
					id = id + "_" + jv.Unit
				}

				gauge.WithLabelValues(id).Set(v)
				fmt.Println("registered", id, v)
			}
		}
		time.Sleep(time.Second)
	}

}
