package rpc_test

import (
	"errors"
	"github.com/Forau/nordnet/api"
	"github.com/Forau/nordnet/api/rpc"

	"github.com/Forau/nordnet/util/models"
	"github.com/stretchr/testify/assert"

	"testing"

	"encoding/json"
	"fmt"
	"net"
	"strings"
)

func jsonEqual(t *testing.T, exp interface{}, res interface{}) {
	exps, err1 := json.Marshal(exp)
	if err1 != nil {
		t.Errorf("Unable to marshal: %T -> %+v", err1, err1)
	} else {
		ress, err2 := json.Marshal(res)
		if err2 != nil {
			t.Errorf("Unable to marshal: %T -> %+v", err2, err2)
		} else {
			assert.Equal(t, exps, ress)
		}
		t.Logf("Compare %+v with %+v", string(exps), string(ress))
	}
}

// Simple test of login function
func TestRpcCallLoginError(t *testing.T) {
	var transp api.TransportFn
	transp = func(method, path string, params *api.Params, res interface{}) error {
		if _, ok := res.(*models.Login); ok {
			return errors.New("Got the error")
		} else {
			t.Error("Was not called with Login type: ", res)
			return nil
		}
	}

	l, _ := net.Listen("tcp", ":")

	srv := rpc.NewRpcTransportServer(l, transp)
	defer srv.Close()

	t.Log(srv)
	cli := rpc.NewRpcTransportClient(srv.Listener.Addr().String())
	defer cli.Close()
	t.Log(cli)

	apic := &api.APIClient{
		Transport: cli,
	}

	_, err := apic.Login()
	assert.EqualError(t, err, "Got the error")
}

// Simple test of login function
func TestRpcCallLoginHappy(t *testing.T) {
	expected := &models.Login{Environment: "TEST", SessionKey: "TestSession"}
	var transp api.TransportFn
	transp = func(method, path string, params *api.Params, res interface{}) error {
		if rPtr, ok := res.(*models.Login); ok {
			*rPtr = *expected
		} else {
			t.Error("Was not called with Login type: ", res)
		}
		return nil
	}

	l, _ := net.Listen("tcp", ":")
	srv := rpc.NewRpcTransportServer(l, transp)
	defer srv.Close()
	t.Log(srv)
	cli := rpc.NewRpcTransportClient(srv.Listener.Addr().String())
	defer cli.Close()
	t.Log(cli)

	apic := &api.APIClient{
		Transport: cli,
	}
	res, err := apic.Login()

	assert.Nil(t, err)
	jsonEqual(t, expected, res)
}

// Simple test of account function
func TestRpcCallAccountsHappy(t *testing.T) {
	expected := []models.Account{
		models.Account{Accno: 123, Type: "Big money"},
		models.Account{Accno: 321, Blocked: true, BlockedReason: "Personal"},
	}

	var transp api.TransportFn
	transp = func(method, path string, params *api.Params, res interface{}) (err error) {
		t.Logf("Got %T -> %+v", res, res)
		body := `[{"accno": 123, "type": "Big money"},{"accno": 321, "blocked": true, "blocked_reason": "Personal"}]`
		err = json.NewDecoder(strings.NewReader(body)).Decode(&res)
		t.Logf("Res is now %+v", res)
		return
	}

	l, _ := net.Listen("tcp", ":")
	srv := rpc.NewRpcTransportServer(l, transp)
	defer srv.Close()
	t.Log(srv)
	cli := rpc.NewRpcTransportClient(srv.Listener.Addr().String())
	defer cli.Close()
	t.Log(cli)

	apic := &api.APIClient{
		Transport: cli,
	}
	res, err := apic.Accounts()

	assert.Nil(t, err)
	jsonEqual(t, expected, res)
}

// A transport handler that we can tell to return a byte array, or an error
type TestTransportHandler struct {
	Reply []byte
	Error error
}

// Implement TransportHandler
func (tth *TestTransportHandler) Perform(method, path string, params *api.Params, res interface{}) (err error) {
	if tth.Error != nil {
		err = tth.Error
	} else {
		err = json.Unmarshal(tth.Reply, &res)
	}
	return
}

func (tth *TestTransportHandler) SetResultJson(r []byte, e error) {
	tth.Reply = r
	tth.Error = e
}

