package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
)

func NewClient() *Client {
	return &Client{
		ctx:        context.Background(),
		HTTPClient: newHTTPClient(),
		config:     newConfig(),
	}
}

func newConfig() *config {

	c := flag.String("client.url", "https://127.0.0.1:2999", "url of league client")
	s := flag.String("server.url", "https://discord-js-boi-bot.herokuapp.com", "url of output server")
	flag.Parse()
	return &config{
		clientURL: *c,
		serverURL: *s,
	}
}

func newHTTPClient() *http.Client {
	tl := &tls.Config{
		InsecureSkipVerify: true,
	}
	t := &http.Transport{
		MaxIdleConnsPerHost: 30,
		MaxConnsPerHost:     30,
		TLSClientConfig:     tl,
	}
	return &http.Client{
		Transport: t,
	}
}

func (c *Client) getEvents() {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/liveclientdata/eventdata", c.config.clientURL), nil)
	req = req.WithContext(c.ctx)

	resp, err := c.HTTPClient.Do(req)
	if err != nil || resp.StatusCode/100 != 2 {
		fmt.Println("Waiting for league client ...")
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(resp.Body)

	events := Events{}
	err = json.NewDecoder(resp.Body).Decode(&events)
	if err != nil {
		return
	}

	if reflect.ValueOf(events).IsValid() {
		if events.Events[0].EventName != "" {
			eventsChan <- events
		}
	}
}

func (c *Client) getSummonerName() string {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/liveclientdata/activeplayername", c.config.clientURL), nil)
	req = req.WithContext(c.ctx)

	resp, err := c.HTTPClient.Do(req)
	if err != nil || resp.StatusCode/100 != 2 {
		fmt.Println(err)
		return ""
	}

	body, err := ioutil.ReadAll(resp.Body)
	s := string(body)
	s = s[1 : len(s)-1]
	return s
}
func (c *Client) sendEvent(v interface{}, path string) {

	e, err := json.Marshal(v)
	if err != nil {
		fmt.Println("Error encoding JSON")
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", c.config.serverURL, path), bytes.NewBuffer(e))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(c.ctx)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
		}
	}(resp.Body)
}
