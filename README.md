<p align="center">
    <h1 align="center">luxtronik2-exporter</h1>
    <p align="center">
    Prometheus exporter for luxtronik2-based heatpumps.
    </p>
    <p align="center">
        See <a href="#features">Features</a> section for an in-depth summary
        about the features of <code>luxtronik2-exporter</code>.
    </p>
</p>
<p align="center">
    <a href="https://golang.org/">
        <img src="https://img.shields.io/badge/language-go-blue.svg" alt="Golang">
    </a>
    <a href="https://github.com/golang/dep">
        <img src="https://img.shields.io/badge/vendoring-dep-red.svg" alt="Dep">
    </a>
    <a href="https://github.com/sh0rez/neg/releases">
        <img src="https://img.shields.io/github/tag/sh0rez/luxtronik2-exporter.svg" alt="Release">
    </a>
    <a href="LICENSE">
        <img src="https://img.shields.io/github/license/sh0rez/luxtronik2-exporter.svg" alt="License Apache2">
    </a>
</p>

## Motivation
Multiple german companies selling heatpumps equip them with a smart-module internally called [`luxtronik2`](https://www.alpha-innotec.de/endkunde/produkte/waermepumpen-produktkatalog/alterra-serie/flexibles-bedienkonzept-weltweite-steuerung.html). It is fabricated by the german company [AlphaInnotec](https://www.alpha-innotec.de/endkunde/home.html) and put on heatpumps of the former, Bosch, Siemens, et al. The system includes a webserver showing some stats about the heatpump in a crude web-interface. But this has several problems:
* it implements a custom polling protocol on top of `WebSockets` (wtf)
* no encryption, so we have unprotected authentication (goodbye password), which is a **severe security problem**, even in local networks
* the data is not tracked over time, so it is fairly useless.

This application primarily addresses the last point, but allows solving the former ones along the way.

## Features
The main feature of this exporter is to **expose all stats as Prometheus `Counters`**. It dynamically fetches the data from the heatpump and exposes only what it gets.

#### Furthermore:
* **Filters**: Luxtronik is horrible and returns language-specific names for all stats and includes units in the value. I do not appreciate this, neither does Prometheus. So I came up if a mutation-engine, to modify the inflight data, before it meets other parts of the app:
* **Read-Only**: This exporter runs fine in read-only mode, so no authentication-configuration is required for accessing `luxtronik2`. So no secret needs to be send over the line at any time.
* **12-factor**: This app follows the 12-factor guidelines and everything this imposes, e.g. configuration using files, the environment, etc.
* **Prometheus Protocol**: The rewritten metrics are exposed in a standardized and easily parsable way in the native Prometheus format, ready to be scraped. The format is suited way better for the purpose than the self-baked WebSocket-polling mechanism luxtronik uses.

## Filters
Like already mentioned, Luxtronik exposes it's metrics in a very weird way, including units and language-specific tings in the metric values. To address this, the following seemed reasonable:

To mutate the data, filters are used. A filter is a pair of regex (for matching) and a GoLang `text/template` for mutating the result. Consider the following example:

```yaml
  - match:
      value: Ein|Aus
    set:
      value: '{{$v := "-1"}} {{if (eq . "Ein")}} {{$v = "1"}} {{else}} {{$v = "0"}} {{end}} {{$d := dict "value" $v}} {{toJson $d}}'
      key: '{{if regexMatch "_state" . }} {{.}} {{else}} {{.}}_state {{end}}'
```

Whenever a message comes in, the regular expression defined in match is evaluated, in this case on the value (The key is the name of the metric, the value the actual data). In this case, a value of either `Ein` or `Aus` will trigger this filter.  
If the regex matches, the templates defined in set are evaluated.
* `value`: Allows overriding the value of the metric. In this case, if the value equals `Ein` it becomes `1`, or else if it equals `Aus` it will be `0` (Representation of a `bool` in Prometheus). In case it is neither `Ein` nor `Aus` (which cannot happen, but still), the value becomes the invalid `-1`.
* `key`: The same as value, allows overriding, but this time the key. In this case, the literal `_state` is appended to the key to note it now holds a boolean.

The configuration file `lux.yml` already contains a default set of filters, suitable for the default deployment in Germany **(!)**, it might need adaptions for other locales. PR's are welcome.

## Configuration
**Consider `lux.yml` for the latest configuration options!**

Example:
```yaml
address: 192.168.1.100
verbose: false
mutes:
  - domain: "ablaufzeiten"
    field: "wp-seit|ssp-zeit"
filters:
  # Make bools to bools
  - match:
      value: Ein|Aus
    set:
      value: '{{$v := "-1"}} {{if (eq . "Ein")}} {{$v = "1"}} {{else}} {{$v = "0"}} {{end}} {{$d := di
ct "value" $v}} {{toJson $d}}'
      key: '{{if regexMatch "_state" . }} {{.}} {{else}} {{.}}_state {{end}}'
```

| Key       | Purpose                                                      | Example                                            |
|-----------|--------------------------------------------------------------|----------------------------------------------------|
| `address` | IP or hostname of the heatpump                               | `192.168.1.100`                                    |
| `verbose` | Debug logging                                                | `true` / `false`                                   |
| `mutes`   | Array of metrics to be excluded from logs (reduce verbosity) | `[{"domain": "ablaufzeiten", "field": "wp-seit"}]` |
| `filters` | Array of filters                                             | See above                                          |

## Security
Using this piece of software, it is quite simple to make `luxtronik2` secure:

1. Prevent access to the "real" luxtronik for all but a single host
    1. using a firewall (pfSense, iptables, etc.)
    2. using any single-purpose computer (e.g. Raspberry Pi): Connect the heatpump directly to the Ethernet port of the Pi and set up static IP's on both devices. The heatpump is only available on the Pi now.
2. Run this exporter on the single host that has access
3. Voila! No secrets to be leaked, only a single HTTP endpoint with the metrics, ready to be scraped!
4. *Optional:* If needed, a reverse proxy (e.g. `nginx`) could be used to setup authentication in a secure manner and especially encryption using `TLS`-termination

This setup is secure by design, as it has a nearly no attack vector when used with `TLS`. Furthermore, it only uses widely trusted, standard protocols (`HTTP(s)`).
