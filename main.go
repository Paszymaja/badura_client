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
	"time"
)

const (
	channelID = "387298617431425025"
)

var eventsChan = make(chan Events)
var Started = false

func main() {

	fmt.Println("Starting boi client ...")

	clientURL := flag.String("client.url", "https://127.0.0.1:2999", "url of league client")
	serverURL := flag.String("server.url", "https://discord-js-boi-bot.herokuapp.com", "url of output server")
	flag.Parse()

	c := &http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}}
	ctx := context.Background()

	var summonerName string
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case _ = <-ticker.C:
			go getEvents(ctx, c, *clientURL)

		case e, ok := <-eventsChan:
			if ok {
				if Started == false {
					if len(summonerName) == 0 {
						summonerName = getSummonerName(ctx, c, *clientURL)
					}
					startEvent := NewGameStart(e, summonerName)
					fmt.Println("Sending game start data to server")
					sendEvent(ctx, c, startEvent, *serverURL, "game_started")
					Started = true
				} else {
					deathEvent := NewDeath(e, summonerName)
					fmt.Println("Sending game death data to server")
					sendEvent(ctx, c, deathEvent, *serverURL, "death")
				}
			}
		}
	}
}

func sendEvent(ctx context.Context, c *http.Client, v interface{}, serverURL string, url string) {

	e, err := json.Marshal(v)
	if err != nil {
		fmt.Println("Error encoding JSON")
		fmt.Println(err)
		return
	}

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/%s", serverURL, url), bytes.NewBuffer(e))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Accept", "application/json; charset=utf-8")
	req = req.WithContext(ctx)

	resp, err := c.Do(req)
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

func getEvents(ctx context.Context, c *http.Client, clientURL string) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/liveclientdata/eventdata", clientURL), nil)
	req = req.WithContext(ctx)

	resp, err := c.Do(req)
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

func getSummonerName(ctx context.Context, c *http.Client, clientURL string) string {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/liveclientdata/activeplayername", clientURL), nil)
	req = req.WithContext(ctx)

	resp, err := c.Do(req)
	if err != nil || resp.StatusCode/100 != 2 {
		fmt.Println(err)
		return ""
	}

	body, err := ioutil.ReadAll(resp.Body)
	s := string(body)
	s = s[1 : len(s)-1]
	return s
}

func NewGameStart(e Events, sm string) *GameStart {

	gs := GameStart{SummonerName: sm,
		EventID:   e.Events[0].EventID,
		EventName: e.Events[0].EventName,
		EventTime: e.Events[0].EventTime,
		ChannelId: channelID,
	}

	return &gs

}

func NewDeath(e Events, sm string) *PlayerDeath {

	var pd PlayerDeath

	for i, v := range e.Events {
		if v.VictimName == sm {
			pd = PlayerDeath{
				EventID:    e.Events[i].EventID,
				EventName:  e.Events[i].EventName,
				EventTime:  e.Events[i].EventTime,
				KillerName: e.Events[i].KillerName,
				VictimName: e.Events[i].VictimName,
			}
		}
	}
	return &pd
}
