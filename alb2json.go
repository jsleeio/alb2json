package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"runtime/pprof"
)

type field struct {
	key         string
	unformatter unformatterfunc
}

func newfield(k string, f unformatterfunc) field {
	return field{key: k, unformatter: f}
}

func ALBLogSpec() []field {
	return []field{
		newfield("type", unformatstring),
		newfield("timestamp", unformatstring),
		newfield("elb", unformatstring),
		newfield("client", unformathostport),
		newfield("target", unformathostport),
		newfield("request_processing_time_seconds", unformatfloat),
		newfield("target_processing_time_seconds", unformatfloat),
		newfield("response_processing_time", unformatfloat),
		newfield("elb_status_code", unformatint),
		newfield("target_status_code", unformatint),
		newfield("received_bytes", unformatint),
		newfield("sent_bytes", unformatint),
		newfield("request", unformatstring),
		newfield("user_agent", unformatstring),
		newfield("ssl_cipher", unformatstring),
		newfield("ssl_protocol", unformatstring),
		newfield("target_group_arn", unformatstring),
		newfield("trace_id", unformatstring),
		newfield("domain_name", unformatstring),
		newfield("chosen_cert_arn", unformatstring),
		newfield("matched_rule_priority", unformatint),
		newfield("request_creation_time", unformatstring),
		newfield("actions_executed", unformatcsv),
		newfield("redirect_url", unformatstring),
		newfield("error_reason", unformatstring),
		newfield("targets_all", unformathostportlist),
		newfield("target_status_codes_all", unformatstatuscodelist),
	}
}

type FieldEncoder struct {
	fields []field
}

func NewFieldEncoder(spec []field) FieldEncoder {
	return FieldEncoder{fields: spec}
}

func (fe FieldEncoder) EncodeTo(w io.Writer, v []string) error {
	// field definitions from AWS docs:
	// https://docs.aws.amazon.com/elasticloadbalancing/latest/application/load-balancer-access-logs.html
	m := make(map[string]interface{})
	for index, sval := range v {
		var key string
		var value interface{}
		var err error
		if index > len(fe.fields)-1 {
			key = fmt.Sprintf("unknown_%d", index)
			value, err = unformatstring(sval)
		} else {
			key = fe.fields[index].key
			if sval == "-" || sval == "" {
				value, err = nil, nil
			} else {
				value, err = fe.fields[index].unformatter(sval)
			}
		}
		if err != nil {
			return fmt.Errorf("error parsing field %v with value %v: %v", key, sval, err)
		}
		m[key] = value
	}
	enc, err := json.Marshal(m)
	if err != nil {
		return fmt.Errorf("unable to encode output line: %v", err)
	}
	_, err = w.Write(enc)
	if err != nil {
		return fmt.Errorf("unable to write complete output: %v", err)
	}
	fmt.Fprintln(w)
	return nil
}

func convertLogEntries(r io.Reader, w io.Writer, e FieldEncoder) error {
	fields := []string{}
	quoting := false
	escaping := false
	line := 0
	field := 0
	current := ""
	last := false
	rb := bufio.NewReader(r)
	for {
		c, size, err := rb.ReadRune()
		if err != nil {
			if size == 0 {
				c = '\n'
				last = true
			} else {
				fmt.Printf("read rune %q of length %d with error: %v", c, size, err)
				continue
			}
		}
		if escaping {
			escaping = false
			current += string(c)
			continue
		}
		switch c {
		case '\n':
			if quoting {
				return fmt.Errorf("line %v: EOL without closing quote", line)
			}
			if len(fields) > 0 || current != "" {
				fields = append(fields, current)
				err := e.EncodeTo(w, fields)
				if err != nil {
					return fmt.Errorf("line %v: encoding error: %v", err)
				}
			}
			if last == true {
				return nil
			}
			fields = []string{}
			line++
			field = 0
			current = ""
		case '\\':
			escaping = true
		case '"':
			if quoting {
				quoting = false
			} else {
				quoting = true
			}
		case ' ':
			if !quoting {
				fields = append(fields, current)
				current = ""
				field++
			} else {
				current += string(c)
			}
		default:
			current += string(c)
		}
	}
	return nil
}

func main() {
	profileOutput := flag.String("profile-output", "", "write CPU profile to file")
	flag.Parse()
	encoder := NewFieldEncoder(ALBLogSpec())
	if *profileOutput != "" {
		f, err := os.Create(*profileOutput)
		if err != nil {
			fmt.Printf("unable to create profiling file %q: %v\n", *profileOutput, err)
			os.Exit(1)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	if err := convertLogEntries(os.Stdin, os.Stdout, encoder); err != nil {
		fmt.Printf("error transcoding ALB logs to JSON: %v\n", err)
		os.Exit(1)
	}
}
