package main

import (
	"net"
	"strings"
	"strconv"
	"fmt"
)

type unformatterfunc func(string) (interface{}, error)

func unformatstring(s string) (interface{}, error) {
	return s, nil
}

func unformathostport(s string) (interface{}, error) {
	m := make(map[string]string)
	host, port, err := net.SplitHostPort(s)
	if err != nil {
		return nil, fmt.Errorf("error parsing host:port field", err)
	}
	m["host"] = host
	m["port"] = port
	return m, nil
}

func unformatfloat(s string) (interface{}, error) {
	if s == "" || s == "-" {
		return nil, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil, err
	}
	return f, nil
}

func unformatint(s string) (interface{}, error) {
	if s == "" || s == "-" {
		return nil, nil
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return nil, err
	}
	return i, nil
}

func unformatlist(s, delimiter, empty string, fn unformatterfunc) ([]interface{}, error) {
	if s == empty || s == "" {
		return []interface{}{}, nil
	}
	a := []interface{}{}
	for _, item := range strings.Split(s, delimiter) {
		v, err := fn(item)
		if err != nil {
			return nil, err
		}
		a = append(a, v)
	}
	return a, nil
}

func unformathostportlist(s string) (interface{}, error) {
	return unformatlist(s, " ", "-", unformathostport)
}

func unformatstatuscodelist(s string) (interface{}, error) {
	return unformatlist(s, " ", "-", unformatint)
}

func unformatcsv(s string) (interface{}, error) {
	return unformatlist(s, ",", "-", unformatstring)
}


