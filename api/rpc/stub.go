package rpc

import (
	"github.com/Forau/nordnet/api"
	"github.com/Forau/nordnet/util/models"
	"net"
	"net/rpc"
	//   "net/http"
	"encoding/json"
	"log"
	"reflect"
	"strings"
)

type RpcRequest struct {
	Method, Path string
	Params       *api.Params
	ResponseType string
}

type RpcResponse struct {
	Result []byte
}

func (rr *RpcResponse) SetResult(in interface{}) (err error) {
	rr.Result, err = json.Marshal(in)
	return
}

func (rr *RpcResponse) GetResult(res interface{}) (err error) {
	return json.Unmarshal(rr.Result, res)
}

func (rr *RpcResponse) NewTypeModel(name string) interface{} {
	if strings.Index(name, "*[]models.") == 0 {
		nam := string([]byte(name[10:]))
		_, sl := rr.NewTypeModelObj(nam)
		return sl
	} else {
		el, _ := rr.NewTypeModelObj(name)
		return el
	}
}

// Return single instance, and a empty slice of the type.
func (rr *RpcResponse) NewTypeModelObj(name string) (interface{}, interface{}) {
	switch name {
	case "Account":
		return &models.Account{}, &[]models.Account{}
	case "AccountInfo":
		return &models.AccountInfo{}, &[]models.AccountInfo{}
	case "ActivationCondition":
		return &models.ActivationCondition{}, &[]models.ActivationCondition{}
	case "Amount":
		return &models.Amount{}, &[]models.Amount{}
	case "CalendarDay":
		return &models.CalendarDay{}, &[]models.CalendarDay{}
	case "Country":
		return &models.Country{}, &[]models.Country{}
	case "Feed":
		return &models.Feed{}, &[]models.Feed{}
	case "Indicator":
		return &models.Indicator{}, &[]models.Indicator{}
	case "Instrument":
		return &models.Instrument{}, &[]models.Instrument{}
	case "InstrumentType":
		return &models.InstrumentType{}, &[]models.InstrumentType{}
	case "IntradayGraph":
		return &models.IntradayGraph{}, &[]models.IntradayGraph{}
	case "IntradayTick":
		return &models.IntradayTick{}, &[]models.IntradayTick{}
	case "Issuer":
		return &models.Issuer{}, &[]models.Issuer{}
	case "Ledger":
		return &models.Ledger{}, &[]models.Ledger{}
	case "LedgerInformation":
		return &models.LedgerInformation{}, &[]models.LedgerInformation{}
	case "LeverageFilter":
		return &models.LeverageFilter{}, &[]models.LeverageFilter{}
	case "List":
		return &models.List{}, &[]models.List{}
	case "LoggedInStatus":
		return &models.LoggedInStatus{}, &[]models.LoggedInStatus{}
	case "Login":
		return &models.Login{}, &[]models.Login{}
	case "Market":
		return &models.Market{}, &[]models.Market{}
	case "NewsItem":
		return &models.NewsItem{}, &[]models.NewsItem{}
	case "NewsPreview":
		return &models.NewsPreview{}, &[]models.NewsPreview{}
	case "NewsSource":
		return &models.NewsSource{}, &[]models.NewsSource{}
	case "OptionPair":
		return &models.OptionPair{}, &[]models.OptionPair{}
	case "OptionPairFilter":
		return &models.OptionPairFilter{}, &[]models.OptionPairFilter{}
	case "Order":
		return &models.Order{}, &[]models.Order{}
	case "OrderReply":
		return &models.OrderReply{}, &[]models.OrderReply{}
	case "OrderType":
		return &models.OrderType{}, &[]models.OrderType{}
	case "Position":
		return &models.Position{}, &[]models.Position{}
	case "PublicTrade":
		return &models.PublicTrade{}, &[]models.PublicTrade{}
	case "PublicTrades":
		return &models.PublicTrades{}, &[]models.PublicTrades{}
	case "RealtimeAccess":
		return &models.RealtimeAccess{}, &[]models.RealtimeAccess{}
	case "Sector":
		return &models.Sector{}, &[]models.Sector{}
	case "SystemStatus":
		return &models.SystemStatus{}, &[]models.SystemStatus{}
	case "TickSizeInterval":
		return &models.TickSizeInterval{}, &[]models.TickSizeInterval{}
	case "TicksizeTable":
		return &models.TicksizeTable{}, &[]models.TicksizeTable{}
	case "Tradable":
		return &models.Tradable{}, &[]models.Tradable{}
	case "TradableId":
		return &models.TradableId{}, &[]models.TradableId{}
	case "TradableInfo":
		return &models.TradableInfo{}, &[]models.TradableInfo{}
	case "Trade":
		return &models.Trade{}, &[]models.Trade{}
	case "UnderlyingInfo":
		return &models.UnderlyingInfo{}, &[]models.UnderlyingInfo{}
	case "Validity":
		return &models.Validity{}, &[]models.Validity{}
	}
	return nil, nil
}

