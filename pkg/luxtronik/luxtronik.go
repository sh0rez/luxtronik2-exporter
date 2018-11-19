package luxtronik

import (
	"encoding/xml"
	"strings"
	"time"

	gowebsocket "github.com/sacOO7/GoWebsocket"
	log "github.com/sirupsen/logrus"
)

type Luxtronik struct {
	idRef  map[string]Location
	data   map[string]map[string]string
	socket gowebsocket.Socket
	c      chan string

	onUpdate func(new []Location)
}

func (l *Luxtronik) Value(domain, key string) string {
	return l.data[strings.ToLower(domain)][strings.ToLower(key)]
}

func (l *Luxtronik) Domains() map[string]map[string]string {
	return l.data
}

func (l *Luxtronik) update(new []item, filters Filters) {
	var domain, field string
	for _, updated := range new {
		// luxtronik returns values we did not requested (!?). Ignore those
		if _, ok := l.idRef[updated.ID]; !ok {
			continue
		}

		domain = l.idRef[updated.ID].domain
		field = l.idRef[updated.ID].field

		loc, val := filters.filter(domain, field, updated.Value)

		if l.data[loc.domain][loc.field] != val {
			l.data[loc.domain][loc.field] = val
			log.WithFields(log.Fields{
				"domain": loc.domain,
				"field":  loc.field,
				"value":  val,
				"id":     updated.ID,
			}).Debug("set value")
		}
	}
}

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
		var updatedVals category
		err := xml.Unmarshal([]byte(message), &updatedVals)
		if err != nil {
			panic(err)
		}
		lux.update(updatedVals.Items, filters)
	}

	go func() {
		for {
			lux.socket.SendText("REFRESH")
			time.Sleep(time.Second)
		}
	}()

	return &lux
}

func login(c *chan string, socket *gowebsocket.Socket) string {
	// supply invalid password here, read-only password is fine
	socket.SendText("LOGIN;0")
	response := <-*c

	var structure content
	err := xml.Unmarshal([]byte(response), &structure)
	if err != nil {
		panic(err)
	}

	return structure.Categories[0].ID
}
