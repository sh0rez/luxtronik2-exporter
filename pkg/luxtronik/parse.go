package luxtronik

import (
	"encoding/xml"
	"fmt"
	"strings"
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
//
// A problem arises as luxtronik tends to return multiple with the same domain-field data.
// To mitigate this, the filter functions dynamically rewrite those specific cases. They provide a flexible,
// extendable and are easy to work with.
// Filters are processed from top to bottom, the first matching filter will be used. A map is used, as slices have
// no fixed order.
func parseStructure(response string, filters map[int]filter) (data map[string]map[string]string, idRef map[string]location) {
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

			for _, filter := range filters {
				fDomain, fField := filter(loc.domain, loc.field, i.Value)
				if fDomain != "" && fField != "" {
					loc.domain = strings.ToLower(fDomain)
					loc.field = strings.ToLower(fField)
				}
			}

			// Store the data domain-field based
			data[loc.domain][loc.field] = i.Value

			// Store references where we put the data for easier updating
			fmt.Println("Assign", i.ID, loc.domain, loc.field)
			idRef[i.ID] = loc
		}

	}

	return data, idRef
}
