package luxtronik

import (
	"bytes"
	"encoding/xml"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
	"github.com/gosimple/slug"
	log "github.com/sirupsen/logrus"
)

// Luxtronik XML types
// Those represent the way luxtronik returns the data. Only used for parsing.
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
	Items []item `xml:"item"` // new array for depth more than one
}

// Location is just a pair of Domain and Field and represents the location of data in our datastore
type Location struct {
	Domain, Field string
}

// Filters are needed, as the luxtronik does a very bad job at serializing it's data. Numeric data gets messed up with units, extra-chars, etc. They even differ by language.
// This is addressed by dynamic filters. The first filter that matches is used.
//
// Match.Value: Regular Expression (re2) matched against the original value
// Set.Key: text/template used as the new key, ignored if blank (MAY BE)
// Set.Value: text/template used as the new value, in json (MUST), not blank (MUST NOT)
type Filters []struct {
	Match struct {
		Value string `yaml:"value"`
	} `yaml:"match"`
	Set struct {
		Key   string `yaml:"key"`
		Value string `yaml:"value"`
	} `yaml:"set"`
}

// parseStructure converts the structure supplied by luxtronik into the internally used one.
//
// Luxtronik uses a flat key-value store. This is hard to use and requires processing on every query.
// Instead, we go with a two-dimensional map containing the data in the way it would be queried and maintain
// a mapping of luxtronik's ID's to the place where we put the data.
func parseStructure(response string, filters Filters) (data map[string]map[string]string, idRef map[string]Location) {
	var structure content
	err := xml.Unmarshal([]byte(response), &structure)
	if err != nil {
		panic(err)
	}
	// change nested structure to depth of one level
	structure = flattenNestedStructure(structure)

	// Stores the data sorted in Domain and Field
	data = make(map[string]map[string]string)

	// Maps luxtronik ID's to the actual Location in the data map. This represents luxtroniks way of storing the data.
	idRef = make(map[string]Location)

	for _, cat := range structure.Categories {
		data[slug.MakeLang(strings.ToLower(cat.Name), "de")] = make(map[string]string)
		for _, i := range cat.Items {
			loc, val := filters.filter(cat.Name, i.Name, i.Value)

			// Store the data Domain-Field based
			data[loc.Domain][loc.Field] = val

			// Store references where we put the data for easier updating
			log.WithFields(log.Fields{
				"domain": loc.Domain,
				"field":  loc.Field,
				"value":  val,
				"lux_id": i.ID,
			}).Debug("set value")
			idRef[i.ID] = loc
		}
	}
	return data, idRef
}

// flatten the nested structure to get only depth of one
func flattenNestedStructure(data content) content {
	for indc, cat := range data.Categories {
		var newItems []item
		for indi, i := range cat.Items {
			for _, j := range i.Items {
				// create new item
				var ni item
				ni.Name = i.Name + "_" + j.Name
				ni.Value = j.Value
				ni.ID = j.ID
				// append new item to parent item
				newItems = append(newItems, ni)
			}
			// if we had add new item(s), then delete the content of this entry
			if len(newItems) > 0 {
				data.Categories[indc].Items[indi].Name = ""
				data.Categories[indc].Items[indi].ID = ""
				data.Categories[indc].Items[indi].Items = nil
			}
		}
		// append new items to category
		data.Categories[indc].Items = append(cat.Items, newItems...)
	}
	return data
}

// filter applies the supplied filters to key and value and enforces the filter constraints. See Filters for reference.
func (filters Filters) filter(cat, field, value string) (Location, string) {
	loc := Location{
		Domain: slug.MakeLang(strings.ToLower(cat), "de"),
		Field:  slug.MakeLang(strings.ToLower(field), "de"),
	}

filterLoop:
	for _, f := range filters {
		// check if filter applies
		if regexp.MustCompile(f.Match.Value).MatchString(value) {
			var val, key bytes.Buffer

			// process new value
			err := template.Must(template.New("val").Funcs(sprig.TxtFuncMap()).Parse(f.Set.Value)).Execute(&val, value)
			if err != nil {
				panic(err)
			}

			// process new key
			err = template.Must(template.New("key").Funcs(sprig.TxtFuncMap()).Parse(f.Set.Key)).Execute(&key, loc.Field)
			if err != nil {
				panic(err)
			}

			value = strings.TrimSpace(val.String())
			// ignore blank key
			if strings.TrimSpace(key.String()) != "" {
				loc.Field = strings.TrimSpace(key.String())
			}

			// filter successful. avoid further filtering
			break filterLoop
		}
	}

	return loc, value
}
