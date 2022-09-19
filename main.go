package main

import (
	"bytes"
	"encoding/json"
	"errors"
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

// parses data from array and rounds it to two decimals accuracy
func parseResults(results []uint8) (float32, float32) {
	temperature := math.Floor((float64(results[1])/10.0)*100) / 100
	humidity := math.Floor((float64(results[3])/10.0)*100) / 100
	return float32(temperature), float32(humidity)
}

// read dato from 2 XT-MD02 temperature sensors
func readData(handler *modbus.RTUClientHandler, data *parsedData) {
	for i := 1; i <= 2; i++ {
		handler.SlaveId = byte(i)
		client := modbus.NewClient(handler)

		results, err := client.ReadInputRegisters(1, 2)

		if err != nil {
			fmt.Println("Reading error: ", err)
		}

		temperature, humidity := parseResults(results)

		if i == 1 {
			data.OutsideTemperature = temperature
			data.OutsideHumidity = humidity
		} else {
			data.InsideTemperature = temperature
			data.InsideHumidity = humidity
		}
	}
}

// sends measurement data to backend.
// input: data wich will be send to backend.
// output: http staus code or error code.
func sendData(data []byte) (string, error) {

	resp, err := http.Post("http://localhost:8080/temperatures", "application/json", bytes.NewBuffer(data))

	if err != nil {
		fmt.Println(err)
	}

	var res map[string]interface{}

	json.NewDecoder(resp.Body).Decode(&res)

	if resp.Status != "201" {
		return "", errors.New("Networking error")
	}

	return "", nil
}

func main() {

	// laod enviroment variables
	godotenv.Load(".env")

	//Configurate modbus connection
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

	// create struct where data will be stored
	var measurementData = parsedData{
		Key: os.Getenv("Key"),
	}

	// this loop runs
	for ok := true; ok; ok = true {

		readData(handler, &measurementData)

		json, err := json.Marshal(measurementData)

		fmt.Println(measurementData)

		if err != nil {
			fmt.Println(err)
		}

		_, NetworkError := sendData(json)

		if NetworkError != nil {
			fmt.Println("Error sending data: ", NetworkError)
		}

		// measurements are done in one hour intervals
		time.Sleep(1 * time.Hour)
	}
}
