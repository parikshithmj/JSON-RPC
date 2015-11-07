package main

import (
    	"fmt"
    	"net/http"
    	"io/ioutil"
    	"bytes"
    	"os"
		)

func main() {
	//make sure Json-rpc-server.go is running
	//buy stocks usage  : go run filename budget "stock values with percentage"
	//example:go run Json-rpc-client.go 7000 "AAPL:50,YHOO:40,GOOG:10"
	
	//check portfolio usage  : go run filename tradeid
    url := "http://127.0.0.1:8080/rpc"
    var stringJson string
    if len(os.Args)==3{
    budget := os.Args[1]
    StockSymbolAndPercentage := os.Args[2]
    stringJson = "{\"method\":\"Calculator.BuyStocks\",\"params\":[{\"Budget\":"+budget+",\"StockSymbolAndPercentage\":\""+StockSymbolAndPercentage+"\"}],\"id\":1}"
    }else if len(os.Args)==2{
    TradeId :=os.Args[1]
    stringJson = "{\"method\":\"Calculator.CheckPortfolio\",\"params\":[{\"TradeId\":"+TradeId +"}],\"id\":1}"
    }
    var payload = []byte(stringJson)
    req, err := http.NewRequest("POST", url, bytes.NewBuffer(payload))
 	
    req.Header.Set("Content-Type", "application/json")

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()
    body, _ := ioutil.ReadAll(resp.Body)
    fmt.Println("Response Body:", string(body))
}
