package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"time"
)

const (
	ClientURL = "https://127.0.0.1:2999"
	ServerURL = "https://httpbin.org"
)

// A backoff schedule for when and how often to retry failed HTTP
// requests. The first element is the time to wait after the
// first failure, the second the time to wait after the second
// failure, etc. After reaching the last element, retries stop
// and the request is considered failed.
var backoffSchedule = []time.Duration{
	10 * time.Second,
	20 * time.Second,
	30 * time.Second,
}

type Task struct {
	closed chan struct{}
	wg     sync.WaitGroup
	ticker *time.Ticker
}

type Client struct {
	ClientURL  string
	ServerURL  string
	HTTPClient *http.Client
}

type EventsStruck struct {
	Events Event `json:"Events"`
}

type Event []struct {
	EventID    int           `json:"EventID"`
	EventName  string        `json:"EventName"`
	EventTime  float64       `json:"EventTime"`
	Assisters  []interface{} `json:"Assisters,omitempty"`
	KillerName string        `json:"KillerName,omitempty"`
	VictimName string        `json:"VictimName,omitempty"`
}

type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
}

func NewClient() *Client {
	return &Client{
		ClientURL: ClientURL,
		ServerURL: ServerURL,
		HTTPClient: &http.Client{
			Timeout:   time.Minute,
			Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}},
		},
	}
}

func (c *Client) GetEvents(ctx context.Context) (*EventsStruck, error) {

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/liveclientdata/eventdata", c.ClientURL), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	res := EventsStruck{}
	if err := c.sendRequest(req, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) PushEvents(ctx context.Context, v interface{}) (*Response, error) {
	EventsJSON, err := json.Marshal(v)
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/anything", c.ServerURL), bytes.NewBuffer(EventsJSON))
	if err != nil {
		return nil, err
	}

	req = req.WithContext(ctx)

	res := Response{}
	if err := c.sendRequest(req, &res); err != nil {

		return nil, err
	}
	return &res, nil

}

func (c *Client) sendRequest(req *http.Request, v interface{}) error {

	var err error
	var res *http.Response

	for _, backoff := range backoffSchedule {
		res, err = c.HTTPClient.Do(req)

		if err == nil {
			break
		}
		log.Printf("Request error: %+v\n", err)
		log.Printf("Retrying in %v\n", backoff)
		time.Sleep(backoff)
	}

	if err != nil {
		return err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(res.Body)

	if err != nil || res.StatusCode/100 != 2 {
		return fmt.Errorf("unknown error, status code: %d", res.StatusCode)
	}

	if err = json.NewDecoder(res.Body).Decode(&v); err != nil {
		return err
	}

	return nil
}

func (t *Task) Run(client *Client, ctx context.Context) {
	for {
		select {
		case <-t.closed:
			return
		case <-t.ticker.C:
			event, err := client.GetEvents(ctx)
			if err != nil {
				log.Fatal(err)
			}
			if event != nil {
				log.Println("Events detected. Will push")
				_, err = client.PushEvents(ctx, event)
			}
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}

func (t *Task) Stop() {
	close(t.closed)
	t.wg.Wait()
}

func main() {

	log.Println("Starting Badura Client ...")

	client := NewClient()
	ctx := context.Background()

	task := &Task{
		closed: make(chan struct{}),
		ticker: time.NewTicker(time.Second * 5),
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt)
	task.wg.Add(1)
	go func() { defer task.wg.Done(); task.Run(client, ctx) }()

	select {
	case sig := <-c:
		log.Printf("Got %s signal. Aborting...\n", sig)
		task.Stop()
	}
}
