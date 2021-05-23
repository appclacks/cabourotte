package riemanngo

import (
	"fmt"
	"os"
	"reflect"
	"sort"
	"time"

	pb "github.com/golang/protobuf/proto"
	"github.com/riemann/riemann-go-client/proto"
)

// EventToProtocolBuffer convert an event to a protobuf Event
func EventToProtocolBuffer(event *Event) (*proto.Event, error) {
	if event.Host == "" {
		event.Host, _ = os.Hostname()
	}

	if event.Time.IsZero() {
		event.Time = time.Now()
	}

	var e proto.Event

	e.Host = pb.String(event.Host)
	e.Time = pb.Int64(event.Time.Unix())
	e.TimeMicros = pb.Int64(event.Time.UnixNano() / int64(time.Microsecond))

	if event.Service != "" {
		e.Service = pb.String(event.Service)
	}

	if event.State != "" {
		e.State = pb.String(event.State)
	}

	if event.Description != "" {
		e.Description = pb.String(event.Description)
	}

	e.Tags = event.Tags

	e.Attributes = sortAttributes(
		event.Attributes,
	)

	if event.TTL != 0 {
		e.Ttl = pb.Float32(float32(event.TTL / time.Second))
	}

	if event.Metric != nil {
		switch reflect.TypeOf(event.Metric).Kind() {
		case reflect.Int, reflect.Int32, reflect.Int64:
			e.MetricSint64 = pb.Int64(reflect.ValueOf(event.Metric).Int())
		case reflect.Float32:
			e.MetricD = pb.Float64(reflect.ValueOf(event.Metric).Float())
		case reflect.Float64:
			e.MetricD = pb.Float64(reflect.ValueOf(event.Metric).Float())
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			e.MetricSint64 = pb.Int64(int64(reflect.ValueOf(event.Metric).Uint()))
		default:
			return nil, fmt.Errorf("Metric of invalid type (type %v)",
				reflect.TypeOf(event.Metric).Kind())
		}
	}

	return &e, nil
}

// ProtocolBuffersToEvents converts an array of proto.Event to an array of Event
func ProtocolBuffersToEvents(pbEvents []*proto.Event) []Event {
	var events []Event
	for _, event := range pbEvents {
		e := Event{
			State:       event.GetState(),
			Service:     event.GetService(),
			Host:        event.GetHost(),
			Description: event.GetDescription(),
			TTL:         time.Duration(event.GetTtl()) * time.Second,
			Tags:        event.GetTags(),
		}
		if event.TimeMicros != nil {
			e.Time = time.Unix(0, event.GetTimeMicros()*int64(time.Microsecond))
		} else if event.Time != nil {
			e.Time = time.Unix(event.GetTime(), 0)
		}
		if event.MetricF != nil {
			e.Metric = event.GetMetricF()
		} else if event.MetricD != nil {
			e.Metric = event.GetMetricD()
		} else {
			e.Metric = event.GetMetricSint64()
		}
		if event.Attributes != nil {
			e.Attributes = make(map[string]string, len(event.GetAttributes()))
			for _, attr := range event.GetAttributes() {
				e.Attributes[attr.GetKey()] = attr.GetValue()
			}
		}
		events = append(events, e)
	}
	return events
}

func sortAttributes(attributes map[string]string) []*proto.Attribute {
	keys := make(
		[]string, len(attributes),
	)

	i := 0

	for attr := range attributes {
		keys[i] = attr
		i++
	}

	sort.Strings(keys)

	// ---

	attrs := make(
		[]*proto.Attribute, len(keys),
	)

	for i, key := range keys {
		attrs[i] = &proto.Attribute{
			Key:   pb.String(key),
			Value: pb.String(attributes[key]),
		}
	}

	return attrs
}
