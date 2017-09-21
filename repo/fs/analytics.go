package fs_repo

import (
	"encoding/json"
	"github.com/ipfs/go-datastore/query"
	"github.com/qri-io/analytics"
	"time"
)

type Analytics struct {
	basepath
	batchSize int
	batch     []*analytics.Event
	// TODO - add timer that auto drains batch in a goroutine
}

func NewAnalytics(base string) Analytics {
	return Analytics{
		basepath:  basepath(base),
		batchSize: 10,
		batch:     []*analytics.Event{},
	}
}

func (a Analytics) Track(event string, props map[string]interface{}) error {
	e := &analytics.Event{
		Name:    event,
		Created: time.Now(),
		Props:   props,
	}

	if a.batch != nil {
		a.batch = append(a.batch, e)
		if len(a.batch) < a.batchSize {
			return nil
		}

		events, err := a.readAll()
		if err != nil {
			return err
		}
		events = append(events, a.batch...)
		err = a.saveFile(events, FileAnalytics)
		if err != nil {
			return err
		}
		a.batch = []*analytics.Event{}
		return nil
	}

	events, err := a.readAll()
	if err != nil {
		return err
	}
	events = append(events, e)
	return a.saveFile(events, FileAnalytics)
}

func (a Analytics) Query(q query.Query) (query.Results, error) {
	events, err := a.readAll()
	if err != nil {
		return nil, err
	}
	if a.batch != nil {
		events = append(a.batch, events...)
	}

	re := make([]query.Entry, 0, len(events))
	for _, e := range events {
		re = append(re, query.Entry{Key: e.Name, Value: e})
	}
	res := query.ResultsWithEntries(q, re)
	res = query.NaiveQueryApply(q, res)
	return res, nil
}

func (a Analytics) readAll() ([]*analytics.Event, error) {
	data, err := a.readBytes(FileAnalytics)
	if err != nil {
		return nil, err
	}
	events := []*analytics.Event{}
	err = json.Unmarshal(data, &events)
	return events, err
}
