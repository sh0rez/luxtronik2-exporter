package luxtronik

import (
	"encoding/xml"
	"strings"
)

type content struct {
	ID         string     `xml:"id,attr"`
	Categories []category `xml:"item"`
}

type category struct {
	ID    string `xml:"id,attr"`
	Name  string `xml:"name"`
	Items []item `xml:"item"`
}

type item struct {
	ID    string `xml:"id,attr"`
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

func parseStructure(response string) (map[string]map[string]string, map[string]string) {
	var structure content
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
