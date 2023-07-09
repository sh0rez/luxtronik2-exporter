package luxtronik

import (
	"encoding/xml"
	"strings"
	"time"

	gowebsocket "github.com/sacOO7/GoWebsocket"
	log "github.com/sirupsen/logrus"
)

// Luxtronik represents a luxtronik2 heatpump (AlphaInnotec, Siemens, etc.) and contains the data returned by
// the appliance, but also the things required to interact with it (websocket, etc.).
type Luxtronik struct {
	idRef  map[string]Location
	data   map[string]map[string]string
	socket gowebsocket.Socket
	c      chan string

	// OnUpdate gets called as soon as new data got is received and stored.
	OnUpdate func(new []Location)
}

// Value returns the value of a domain.key
func (l *Luxtronik) Value(domain, key string) string {
	return l.data[strings.ToLower(domain)][strings.ToLower(key)]
}

// Domains return all available domains
func (l *Luxtronik) Domains() map[string]map[string]string {
	return l.data
}

// Connect initiates the connection to the heatpump, retrieves the initial data-set, stores it and launches the update-routine.
// The heatpump object is returned.
func Connect(ip string, filters Filters) *Luxtronik {
	log.WithField("ip", ip).Info("Connecting to heatpump")

	var lux Luxtronik
	lux.socket = gowebsocket.New("ws://" + ip + ":8214")
	lux.socket.ConnectionOptions = gowebsocket.ConnectionOptions{
		Subprotocols: []string{"Lux_WS"},
	}
	lux.socket.OnConnected = func(socket gowebsocket.Socket) {
		log.WithField("to", socket.Url).Info("Connection established successfully")
	}
	lux.socket.OnConnectError = func(err error, socket gowebsocket.Socket) {
		log.WithField("error", err).Error("connection failed")
	}

	// initiate connection
	lux.socket.Connect()

	lux.c = make(chan string)
	lux.socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		lux.c <- message
	}

	// Make first request to get the ID of the metrics
	id := login(&lux.c, &lux.socket)

	// Request the values, set the target, parse the response
	lux.socket.SendText("GET;" + id)
	lux.data, lux.idRef = parseStructure(<-lux.c, filters)

	// register update func
	lux.socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		var updatedVals content
		err := xml.Unmarshal([]byte(message), &updatedVals)
		if err != nil {
			panic(err)
		}
		// change nested structure to depth of one level
		updatedVals = flattenNestedStructure(updatedVals)

		items := []item{}
		for _, c := range updatedVals.Categories {
			items = append(items, c.Items...)
		}

		updated := lux.update(items, filters)
		lux.OnUpdate(updated)
	}

	go func() {
		for {
			lux.socket.SendText("REFRESH")
			time.Sleep(time.Second)
		}
	}()

	return &lux
}

// Login mimics the login behavior with an invalid password, which still grants read-only access. (!?)
func login(c *chan string, socket *gowebsocket.Socket) string {

	// supply invalid password here, read-only password is fine
	socket.SendText("LOGIN;0")

	// luxtronik returns data structure
	response := <-*c

	var structure content
	err := xml.Unmarshal([]byte(response), &structure)
	if err != nil {
		panic(err)
	}

	// return arbitrary hex-id where the metrics are located. This ID changes between requests. (!?)
	return structure.Categories[0].ID
}

// update handles subsequent (non-inital) received data
func (l *Luxtronik) update(new []item, filters Filters) []Location {
	var domain, field string
	var locs []Location
	for _, updated := range new {
		// luxtronik returns values we did not requested (!?). Ignore those
		if _, ok := l.idRef[updated.ID]; !ok {
			continue
		}

		domain = l.idRef[updated.ID].Domain
		field = l.idRef[updated.ID].Field

		// apply filters
		loc, val := filters.filter(domain, field, updated.Value)

		if l.data[loc.Domain][loc.Field] != val {
			l.data[loc.Domain][loc.Field] = val
			log.WithFields(log.Fields{
				"domain": loc.Domain,
				"field":  loc.Field,
				"value":  val,
				"id":     updated.ID,
			}).Debug("set value")
			locs = append(locs, loc)
		}
	}
	return locs
}
