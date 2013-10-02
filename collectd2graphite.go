package main

/*
 * go run collectd2graphite.go
 * or compile it using `go build`
 *
 */

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

type CollectdEntry struct {
	Time           int64     `json:"time"`
	Interval       uint64    `json:"interval"`
	Host           string    `json:"host"`
	Plugin         string    `json:"plugin"`
	PluginInstance string    `json:"plugin_instance"`
	Type           string    `json:"type"`
	TypeInstance   string    `json:"type_instance"`
	Values         []float64 `json:"values"`
	DSTypes        []string  `json:"dstypes"`
	DSNames        []string  `json:"dsnames"`
}

type GraphiteEntry struct {
	Metric    string
	Value     float64
	Timestamp int64
}

type Graphite struct {
	endpoint       string
	conn           net.Conn
	ch             chan GraphiteEntry
	connectTimeout time.Duration
}

func NewGraphite(endpoint string) *Graphite {
	g := &Graphite{
		endpoint:       endpoint,
		ch:             make(chan GraphiteEntry),
		connectTimeout: 2 * time.Second}
	g.getConnection()
	go g.sender()
	return g
}

func graphiteFriendly(s string) string {
	s = strings.Replace(s, ".", "_", -1)
	s = strings.Replace(s, " ", "_", -1)
	return s
}

func (g *Graphite) getConnection() {
	var (
		err  error
		wait = 10 * time.Second
	)
	g.conn, err = net.DialTimeout("tcp", g.endpoint, g.connectTimeout)
	for err != nil {
		g.conn, err = net.DialTimeout("tcp", g.endpoint, g.connectTimeout)
		time.Sleep(wait)
	}
}

func (g *Graphite) Close() {
	g.conn.Close()
}

func (g *Graphite) sender() {
	for m := range g.ch {
		g.Write(m)
	}
}

// Send a single metric to graphite
func (g *Graphite) Send(m GraphiteEntry) {
	g.ch <- m
}

func (g *Graphite) Write(m GraphiteEntry) error {
	_, err := g.conn.Write([]byte(m.String()))
	return err
}

func (m *GraphiteEntry) String() string {
	if m.Timestamp == 0 {
		m.Timestamp = time.Now().Unix()
	}
	return fmt.Sprintf("%s %v %d\n", m.Metric, m.Value, m.Timestamp)
}

func buildGraphiteEntries(centries []CollectdEntry) []GraphiteEntry {
	ret := []GraphiteEntry{}

	for _, centry := range centries {
		if centry.Interval != 10 {
			fmt.Println("Interval is ", centry.Interval, centry.Plugin)
		}
		for iv := range centry.Values {
			var gentry GraphiteEntry
			entryName := centry.Plugin
			tinstance := centry.TypeInstance
			//plugin dependant (ex: cpu.0 or just interface)
			if centry.PluginInstance != "" {
				entryName = fmt.Sprintf("%s.%s", entryName, graphiteFriendly(centry.PluginInstance))
			}
			if tinstance != "" {
				tinstance = fmt.Sprintf("%s.", tinstance)
			}
			gentry.Metric = fmt.Sprintf("collectd.%s.%s.%s%s.%s",
				graphiteFriendly(centry.Host),
				entryName,
				tinstance,
				centry.Type,
				centry.DSNames[iv])
			gentry.Timestamp = centry.Time
			gentry.Value = centry.Values[iv]
			ret = append(ret, gentry)
		}
	}

	return ret
}

func handler(graphite *Graphite, w http.ResponseWriter, r *http.Request) {

	var centries []CollectdEntry
	body, err := ioutil.ReadAll(r.Body)

	if err != nil {
		w.Write([]byte(err.Error()))
	}

	err = json.Unmarshal(body, &centries)

	if err != nil {
		w.Write([]byte(err.Error()))
	}

	gentries := buildGraphiteEntries(centries)

	total := len(gentries)
	errors := 0

	for _, gentry := range gentries {
		err = graphite.Write(gentry)
		if err != nil {
			errors++
		}
	}

	w.Write([]byte(fmt.Sprintf("Ok: %d\nErrors: %d\n", total, errors)))
}

func main() {
	graphiteEndpoint := flag.String("graphite", "localhost:2003", "Graphite LineReceiver port")
	httpEndpoint := flag.String("http", ":9292", "HTTP Listener")
	flag.Parse()
	graphite := NewGraphite(*graphiteEndpoint)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		handler(graphite, w, r)
	})
	http.ListenAndServe(*httpEndpoint, nil)
}
