package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/fatih/structs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sh0rez/luxtronik2-exporter/pkg/luxtronik"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var gauges = make(map[string]*prometheus.GaugeVec)

type Config struct {
	Address string `flag:"address" short:"a" help:"IP or hostname of the heatpump"`
	Filters luxtronik.Filters
}

func main() {
	config := getConfig()

	lux := luxtronik.Connect(config.Address, config.Filters)

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

func getConfig() *Config {

	viper.SetConfigName("lux")
	viper.AddConfigPath(".")

	for _, s := range structs.Fields(Config{}) {
		if s.Tag("flag") != "" {
			pflag.StringP(s.Tag("flag"), s.Tag("short"), s.Tag("default"), s.Tag("help"))
		}
	}
	viper.BindPFlags(pflag.CommandLine)
	pflag.Parse()

	viper.SetEnvPrefix("lux")
	viper.AutomaticEnv()

	var config Config
	if err := viper.ReadInConfig(); err != nil {
		log.WithField("err", err).Fatal("Error getting config from sources")
	}
	if err := viper.Unmarshal(&config); err != nil {
		log.Error(err)
		log.WithField("err", err).Fatal("invalid config")
	}
	return &config
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
