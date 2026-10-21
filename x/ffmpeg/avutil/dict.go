package avutil

import "strings"

// Dict is a string map that keeps insertion order, as AVDictionary does for listing.
type Dict struct {
	key   []string
	value []string
}

// Set stores key. An empty value deletes it.
func (d *Dict) Set(key, value string) {
	if d == nil {
		return
	}
	for i, have := range d.key {
		if strings.EqualFold(have, key) {
			if value == "" {
				d.key = append(d.key[:i], d.key[i+1:]...)
				d.value = append(d.value[:i], d.value[i+1:]...)
				return
			}
			d.value[i] = value
			return
		}
	}
	if value == "" {
		return
	}
	d.key = append(d.key, key)
	d.value = append(d.value, value)
}

// Get returns the value and whether key was present.
func (d *Dict) Get(key string) (string, bool) {
	if d == nil {
		return "", false
	}
	for i, have := range d.key {
		if strings.EqualFold(have, key) {
			return d.value[i], true
		}
	}
	return "", false
}

// Pairs lists keys in insertion order.
func (d *Dict) Pairs() [][2]string {
	if d == nil {
		return nil
	}
	out := make([][2]string, len(d.key))
	for i := range d.key {
		out[i] = [2]string{d.key[i], d.value[i]}
	}
	return out
}
