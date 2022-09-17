package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"os"
	"time"

	"github.com/goburrow/modbus"
	"github.com/joho/godotenv"
)

type parsedData struct {
	Key                string
	OutsideTemperature float32
	OutsideHumidity    float32
	InsideTemperature  float32
	InsideHumidity     float32
}

func parseResults(results []uint8) (float32, float32) {
	temperature := math.Floor((float64(results[1])/10.0)*100) / 100
	humidity := math.Floor((float64(results[3])/10.0)*100) / 100
	return float32(temperature), float32(humidity)
}

func readData(h *modbus.RTUClientHandler, data *parsedData) {
	for i := 1; i <= 2; i++ {
		h.SlaveId = byte(i)
		client := modbus.NewClient(h)

		results, err := client.ReadInputRegisters(1, 2)

		if err != nil {
			fmt.Println("Reading error: ", err)
		}

		t, h := parseResults(results)

		if i == 1 {
			data.OutsideTemperature = t
			data.OutsideHumidity = h
		} else {
			data.InsideTemperature = t
			data.InsideHumidity = h
		}
	}
}

func sendData(data []byte) {

	resp, err := http.Post("http://localhost:8080/temperatures", "application/json", bytes.NewBuffer(data))

	if err != nil {
		fmt.Println(err)
	}

	var res map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&res)

	fmt.Println(resp.Status)
}

func main() {

	godotenv.Load(".env")
	handler := modbus.NewRTUClientHandler("COM2")
	handler.BaudRate = 9600
	handler.DataBits = 8
	handler.Parity = "N"
	handler.StopBits = 1
	handler.Timeout = 1 * time.Second

	err := handler.Connect()

	if err != nil {
		fmt.Println(err)
	}

	defer handler.Close()

	var data = parsedData{
		Key: os.Getenv("Key"),
	}

	readData(handler, &data)

	j, err := json.Marshal(data)

	if err != nil {
		fmt.Println(err)
	}

	sendData(j)
}
