package luxtronik

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
)

// Luxtronik XML types
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

// Internal types
type location struct {
	domain, field string
}

type filter func(domain, key, value string) (string, string)

// parseStructure converts the structure supplied by luxtronik into the internally used one.
//
// Luxtronik uses a flat key-value store. This is hard to use and requires processing on every query.
// Instead, we go with a two-dimensional map containing the data in the way it would be queried and maintain
// a mapping of luxtronik's ID's to the place where we put the data.
func parseStructure(response string, filters []Filter) (data map[string]map[string]string, idRef map[string]location) {
	var structure content
	err := xml.Unmarshal([]byte(response), &structure)
	if err != nil {
		panic(err)
	}

	// Stores the data sorted in domain and field
	data = make(map[string]map[string]string)

	// Maps luxtronik ID's to the actual location in the data map. This represents luxtroniks way of storing the data.
	idRef = make(map[string]location)

	for _, cat := range structure.Categories {
		data[strings.ToLower(cat.Name)] = make(map[string]string)
		for _, i := range cat.Items {
			loc := location{
				domain: strings.ToLower(cat.Name),
				field:  strings.ToLower(i.Name),
			}

		filterLoop:
			for _, f := range filters {
				if regexp.MustCompile(f.Match.Value).MatchString(i.Value) {
					var tpl bytes.Buffer
					if err := template.Must(template.New("val").Funcs(sprig.TxtFuncMap()).Parse(f.Set.Value)).Execute(&tpl, i.Value); err != nil {
						panic(err)
					}
					i.Value = strings.TrimSpace(tpl.String())
					break filterLoop
				}
			}

			// Store the data domain-field based
			data[loc.domain][loc.field] = i.Value

			// Store references where we put the data for easier updating
			fmt.Println("Assign", i.ID, loc.domain, loc.field, i.Value)
			idRef[i.ID] = loc
		}
	}
	return data, idRef
}
