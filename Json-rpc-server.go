package main
import (
	"fmt"
	"log"
	"net/http"
	"io/ioutil"
	"strings"
	"os"
	"errors"
	"strconv"
	"github.com/bakins/net-http-recover"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/rpc"
	"github.com/gorilla/rpc/json"
	"github.com/justinas/alice"
)
var tradeid int =0
var globalMap map[int]string
type BuyStockRequest struct {
	StockSymbolAndPercentage string 
	Budget float32
}

type BuyStockResponse struct {
	TradeId int
	Stocks string 
	UnvestedAmount float32

}

type CheckPortfolioRequest struct {
	TradeId int
}

type CheckPortfolioResponse struct {
	Stocks string
	CurrentMarketValue float32
	UnvestedAmount float32
}

type Calculator int

func (t *Calculator) BuyStocks(r *http.Request,req *BuyStockRequest, reply *BuyStockResponse) error {
	
	tradeid++
	reply.TradeId=tradeid
	
	stocksMap := make(map[int]string)
    reqarr :=strings.Split(req.StockSymbolAndPercentage,",")
    //store the requests in a map
    for in,s := range reqarr {
   		stocksMap[in] = s
	} 
	var params string
	var percentage,totalPercentage float64
	//form the request params by extracting the company Symbols
	for key:= range stocksMap{
		s := stocksMap[key]
		tmp := strings.Split(s,":")
   		percentage,_ = strconv.ParseFloat(tmp[1],64)
   		totalPercentage = totalPercentage +percentage
   		params =params+tmp[0]+"+"
	}
	if totalPercentage !=100{
		return errors.New("Percentages of stocks doesnt add upto 100percent")
	}
	
	contents:= invokeYahooApi(params)
	stocksRateMap := make(map[int]float64)
	//save the stock rates in memory
	con :=strings.Split(string(contents),"\n")
		
        for i, ss := range con {
        	if strings.Contains(ss,"N/A"){
    		  return errors.New("Invalid Company Symbol")
   			}
        	stocksRateMap[i],_ =strconv.ParseFloat(ss,64)
		} 
	//start forming the response
	var remaining float32 = req.Budget
	var stocks string
	var info string
	
	for key:= range stocksMap{
		s := stocksMap[key]
		tmp := strings.Split(s,":")
   		percentage,_ = strconv.ParseFloat(tmp[1],64)
   		//amount allocated for each company
   		amount := (float32(percentage)/100)*req.Budget
   		tmpRem := int(amount) % int(stocksRateMap[key])
   		finalstr := (amount)/float32(stocksRateMap[key])
   		remaining = remaining -(finalstr*float32(stocksRateMap[key]))+float32(tmpRem)
   		fmt.Println("No of shares of",tmp[0]," is ",int(finalstr),"tmp rem",tmpRem,"remaining",remaining,"worth",int(stocksRateMap[key])*int(finalstr))
   		info = info + tmp[0]+","+strconv.Itoa(int(finalstr)) + ","+strconv.Itoa(int(stocksRateMap[key])*int(finalstr))+","
   		
   		stocks = stocks+tmp[0]+":"+strconv.Itoa(int(finalstr))+":$"+strconv.Itoa(int(stocksRateMap[key])*int(finalstr))+","
   		
	}
	info = info+strconv.FormatFloat(float64(remaining),'f',-1,32)
	globalMap[tradeid]=info
	fmt.Println(globalMap[tradeid])
	reply.Stocks = stocks
	reply.UnvestedAmount = remaining
	return nil
}


func invokeYahooApi(params string) []byte{
	url:="http://finance.yahoo.com/d/quotes.csv?s="
	format:="&f=a"
	response,_ := http.Get(url+params+format)
	contents,_ := ioutil.ReadAll(response.Body)
	return contents
}	

func (t *Calculator) CheckPortfolio(r *http.Request,args *CheckPortfolioRequest, reply *CheckPortfolioResponse) error {
	id := (args.TradeId)
	//get the info in a csv
	info := globalMap[id]
	fmt.Println("Checking portfolio ********",info)
	tmp,_:= strconv.ParseFloat(info[strings.LastIndex(info,",")+1:len(info)],32)
	reply.UnvestedAmount = float32(tmp)
	tmpInfo := strings.Split(info,",")
	var params string
	for i:=0;i<len(tmpInfo)-1;i=i+3{
	params =params+tmpInfo[i]+"+"
	}
	contents:= invokeYahooApi(params)
	stocksRateMap := make(map[int]float64)
	//save the results of stock price from yahoo api
	con :=strings.Split(string(contents),"\n")		
        for i, ss := range con {
        	if strings.Contains(ss,"N/A"){
    		  return errors.New("Invalid Company Symbol")
   			}
        	stocksRateMap[i],_ =strconv.ParseFloat(ss,64)
		}
		var j int
		j = 0
		var marketValue float64
		marketValue = 0
		for i:=0;i<len(tmpInfo)-1;i++{
			if i%3==2{
				curRate := stocksRateMap[j]
				noOfShares,_ :=(strconv.ParseFloat(tmpInfo[i-1],64))
				value:= curRate * noOfShares
				fmt.Println("tmpInfo[i]",tmpInfo[i],"value is ",value,"no of shares",noOfShares)
				old,_:= strconv.ParseFloat(tmpInfo[i],32)
				fmt.Println("old is ",old,"curRate is ",value)
				marketValue = marketValue +value
				if curRate >(old/noOfShares){
					tmpInfo[i] ="+$" +strconv.FormatFloat(value,'f',-1,64)
				}else if curRate<(old/noOfShares) {
					tmpInfo[i] ="-$" +strconv.FormatFloat(value,'f',-1,64)
				}else{
					tmpInfo[i] ="$" +strconv.FormatFloat(value,'f',-1,64)
				}
				j++
			}
		}
	//form the response string
	var respStocks string
	for i:=0;i<len(tmpInfo)-1;i++{
		if i%3!=0{
			respStocks = respStocks + tmpInfo[i] +":"
		}else{
			respStocks = respStocks +"," + tmpInfo[i] +":"
		}
	}

	reply.CurrentMarketValue= float32(marketValue)
	reply.Stocks = respStocks
	return nil
}

func main() {
	globalMap = make(map[int]string)
	r := mux.NewRouter()

	s := rpc.NewServer()
	s.RegisterCodec(json.NewCodec(), "application/json")

	cal := new(Calculator)
	s.RegisterService(cal, "")

	chain := alice.New(
		func(h http.Handler) http.Handler {
			return handlers.CombinedLoggingHandler(os.Stdout, h)
		},
		handlers.CompressHandler,
		func(h http.Handler) http.Handler {
			return recovery.Handler(os.Stderr, h, true)
		})

    r.Handle("/rpc", chain.Then(s))
	log.Fatal(http.ListenAndServe(":8080", r))
}