// Batch test of calls with 'happy' results
func TestHappyCases(t *testing.T) {
	var testParams api.Params
	testParams = make(api.Params)
	testParams["test"] = "Yes, we are testing"

	testData := []struct {
		Exp  interface{}
		Call func(api *api.APIClient) (interface{}, error)
	}{
		{
			Exp:  &models.SystemStatus{Timestamp: 0xdeadbeef, Message: "Stake"},
			Call: func(api *api.APIClient) (interface{}, error) { return api.SystemStatus() },
		},
		{
			Exp:  &models.Login{Environment: "TEST2", SessionKey: "TestSession2"},
			Call: func(api *api.APIClient) (interface{}, error) { return api.Login() },
		},
		{
			Exp:  &models.LoggedInStatus{LoggedIn: true},
			Call: func(api *api.APIClient) (interface{}, error) { return api.Logout() },
		},
		{
			Exp:  &models.LoggedInStatus{LoggedIn: true},
			Call: func(api *api.APIClient) (interface{}, error) { return api.Touch() },
		},
		{
			Exp:  []models.Account{models.Account{Accno: 123, Type: "Big money"}},
			Call: func(api *api.APIClient) (interface{}, error) { return api.Accounts() },
		},
		{
			Exp:  &models.AccountInfo{AccountCurrency: "HotAir", AccountCredit: models.Amount{Value: 3.141592, Currency: "PI"}},
			Call: func(api *api.APIClient) (interface{}, error) { return api.Account(1234) },
		},
		{Exp: &models.OrderReply{}, Call: func(api *api.APIClient) (interface{}, error) { return api.DeleteOrder(1234, 42) }},
		{Exp: []models.Position{}, Call: func(api *api.APIClient) (interface{}, error) { return api.AccountPositions(1234) }},
		{Exp: []models.Trade{}, Call: func(api *api.APIClient) (interface{}, error) { return api.AccountTrades(1234, &testParams) }},
		{Exp: []models.Country{}, Call: func(api *api.APIClient) (interface{}, error) { return api.Countries() }},
		{Exp: []models.Country{}, Call: func(api *api.APIClient) (interface{}, error) { return api.LookupCountries("panama") }},
		{Exp: []models.Indicator{}, Call: func(api *api.APIClient) (interface{}, error) { return api.Indicators() }},
		{Exp: []models.Indicator{}, Call: func(api *api.APIClient) (interface{}, error) { return api.LookupIndicators("indi") }},
		{Exp: []models.Instrument{}, Call: func(api *api.APIClient) (interface{}, error) { return api.SearchInstruments(&testParams) }},
		{Exp: []models.Instrument{}, Call: func(api *api.APIClient) (interface{}, error) { return api.Instruments("1,2,3") }},
		{Exp: []models.Instrument{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentLeverages(42, &testParams) }},
		{Exp: &models.LeverageFilter{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentLeverageFilters(42, &testParams) }},
		{Exp: []models.OptionPair{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentOptionPairs(42, &testParams) }},
		{Exp: &models.OptionPairFilter{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentOptionPairFilters(42, &testParams) }},
		{Exp: []models.Instrument{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentLookup("bear", "omx") }},
		{Exp: []models.Sector{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentSectors(&testParams) }},
		{Exp: []models.Sector{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentSector("banking") }},
		{Exp: []models.InstrumentType{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentTypes() }},
		{Exp: []models.InstrumentType{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentType("good") }},
		{Exp: []models.Instrument{}, Call: func(api *api.APIClient) (interface{}, error) { return api.InstrumentUnderlyings("good", "pi") }},
		{Exp: []models.List{}, Call: func(api *api.APIClient) (interface{}, error) { return api.Lists() }},
		{Exp: []models.Instrument{}, Call: func(api *api.APIClient) (interface{}, error) { return api.List(42) }},
		{Exp: []models.Market{}, Call: func(api *api.APIClient) (interface{}, error) { return api.Markets() }},
		{Exp: []models.Market{}, Call: func(api *api.APIClient) (interface{}, error) { return api.Market("magic") }},
		{Exp: []models.NewsPreview{}, Call: func(api *api.APIClient) (interface{}, error) { return api.SearchNews(&testParams) }},

		{Exp: []models.NewsItem{}, Call: func(api *api.APIClient) (interface{}, error) { return api.News("1,2,13") }},
		{Exp: []models.NewsSource{}, Call: func(api *api.APIClient) (interface{}, error) { return api.NewsSources() }},
		{Exp: []models.RealtimeAccess{}, Call: func(api *api.APIClient) (interface{}, error) { return api.RealtimeAccess() }},
		{Exp: []models.TicksizeTable{}, Call: func(api *api.APIClient) (interface{}, error) { return api.TickSizes() }},
		{Exp: []models.TicksizeTable{}, Call: func(api *api.APIClient) (interface{}, error) { return api.TickSize("2") }},
		{Exp: []models.TradableInfo{}, Call: func(api *api.APIClient) (interface{}, error) { return api.TradableInfo("42") }},
		{Exp: []models.IntradayGraph{}, Call: func(api *api.APIClient) (interface{}, error) { return api.TradableIntraday("42") }},
		{Exp: []models.PublicTrades{}, Call: func(api *api.APIClient) (interface{}, error) { return api.TradableTrades("42") }},
	}

	transh := &TestTransportHandler{}

	l, _ := net.Listen("tcp", ":")
	srv := rpc.NewRpcTransportServer(l, transh)
	defer srv.Close()
	t.Log(srv)

	cli := rpc.NewRpcTransportClient(srv.Listener.Addr().String())
	defer cli.Close()
	t.Log(cli)

	apic := &api.APIClient{
		Transport: cli,
	}

	for idx, td := range testData {
		// First test happy case
		transh.SetResultJson(json.Marshal(td.Exp))
		res, err := td.Call(apic)
		t.Logf("Test data[%d] %+v yields %+v, %+v", idx, td, res, err)
		if err != nil {
			t.Error(err)
		}
		jsonEqual(t, td.Exp, res)

		// And test error case
		transh.Error = fmt.Errorf("Bad bad error! Code: %d ", idx)
		transh.Reply = nil
		res, err = td.Call(apic)
		t.Logf("Test data[%d] for errors: %+v yields %+v, %+v", idx, td, res, err)
		assert.EqualError(t, err, fmt.Sprintf("Bad bad error! Code: %d ", idx))
	}
}
