package main

import (
	"log"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/google/uuid"
)

type ConfigStruct struct {
	Token            string            `json:"token"`
	UUIDSpace        uuid.UUID         `json:"uuid_space"`
	UserinfoEndpoint string            `json:"userinfo_endpoint"`
	Redirects        map[string]string `json:"redirects"`
	Maps             map[string]MapStruct
	Tags             map[string][]string
}

type MapStruct struct {
	MapUrl      string   `json:"mapUrl,omitempty"`
	WamUrl      string   `json:"wamUrl,omitempty"`
	Group       string   `json:"group"`
	RoomName    string   `json:"roomName"`
	EditorTags  []string `json:"-" yaml:"editor_tags"`
	AllowedTags []string `json:"-" yaml:"allowed_tags"`
}

var Config ConfigStruct

var GroupMap = make(map[string][]string)

func LoadConfig() {
	f, err := os.Open("config.yaml")
	if err != nil {
		log.Panic("Failed to open config.yaml: ", err)
	}
	defer f.Close()
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&Config); err != nil {
		log.Panic("Failed to decode config: ", err)
	}

	for k, m := range Config.Maps {
		GroupMap[m.Group] = append(GroupMap[m.Group], k)
	}
}
