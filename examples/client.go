package main

import (
	"flag"
	"github.com/Forau/nordnet/api"
	"github.com/Forau/nordnet/api/rpc"

	"log"

	"fmt"
	"os"

	"encoding/json"
	"strconv"
	"strings"
)

var (
	srv = flag.String("srv", ":2008", "RPC server to bind to. The address daemon is bound to")
)

type ArgType uint64

const (
	Unused ArgType = iota
	Number
	String
	Params
)

type Command struct {
	Name    string
	Args    []ArgType
	argRest []string // Unparsed args
	Func    func(*Command, *api.APIClient) (interface{}, error)
}

func (c *Command) GetStr() string {
  if len(c.argRest) == 0 {
    log.Fatal("Not enough arguments for: ",c.String())
  }
	a := c.argRest[0]
	c.argRest = c.argRest[1:]
	return a
}

func (c *Command) GetNum() int64 {
	i, _ := strconv.ParseInt(c.GetStr(), 10, 64)
	return i
}

func (c *Command) GetPar() *api.Params {
	r := api.Params{}
  if len(c.argRest) & 1 != 0 {
    log.Fatal("Need an odd number of arguments for params. Key Value.")
  }
	for i := 0; i < len(c.argRest)-1; i += 2 {
		r[c.argRest[i]] = c.argRest[i+1]
	}
	return &r
}

func (c *Command) Call(a *api.APIClient, args []string) (res string, err error) {
	c.argRest = args
	ob, err := c.Func(c, a)
	if err == nil {
		resb, err2 := json.MarshalIndent(ob, "", "\t")
		if err2 != nil {
			err = err2
		} else {
			res = string(resb)
		}
	}
	return
}

func (c *Command) String() string {
	buf := []byte(c.Name)
	for _, a := range c.Args {
		buf = append(buf, []byte("\t")...)
		switch a {
		case Number:
			buf = append(buf, []byte("<num>")...)
		case String:
			buf = append(buf, []byte("<str>")...)
		case Params:
			buf = append(buf, []byte("{<param key>\t<param value>\t...}")...)
			return string(buf)
		}
	}
	return string(buf)
}

func (c *Command) IsCmd(cmd string) bool {
	return strings.ToLower(cmd) == strings.ToLower(c.Name)
}

func init() {
	flag.Parse()

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s <flags> CMD <args>...\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Commands: (Command name is case insensitive)\n")
		for _, c := range CommandList {
			fmt.Fprintf(os.Stderr, "\t%s\n", c.String())
		}
	}
}

var CommandList = []Command{
	{Name: "SystemStatus", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.SystemStatus() }},
	{Name: "Login", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Login() }},
	{Name: "Logout", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Logout() }},
	{Name: "Touch", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Touch() }},
	{Name: "Accounts", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Accounts() }},
	{Name: "Account", Args: []ArgType{Number}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Account(c.GetNum()) }},
	{Name: "AccountLedgers", Args: []ArgType{Number}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.AccountLedgers(c.GetNum()) }},

	{Name: "AccountOrders", Args: []ArgType{Number, Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.AccountOrders(c.GetNum(), c.GetPar())
	}},
	{Name: "ActivateOrder", Args: []ArgType{Number, Number}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.ActivateOrder(c.GetNum(), c.GetNum())
	}},
	{Name: "CreateOrder", Args: []ArgType{Number, Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.CreateOrder(c.GetNum(), c.GetPar()) }},

	{Name: "UpdateOrder", Args: []ArgType{Number, Number, Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.UpdateOrder(c.GetNum(), c.GetNum(), c.GetPar())
	}},

	{Name: "DeleteOrder", Args: []ArgType{Number, Number}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.DeleteOrder(c.GetNum(), c.GetNum()) }},

	{Name: "AccountPositions", Args: []ArgType{Number}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.AccountPositions(c.GetNum()) }},
	{Name: "AccountTrades", Args: []ArgType{Number, Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.AccountTrades(c.GetNum(), c.GetPar())
	}},
	{Name: "Countries", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Countries() }},

	{Name: "LookupCountries", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.LookupCountries(c.GetStr()) }},
	{Name: "Indicators", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Indicators() }},

	{Name: "LookupIndicators", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.LookupIndicators(c.GetStr()) }},

	{Name: "SearchInstruments", Args: []ArgType{Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.SearchInstruments(c.GetPar()) }},

	{Name: "Instruments", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Instruments(c.GetStr()) }},

	{Name: "InstrumentLeverages", Args: []ArgType{Number, Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.InstrumentLeverages(c.GetNum(), c.GetPar())
	}},

	{Name: "InstrumentLeverageFilters", Args: []ArgType{Number, Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.InstrumentLeverageFilters(c.GetNum(), c.GetPar())
	}},

	{Name: "InstrumentOptionPairs", Args: []ArgType{Number, Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.InstrumentOptionPairs(c.GetNum(), c.GetPar())
	}},

	{Name: "InstrumentOptionPairFilters", Args: []ArgType{Number, Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.InstrumentOptionPairFilters(c.GetNum(), c.GetPar())
	}},

	{Name: "InstrumentLookup", Args: []ArgType{String, String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.InstrumentLookup(c.GetStr(), c.GetStr())
	}},

	{Name: "InstrumentSectors", Args: []ArgType{Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.InstrumentSectors(c.GetPar()) }},

	{Name: "InstrumentSector", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.InstrumentSector(c.GetStr()) }},

	{Name: "InstrumentTypes", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.InstrumentTypes() }},

	{Name: "InstrumentType", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.InstrumentType(c.GetStr()) }},

	{Name: "InstrumentUnderlyings", Args: []ArgType{String, String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) {
		return a.InstrumentUnderlyings(c.GetStr(), c.GetStr())
	}},

	{Name: "Lists", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Lists() }},

	{Name: "List", Args: []ArgType{Number}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.List(c.GetNum()) }},

	{Name: "Markets", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Markets() }},

	{Name: "Market", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.Market(c.GetStr()) }},

	{Name: "SearchNews", Args: []ArgType{Params}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.SearchNews(c.GetPar()) }},

	{Name: "News", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.News(c.GetStr()) }},

	{Name: "NewsSources", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.NewsSources() }},

	{Name: "RealtimeAccess", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.RealtimeAccess() }},

	{Name: "TickSizes", Args: []ArgType{}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.TickSizes() }},

	{Name: "TickSize", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.TickSize(c.GetStr()) }},
	{Name: "TradableInfo", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.TradableInfo(c.GetStr()) }},
	{Name: "TradableIntraday", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.TradableIntraday(c.GetStr()) }},
	{Name: "TradableTrades", Args: []ArgType{String}, Func: func(c *Command, a *api.APIClient) (interface{}, error) { return a.TradableTrades(c.GetStr()) }},
}

func main() {
	cli := rpc.NewRpcTransportClient(*srv)
	apic := &api.APIClient{
		Transport: cli,
	}

	/*
	  log.Printf("Cmd: %s", CommandList[0].String())

	  a,e1 := apic.Accounts()
	  log.Printf("Accounts: %+v %+v\n",a,e1)

	  ti,e2 := apic.TradableInfo("11:101")
	  log.Printf("TI: %+v %+v\n",ti,e2)
	*/
	args := flag.Args()
	if len(args) == 0 {
		flag.Usage()
		return
	}
	cmd := args[0]
	args = args[1:]
	for _, c := range CommandList {
		if c.IsCmd(cmd) {
			s, e := c.Call(apic, args)
			if e != nil {
				log.Fatal(e)
			} else {
				log.Print(s)
			}
			return
		}
	}
	flag.Usage()
}