func (rr *RpcResponse) TypeModelName(in interface{}) (name string) {
	typ := reflect.TypeOf(in)
	val := reflect.ValueOf(in)
	if reflect.Indirect(val).Kind() == reflect.Slice {
		name = typ.String()
	} else {
		name = typ.Elem().Name()
	}
	//	log.Print("TypeModelName: " + name)
	return name
}

func (rr *RpcResponse) CallMethod(typ string, fn func(in interface{})) {
	switch typ {
	case "*[]models.Account":
		r := []models.Account{}
		fn(&r)
		log.Print("CallMethod ", r)
	}
}

// Will make rpc-friendly functions, that when invoce deligate to the session
type RpcTransportServer struct {
	Transport api.TransportHandler
	Listener  net.Listener
	Running   bool
}

func NewRpcTransportServer(listener net.Listener, sess api.TransportHandler) *RpcTransportServer {
	srv := &RpcTransportServer{Transport: sess, Listener: listener, Running: true}
	rpcSrv := rpc.NewServer()
	rpcSrv.Register(srv)
	go srv.acceptLoop(rpcSrv)
	return srv
}

func (rss *RpcTransportServer) acceptLoop(rpcSrv *rpc.Server) {
	for rss.Running {
		conn, err := rss.Listener.Accept()
		if err != nil || conn == nil {
			log.Printf("rpc.Serve: accept error: %+v, %+v", conn, err)
		} else {
			go rpcSrv.ServeConn(conn)
		}
	}
}

func (rss *RpcTransportServer) Close() error {
	rss.Running = false
	return rss.Listener.Close()
}

func (rss *RpcTransportServer) PerformRpc(req RpcRequest, res *RpcResponse) (err error) {
	caller := func(r interface{}) {
		err = rss.Transport.Perform(req.Method, req.Path, req.Params, r)
		res.SetResult(r)
		return
	}

	_ = caller

	var resOb = res.NewTypeModel(req.ResponseType)
	err = rss.Transport.Perform(req.Method, req.Path, req.Params, resOb)
	log.Printf("Result..... %+v  %+v", resOb, err)
	res.SetResult(resOb)

	//res.CallMethod(req.ResponseType, caller)

	return
}

// Will implement Transport interface, and deligate all through rpc
type RpcTransportClient struct {
	client *rpc.Client
	addr   string
}

func NewRpcTransportClient(addr string) *RpcTransportClient {
	return &RpcTransportClient{addr: addr}
}

func (rsc *RpcTransportClient) GetClient() (*rpc.Client, error) {
	if rsc.client == nil {
		client, err := rpc.Dial("tcp", rsc.addr)
		if err != nil {
			return nil, err
		}
		rsc.client = client
	}
	return rsc.client, nil
}

func (rsc *RpcTransportClient) Close() error {
	if rsc.client != nil {
		return rsc.client.Close()
	}
	return nil
}

func (rsc *RpcTransportClient) Perform(method, path string, params *api.Params, res interface{}) (err error) {
	var rpcRes RpcResponse
	rpcReq := &RpcRequest{
		Method:       method,
		Path:         path,
		Params:       params,
		ResponseType: rpcRes.TypeModelName(res),
	}
	cli, err := rsc.GetClient()
	if err != nil {
		return err
	}

	err = cli.Call("RpcTransportServer.PerformRpc", rpcReq, &rpcRes)
	rpcRes.GetResult(res)
	return
}
