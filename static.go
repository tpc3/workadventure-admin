package main

import (
	"encoding/json"
	"log"
	"os"
)

var WokaDB map[string]WokaPart
var WokaKV = make(map[string]string)

type WokaPart struct {
	Required    bool             `json:"required"`
	Collections []WokaCollection `json:"collections"`
}

type WokaCollection struct {
	Name     string        `json:"name"`
	Position int           `json:"position"`
	Textures []WokaTexture `json:"textures"`
}

type WokaTexture struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Position int    `json:"position"`
}

var CompanionDB []CompanionCollection
var CompanionKV = make(map[string]string)

type CompanionCollection struct {
	Name     string             `json:"name"`
	Position int                `json:"position"`
	Textures []CompanionTexture `json:"textures"`
}

type CompanionTexture struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Behavior string `json:"behavior"`
	URL      string `json:"url"`
}

func LoadFiles() {
	wokaFile, err := os.Open("woka.json")
	if err != nil {
		log.Panic("Failed to open woka.json: ", err)
	}
	defer wokaFile.Close()
	if err := json.NewDecoder(wokaFile).Decode(&WokaDB); err != nil {
		log.Panic("Failed to decode woka.json: ", err)
	}
	for _, part := range WokaDB {
		for _, collection := range part.Collections {
			for _, texture := range collection.Textures {
				WokaKV[texture.ID] = texture.URL
			}
		}
	}
	companionsFile, err := os.Open("companions.json")
	if err != nil {
		log.Panic("Failed to open companions.json: ", err)
	}
	defer companionsFile.Close()
	if err := json.NewDecoder(companionsFile).Decode(&CompanionDB); err != nil {
		log.Panic("Failed to decode companions.json: ", err)
	}
	for _, collection := range CompanionDB {
		for _, texture := range collection.Textures {
			CompanionKV[texture.ID] = texture.URL
		}
	}
}
