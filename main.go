package main

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sh0rez/luxtronik2-exporter/pkg/luxtronik"
)

var (
	use = []string{"temperaturen", "eingänge", "ausgänge", "wärmemenge", "glt"}
)

func main() {
	lux := luxtronik.Connect("172.21.20.103")

	var wg sync.WaitGroup
	wg.Add(1)
	go lux.Refresh(&wg)

	in := func(s string) bool {
		for _, u := range use {
			if s == u {
				return true
			}
		}
		return false
	}

	for name, domain := range lux.Domains() {
		if !in(name) {
			continue
		}

		name = strings.Replace(name, "ä", "ae", -1)
		gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: "luxtronik",
			Name:      name,
		},
			[]string{
				"attr",
			},
		)
		prometheus.MustRegister(gauge)

		var (
			v   float64
			err error
			id  string
		)
		for field, value := range domain {

			if strings.Contains(value, "Ein") || strings.Contains(value, "Aus") {
				v = 1
				if value == "Aus" {
					v = 0
				}
				id = field
			} else {
				var split string
				if strings.Contains(value, " ") {
					split = " "
				} else if strings.Contains(value, "°") {
					split = "°"
				}
				val := strings.Split(value, split)

				v, err = strconv.ParseFloat(val[0], 64)
				if err != nil {
					continue
				}
				id = field + "_" + val[1]
			}

			fmt.Println(id)
			gauge.WithLabelValues(id).Set(v)
		}
	}

	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
