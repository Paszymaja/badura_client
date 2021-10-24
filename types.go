package main

import (
	"context"
	"net/http"
)

type config struct {
	clientURL string
	serverURL string
}

type Client struct {
	ctx        context.Context
	HTTPClient *http.Client
	config     *config
}

type Events struct {
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

type GameStart struct {
	SummonerName string  `json:"SummonerName"`
	EventID      int     `json:"EventID"`
	EventName    string  `json:"EventName"`
	EventTime    float64 `json:"EventTime"`
	ChannelId    string  `json:"ChannelId"`
}

type PlayerDeath struct {
	EventID    int     `json:"EventID"`
	EventName  string  `json:"EventName"`
	EventTime  float64 `json:"EventTime"`
	KillerName string  `json:"KillerName"`
	VictimName string  `json:"VictimName"`
}
