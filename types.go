package main

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
	KillerName string  `json:"KillerName,omitempty"`
	VictimName string  `json:"VictimName,omitempty"`
}

type Response struct {
	Status string `json:"Status"`
}
