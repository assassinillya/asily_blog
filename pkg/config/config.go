package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	Server struct {
		Port string `json:"port"`
	} `json:"server"`
	User         string `json:"user"`
	Password     string `json:"password"`
	MongoDB      string `json:"mongodb"`
	AccessSecret string `json:"accessSecret"`
}

var C Config

func LoadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&C)
	if err != nil {
		panic(err)
	}
}
