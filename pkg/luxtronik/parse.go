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
type Location struct {
	domain, field string
}

type Filters []struct {
	Match struct {
		Key   string `yaml:"key"`
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

	// Stores the data sorted in domain and field
	data = make(map[string]map[string]string)

	// Maps luxtronik ID's to the actual Location in the data map. This represents luxtroniks way of storing the data.
	idRef = make(map[string]Location)

	for _, cat := range structure.Categories {
		data[slug.MakeLang(strings.ToLower(cat.Name), "de")] = make(map[string]string)
		for _, i := range cat.Items {
			loc, val := filters.filter(cat.Name, i.Name, i.Value)

			// Store the data domain-field based
			data[loc.domain][loc.field] = val

			// Store references where we put the data for easier updating
			log.WithFields(log.Fields{
				"domain": loc.domain,
				"field":  loc.field,
				"value":  val,
				"lux_id": i.ID,
			}).Debug("set value")
			idRef[i.ID] = loc
		}
	}
	return data, idRef
}

func (filters Filters) filter(cat, field, value string) (Location, string) {
	loc := Location{
		domain: slug.MakeLang(strings.ToLower(cat), "de"),
		field:  slug.MakeLang(strings.ToLower(field), "de"),
	}

filterLoop:
	for _, f := range filters {
		if regexp.MustCompile(f.Match.Value).MatchString(value) {
			var val, key bytes.Buffer
			err := template.Must(template.New("val").Funcs(sprig.TxtFuncMap()).Parse(f.Set.Value)).Execute(&val, value)
			if err != nil {
				panic(err)
			}

			err = template.Must(template.New("key").Funcs(sprig.TxtFuncMap()).Parse(f.Set.Key)).Execute(&key, loc.field)
			if err != nil {
				panic(err)
			}

			value = strings.TrimSpace(val.String())
			if strings.TrimSpace(key.String()) != "" {
				loc.field = strings.TrimSpace(key.String())
			}
			break filterLoop
		}
	}

	return loc, value
}
