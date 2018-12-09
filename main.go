package main

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/fatih/structs"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sh0rez/luxtronik2-exporter/pkg/luxtronik"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// metrics
var gauges = make(map[string]*prometheus.GaugeVec)

// Config holds the configuration structure
type Config struct {
	Verbose bool   `flag:"verbose" short:"v", help:"Show debug logs"`
	Address string `flag:"address" short:"a" help:"IP or hostname of the heatpump"`
	Filters luxtronik.Filters
	Mutes    []struct {
		Domain string
		Field  string
	}
}

type Mute struct {
	domain, field *regexp.Regexp
}
type MuteList []Mute

func (mts MuteList) muted(domain, field string) bool {
	for _, m := range mts {
		if m.domain.Match([]byte(domain)) && m.field.Match([]byte(field)) {
			return true
		}
	}
	return false
}

var mutes MuteList

func main() {
	// get config from viper
	config := getConfig()

	log.SetLevel(log.InfoLevel)
	if config.Verbose {
		log.SetLevel(log.DebugLevel)
	}

	mutes = make(MuteList, len(config.Mutes))
	for i, m := range config.Mutes {
		mutes[i] = Mute{
			domain: regexp.MustCompile(m.Domain),
			field:  regexp.MustCompile(m.Field),
		}
	}

	// connect to the heatpump
	lux := luxtronik.Connect(config.Address, config.Filters)

	// create gauge metric for each domain
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

	// register update handler, gets called by the update routine
	// updates changed metrics
	lux.OnUpdate = func(new []luxtronik.Location) {
		for _, loc := range new {
			domain := loc.Domain
			field := loc.Field
			value := lux.Value(domain, field)

			setMetric(domain, field, value)
		}
	}

	// expose all known values as metric
	for domainName, domains := range lux.Domains() {
		for field, value := range domains {
			setMetric(domainName, field, value)
		}
	}

	// serve the /metrics endpoint
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}

// getConfig returns the configuration from flag, environment variable and file, prioritize in that order.
func getConfig() *Config {

	// file config
	viper.SetConfigName("lux")
	viper.AddConfigPath(".")

	// flag config
	for _, s := range structs.Fields(Config{}) {
		if s.Tag("flag") != "" {
			pflag.StringP(s.Tag("flag"), s.Tag("short"), s.Tag("default"), s.Tag("help"))
		}
	}
	viper.BindPFlags(pflag.CommandLine)
	pflag.Parse()

	// env config
	viper.SetEnvPrefix("lux")
	viper.AutomaticEnv()

	// unmarshal sources
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

// jsonMetric represents the json-representation of a metric, created by the filter rules
type jsonMetric struct {
	Unit  string `json:"unit"`
	Value string `json:"value"`
}

// setMetric updates sets the gauge of a metric to a value
func setMetric(domain, field, value string) {
	gauge := gauges[domain]

	var jv jsonMetric
	err := json.Unmarshal([]byte(value), &jv)

	v, err := strconv.ParseFloat(jv.Value, 64)
	if err != nil {
		if !mutes.muted(domain, field) {
			log.WithFields(
				log.Fields{
					"domain": domain,
					"field":  field,
					"value":  value,
				}).Warn("metric value parse failure")
		}
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
	}).Debug("updated metric")
}
