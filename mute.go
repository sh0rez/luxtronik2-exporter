package main

import "regexp"

type Mute struct {
	domain, field *regexp.Regexp
}
type MuteList []Mute

func (mts MuteList) muted(domain, field string) bool {
	for _, m := range mts {
		if m.domain.Match([]byte(domain)) && m.field.Match([]byte(field)) {
			return true
		}
	}
	return false
}

var mutes MuteList
