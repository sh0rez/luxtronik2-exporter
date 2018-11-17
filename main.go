package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"sync"

	"github.com/sh0rez/luxtronik2-exporter/pkg/luxtronik"
	yaml "gopkg.in/yaml.v2"
)

func main() {
	filterSpec, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}

	var filters []luxtronik.Filter
	yaml.Unmarshal([]byte(filterSpec), &filters)

	// fmt.Println(filters)

	lux := luxtronik.Connect("172.21.20.103", filters)
	var wg sync.WaitGroup
	wg.Add(1)
	go lux.Refresh(&wg)

	fmt.Println(lux.Value("eingänge", "evu"))

	// in := func(s string) bool {
	// 	for _, u := range use {
	// 		if s == u {
	// 			return true
	// 		}
	// 	}
	// 	return false
	// }

	// for name, domain := range lux.Domains() {
	// 	if !in(name) {
	// 		continue
	// 	}

	// 	name = strings.Replace(name, "ä", "ae", -1)
	// 	gauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
	// 		Namespace: "luxtronik",
	// 		Name:      name,
	// 	},
	// 		[]string{
	// 			"attr",
	// 		},
	// 	)
	// 	prometheus.MustRegister(gauge)
	// }

	// go func() {
	// 	http.Handle("/metrics", promhttp.Handler())
	// 	http.ListenAndServe(":2112", nil)
	// }()

	// for {
	// 	for name, domain := range lux.Domains() {

	// 		var (
	// 			v   float64
	// 			err error
	// 			id  string
	// 		)
	// 		for field, value := range domain {
	// 			// Convert "Ein"/"Aus to bool"
	// 			if strings.Contains(value, "Ein") || strings.Contains(value, "Aus") {
	// 				v = 1
	// 				if value == "Aus" {
	// 					v = 0
	// 				}
	// 				id = field
	// 			} else {
	// 				// numeric value filter
	// 				// Splits according to separator. Usually whitespace, but degree for temperatures.
	// 				var split string
	// 				if strings.Contains(value, " ") {
	// 					split = " "
	// 				} else if strings.Contains(value, "°") {
	// 					split = "°"
	// 				}
	// 				val := strings.Split(value, split)

	// 				v, err = strconv.ParseFloat(val[0], 64)
	// 				if err != nil {
	// 					continue
	// 				}
	// 				id = field + "_" + val[1]
	// 			}

	// 			fmt.Println(id)
	// 			gauge.WithLabelValues(id).Set(v)
	// 		}
	// 	}
	// 	time.Sleep(time.Second)
	// }

}
