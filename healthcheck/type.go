package healthcheck

import (
	"encoding/json"
	"fmt"
	"net"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

// Duration an alias for the duration type
type Duration time.Duration

// UnmarshalText unmarshal a duration
func (d *Duration) UnmarshalText(text []byte) error {
	if len(text) < 2 {
		return errors.New(fmt.Sprintf("%s is not a duration", text))
	}
	t := text[1 : len(text)-1]
	dur, err := time.ParseDuration(string(t))
	if err != nil {
		return errors.Wrapf(err, "%s is not a duration", text)
	}
	*d = Duration(dur)
	return nil
}

// UnmarshalYAML read a duration fom yaml
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw time.Duration
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read Cabourotte configuration")
	}
	*d = Duration(raw)
	return nil
}

// UnmarshalJSON marshal to json a duration
func (d *Duration) UnmarshalJSON(text []byte) error {
	return d.UnmarshalText(text)
}

// MarshalJSON marshal to json a duration
func (d Duration) MarshalJSON() ([]byte, error) {
	duration := time.Duration(d)
	return json.Marshal(duration.String())
}

// Protocol is the healthcheck http protocol
type Protocol int

const (
	// HTTP the HTTP protocol
	HTTP Protocol = 1 + iota
	// HTTPS the HTTPS protocol
	HTTPS
)

// UnmarshalYAML read a protocol fom yaml
func (p *Protocol) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw string
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read the healthcheck protocol")
	}
	if raw == "http" {
		*p = HTTP
	} else if raw == "https" {
		*p = HTTPS
	} else {
		return errors.New(fmt.Sprintf("Invalid protocol %s", raw))
	}
	return nil
}

// UnmarshalText unmarshal a duration
func (p *Protocol) UnmarshalText(text []byte) error {
	if len(text) < 2 {
		return errors.New(fmt.Sprintf("Invalid protocol %s", text))
	}
	t := text[1 : len(text)-1]
	s := string(t)
	if s == "http" {
		*p = HTTP
	} else if s == "https" {
		*p = HTTPS
	} else {
		return errors.New(fmt.Sprintf("Invalid protocol %s", s))
	}
	return nil
}

// UnmarshalJSON marshal to json a protocol
func (p *Protocol) UnmarshalJSON(text []byte) error {
	return p.UnmarshalText(text)
}

// MarshalJSON marshal to json a protocol
func (p Protocol) MarshalJSON() ([]byte, error) {
	if p == HTTP {
		return json.Marshal("http")
	} else if p == HTTPS {
		return json.Marshal("https")
	}
	return nil, errors.New(fmt.Sprintf("Unknown protocol %d", p))
}

// +kubebuilder:validation:type=string
// +kubebuilder:validation:Type=object
type Regexp regexp.Regexp

// UnmarshalText unmarshal a duration
func (r *Regexp) UnmarshalText(text []byte) error {
	s := string(text)
	reg, err := regexp.Compile(s)
	if err != nil {
		return errors.Wrapf(err, "Invalid regexp: %s", s)
	}
	*r = Regexp(*reg)
	return nil
}

// UnmarshalJSON unmarshal to json a Regexp
func (r Regexp) UnmarshalJSON(text []byte) error {
	return r.UnmarshalText(text)
}

// MarshalText marshals Regexp as string
func (r *Regexp) MarshalText() ([]byte, error) {
	if r != nil {
		reg := regexp.Regexp(*r)
		return []byte(reg.String()), nil
	}

	return nil, nil
}

// MarshalJSON marshal to json a Regexp
func (r *Regexp) MarshalJSON() ([]byte, error) {
	reg := regexp.Regexp(*r)
	s := reg.String()
	return json.Marshal(s)
}

// DeepCopyInto implementation
func (r *Regexp) DeepCopyInto(out *Regexp) {
	if r != nil {
		reg := regexp.Regexp(*r)
		s := reg.String()
		newReg, _ := regexp.Compile(s)
		*out = Regexp(*newReg)
	}
}

// DeepCopy implementation
func (r *Regexp) DeepCopy() *Regexp {
	if r == nil {
		return nil
	}
	out := new(Regexp)
	r.DeepCopyInto(out)
	return out
}

// IP an alias for the IP type
type IP net.IP

// UnmarshalText unmarshal an IP
func (i *IP) UnmarshalText(text []byte) error {
	s := string(text)
	ip := net.ParseIP(s)
	if ip == nil {
		return fmt.Errorf("Invalid IP %s", s)
	}
	*i = IP(ip)
	return nil
}

// UnmarshalJSON unmarshal to json an IP
func (i *IP) UnmarshalJSON(text []byte) error {
	return i.UnmarshalText(text)
}

// MarshalJSON marshal to json an IP
func (i *IP) MarshalJSON() ([]byte, error) {
	ip := net.IP(*i)
	return json.Marshal(ip.String())
}
