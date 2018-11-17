package luxtronik

import (
	"encoding/xml"
	"fmt"
	"strings"
	"sync"
	"time"

	gowebsocket "github.com/sacOO7/GoWebsocket"
)

type Filter struct {
	Match struct {
		Key   string `yaml:"key"`
		Value string `yaml:"value"`
	} `yaml:"match"`
	Set struct {
		Key   string `yaml:"key"`
		Value string `yaml:"value"`
	} `yaml:"set"`
}

type Luxtronik struct {
	idRef  map[string]location
	data   map[string]map[string]string
	socket gowebsocket.Socket
	c      chan string
}

func (l *Luxtronik) Value(domain, key string) string {
	return l.data[strings.ToLower(domain)][strings.ToLower(key)]
}

func (l *Luxtronik) Domains() map[string]map[string]string {
	return l.data
}

func (l *Luxtronik) update(new []item) {
	var domain, field string
	for _, updated := range new {
		// luxtronik returns values we did not requested (!?). Ignore those
		if _, ok := l.idRef[updated.ID]; !ok {
			continue
		}

		domain = l.idRef[updated.ID].domain
		field = l.idRef[updated.ID].field

		if l.data[domain][field] != updated.Value {
			l.data[domain][field] = updated.Value
			fmt.Println("Updated", updated.ID, domain, field, updated.Value)
		}
	}
}

func (l *Luxtronik) Refresh(wg *sync.WaitGroup) {
	for {
		l.socket.SendText("REFRESH")
		time.Sleep(time.Second)
	}
	wg.Done()
}

func Connect(ip string, filters []Filter) *Luxtronik {
	var lux Luxtronik
	lux.socket = gowebsocket.New("ws://" + ip + ":8214")
	lux.socket.ConnectionOptions = gowebsocket.ConnectionOptions{
		Subprotocols: []string{"Lux_WS"},
	}
	lux.socket.Connect()

	lux.c = make(chan string)
	lux.socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		lux.c <- message
		fmt.Println("Connected.")
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
		lux.update(updatedVals.Items)
	}

	return &lux
}

func login(c *chan string, socket *gowebsocket.Socket) string {
	// supply invalid password here, read-only password is fine
	socket.SendText("LOGIN;215318")
	response := <-*c

	var structure content
	err := xml.Unmarshal([]byte(response), &structure)
	if err != nil {
		panic(err)
	}

	return structure.Categories[0].ID
}
