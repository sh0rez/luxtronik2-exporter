package luxtronik

import (
	"encoding/xml"
	"strings"
	"sync"
	"time"

	gowebsocket "github.com/sacOO7/GoWebsocket"
)

type Content struct {
	ID         string     `xml:"id,attr"`
	Categories []Category `xml:"item"`
}

type Category struct {
	ID    string `xml:"id,attr"`
	Name  string `xml:"name"`
	Items []Item `xml:"item"`
}

type Item struct {
	ID    string `xml:"id,attr"`
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

func Connect(ip string, passwd string) *Luxtronik {
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
	id := login(passwd, &lux.c, &lux.socket)

	// Request the values, set the target
	lux.socket.SendText("GET;" + id)
	lux.mapping, lux.data = parseStructure(<-lux.c)

	lux.socket.OnTextMessage = func(message string, socket gowebsocket.Socket) {
		var updatedVals Category
		err := xml.Unmarshal([]byte(message), &updatedVals)
		if err != nil {
			panic(err)
		}
		lux.update(updatedVals.Items)
	}

	return &lux
}

func parseStructure(response string) (map[string]map[string]string, map[string]string) {
	var structure Content
	err := xml.Unmarshal([]byte(response), &structure)
	if err != nil {
		panic(err)
	}

	mapping := make(map[string]map[string]string)
	data := make(map[string]string)

	for _, cat := range structure.Categories {
		mapping[strings.ToLower(cat.Name)] = make(map[string]string)
		for _, i := range cat.Items {
			mapping[strings.ToLower(cat.Name)][strings.ToLower(i.Name)] = i.ID
			data[i.ID] = i.Value
		}
	}

	return mapping, data
}

func login(passwd string, c *chan string, socket *gowebsocket.Socket) string {
	socket.SendText("LOGIN;" + passwd)
	response := <-*c

	var structure Content
	err := xml.Unmarshal([]byte(response), &structure)
	if err != nil {
		panic(err)
	}

	return structure.Categories[0].ID
}

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

func (l *Luxtronik) update(new []Item) {
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
