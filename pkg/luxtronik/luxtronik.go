package luxtronik

import (
	"encoding/xml"
	"strings"
	"sync"
	"time"

	gowebsocket "github.com/sacOO7/GoWebsocket"
)

type Luxtronik struct {
	data    map[string]string
	mapping map[string]map[string]string
	socket  gowebsocket.Socket
	c       chan string
}

func (l *Luxtronik) Value(domain, key string) string {
	return l.data[l.mapping[strings.ToLower(domain)][strings.ToLower(key)]]
}

func (l *Luxtronik) Domains() map[string]map[string]string {
	domains := make(map[string]map[string]string)

	for domain, domainMembers := range l.mapping {
		domains[domain] = make(map[string]string)
		for member, id := range domainMembers {
			domains[domain][member] = l.data[id]
		}
	}

	return domains
}

func (l *Luxtronik) update(new []item) {
	for _, updated := range new {
		l.data[updated.ID] = updated.Value
	}
}

func (l *Luxtronik) Refresh(wg *sync.WaitGroup) {
	for {
		l.socket.SendText("REFRESH")
		time.Sleep(time.Second)
	}
	wg.Done()
}

func Connect(ip string) *Luxtronik {
	var lux Luxtronik
	lux.socket = gowebsocket.New("ws://" + ip + ":8214")
	lux.socket.ConnectionOptions = gowebsocket.ConnectionOptions{
		Subprotocols: []string{"Lux_WS"},
	}
	lux.socket.Connect()

	lux.c = make(chan string)
	lux.socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		lux.c <- message
	}

	// Make first request to get the ID of the metrics
	id := login(&lux.c, &lux.socket)

	// Request the values, set the target
	lux.socket.SendText("GET;" + id)
	lux.mapping, lux.data = parseStructure(<-lux.c)

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
	socket.SendText("LOGIN;1654")
	response := <-*c

	var structure content
	err := xml.Unmarshal([]byte(response), &structure)
	if err != nil {
		panic(err)
	}

	return structure.Categories[0].ID
}
