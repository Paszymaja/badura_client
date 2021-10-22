package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"
)

type TestClient interface {
	Chan() chan<- Event
	Stop()
}

type client struct {
	ClientURL string
	ServerURL string
	events    chan Event
	client    *http.Client
	once      sync.Once
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	Timeout   time.Duration

	BackoffConfig BackoffConfig

	getEventsFunc     func()
	sendGameStartFunc func(start GameStart)
	sendDeathFunc     func(death PlayerDeath)
}

func (c *client) GetEvents(ctx context.Context) (*EventsStruck, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/liveclientdata/eventdata", c.ClientURL), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	res := EventsStruck{}
	if err := c.sendRequestE(req, &res); err != nil {
		return nil, err
	}
	return &res, nil

}

func New(clientURL string, serverURL string, httpClient *http.Client) (TestClient, error) {
	ctx, cancel := context.WithCancel(context.Background())

	c := &client{ClientURL: clientURL,
		ServerURL: serverURL,
		events:    make(chan Event),
		ctx:       ctx,
		cancel:    cancel,
		wg:        sync.WaitGroup{},
		Timeout:   5}
	c.sendGameStartFunc = c.sendGameStart
	c.sendDeathFunc = c.sendDeath

	c.BackoffConfig = BackoffConfig{
		MinBackoff: 10,
		MaxBackoff: 10,
		MaxRetries: 10,
	}

	c.client = httpClient

	c.wg.Add(1)
	go c.run()
	return c, nil
}

func (c *client) Chan() chan<- Event {
	return c.events
}

func (c *client) Stop() {
	c.once.Do(func() { close(c.events) })
	c.wg.Wait()
}

func (c *client) run() {

	maxWaitCheckFrequency := 100 * time.Millisecond
	maxWaitCheck := time.NewTicker(maxWaitCheckFrequency)
	event, _ := c.GetEvents(context.Background())

	defer func() {
		maxWaitCheck.Stop()
		c.wg.Done()
	}()

	for {
		select {
		case e, ok := <-c.events:
			if !ok {
				fmt.Println(e)
				return
			}
		case <-maxWaitCheck.C:
			summonerName := "Paszymaja"
			DeathEvent := NewDeath(event, summonerName)
			c.sendDeathFunc(*DeathEvent)
		}
	}
}

func (c *client) sendGameStart(start GameStart) {

	gs, err := json.Marshal(start)
	if err != nil {
		log.Printf("Error encoding GameStart")
		return
	}
	fmt.Println(string(gs))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/game_started", c.ServerURL), bytes.NewBuffer(gs))

	backoff := NewBackoff(c.ctx, c.BackoffConfig)
	var status int

	for {
		status, err = c.sendRequest(context.Background(), req)
		if err == nil {
			log.Printf("Request send")
			return
		}
		// Only retry 429s, 500s and connection-level errors.
		if status > 0 && status != 429 && status/100 != 5 {
			break
		}
		log.Printf("Only retry 429s, 500s and connection-level errors.")
		backoff.Wait()

		if !backoff.Ongoing() {
			break
		}
	}
}

func (c *client) sendRequest(ctx context.Context, req *http.Request) (int, error) {

	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	req = req.WithContext(ctx)

	defer cancel()

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	resp, err := c.client.Do(req)
	if err != nil {
		return -1, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if resp.StatusCode/100 != 2 {
		scanner := bufio.NewScanner(io.LimitReader(resp.Body, 256))
		line := ""
		if scanner.Scan() {
			line = scanner.Text()
		}
		err = fmt.Errorf("server returned HTTP status %s (%d): %s", resp.Status, resp.StatusCode, line)
	}
	return resp.StatusCode, err
}

func (c *client) sendDeath(death PlayerDeath) {
	gs, err := json.Marshal(death)
	if err != nil {
		log.Printf("Error encoding GameStart")
		return
	}
	fmt.Println(string(gs))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/death", c.ServerURL), bytes.NewBuffer(gs))

	backoff := NewBackoff(c.ctx, c.BackoffConfig)
	var status int

	for {
		status, err = c.sendRequest(context.Background(), req)
		if err == nil {
			log.Printf("Request send")
			return
		}
		// Only retry 429s, 500s and connection-level errors.
		if status > 0 && status != 429 && status/100 != 5 {
			break
		}
		log.Printf("Only retry 429s, 500s and connection-level errors.")
		backoff.Wait()

		if !backoff.Ongoing() {
			break
		}
	}
}

func (c *client) sendRequestE(req *http.Request, v interface{}) error {
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")

	var err error
	var res *http.Response

	for _, backoff := range backoffSchedule {
		res, err = c.client.Do(req)

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
