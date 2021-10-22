package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// A backoff schedule for when and how often to retry failed HTTP
// requests. The first element is the time to wait after the
// first failure, the second the time to wait after the second
// failure, etc. After reaching the last element, retries stop
// and the request is considered failed.

var backoffSchedule = []time.Duration{

	10 * time.Second,
	10 * time.Second,
	10 * time.Second,
}

var Started = false

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

func NewClient(clientURL string, serverURL string, timeout time.Duration) *Client {
	return &Client{
		ClientURL:  clientURL,
		ServerURL:  serverURL,
		HTTPClient: newHttpClient(timeout),
	}
}

func newHttpClient(timeout time.Duration) *http.Client {
	tl := &tls.Config{
		InsecureSkipVerify: true,
	}
	t := &http.Transport{
		IdleConnTimeout:     timeout,
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		TLSClientConfig:     tl,
	}
	return &http.Client{
		Timeout:   timeout,
		Transport: t,
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

func (c *Client) PushDeath(ctx context.Context, v interface{}) (*Response, error) {
	EventsJSON, err := json.Marshal(v)
	fmt.Println(string(EventsJSON))
	req, err := http.NewRequest("POST", fmt.Sprintf("%s/death", c.ServerURL), bytes.NewBuffer(EventsJSON))
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

func (c *Client) PushGameStart(ctx context.Context, v interface{}) (*Response, error) {
	EventsJSON, err := json.Marshal(v)
	fmt.Println(string(EventsJSON))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/game_started", c.ServerURL), bytes.NewBuffer(EventsJSON))
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

func NewGameStart(event *EventsStruck, summonerName string) *GameStart {

	gs := GameStart{SummonerName: summonerName,
		EventID:   event.Events[0].EventID,
		EventName: event.Events[0].EventName,
		EventTime: event.Events[0].EventTime,
		ChannelId: "387298617431425025",
	}

	return &gs

}

func NewDeath(event EventsStruck, summonerName string) *PlayerDeath {

	var pd PlayerDeath

	for i, v := range event.Events {
		if v.VictimName == summonerName {
			pd = PlayerDeath{
				EventID:    event.Events[i].EventID,
				EventName:  event.Events[i].EventName,
				EventTime:  event.Events[i].EventTime,
				KillerName: event.Events[i].KillerName,
				VictimName: event.Events[i].VictimName,
			}
		}
	}

	return &pd

}

func (c *Client) sendRequest(req *http.Request, v interface{}) error {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

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

func (t *Task) Run(client *Client, ctx context.Context, summonerName string) {
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

				if Started == false {
					startEvent := NewGameStart(event, summonerName)
					log.Printf("GameStart detected. Pushing to %s\n", client.ServerURL)
					_, err = client.PushGameStart(ctx, startEvent)
					Started = true
				} else {
					DeathEvent := NewDeath(*event, summonerName)
					log.Printf("NewDeath detected. Pushing to %s\n", client.ServerURL)
					_, err = client.PushDeath(ctx, DeathEvent)
				}
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

	timeout := flag.Duration("client.timeout", 5, "client connection timeout")
	clientURL := flag.String("client.url", "https://127.0.0.1:2999", "url of league client")
	serverURL := flag.String("server.url", "https://discord-js-boi-bot.herokuapp.com", "url of output server")
	flag.Parse()

	c, err := New(*clientURL, *serverURL, newHttpClient(*timeout))
	GetEvents(c, context.Background())
	if err != nil {
		fmt.Print("err", "Failed to create client")
	}
	go setupShutdownHandler(c)
}

func setupShutdownHandler(client TestClient) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	fmt.Print("\nReceived an interrupt, stopping services...\n")
	client.Stop()
	os.Exit(0)
}
