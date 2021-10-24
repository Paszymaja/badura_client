package main

import (
	"fmt"
	"time"
)

const (
	channelID = "387298617431425025"
)

var eventsChan = make(chan Events)
var Started = false

func main() {

	fmt.Println("Starting boi client ...")

	c := NewClient()

	var summonerName string
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case _ = <-ticker.C:
			go c.getEvents()

		case e, ok := <-eventsChan:
			if ok {
				if Started == false {
					if len(summonerName) == 0 {
						summonerName = c.getSummonerName()
					}
					startEvent := NewGameStart(e, summonerName)
					fmt.Println("Sending game start data to server")
					c.sendEvent(startEvent, "game_started")
					Started = true
				} else {
					deathEvent := NewDeath(e, summonerName)
					fmt.Println("Sending game death data to server")
					c.sendEvent(deathEvent, "death")
				}
			}
		}
	}
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
