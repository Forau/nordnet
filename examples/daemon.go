package main

import (
	"flag"
	"github.com/Forau/nordnet/api"
	"github.com/Forau/nordnet/api/rpc"
	"github.com/Forau/nordnet/util"
	"github.com/Forau/nordnet/util/models"

	"log"

	"io/ioutil"
	"os"

	"net"
)

var (
	pemFile = flag.String("pem", "NEXTAPI_TEST_public.pem", "PEM-file for the api")
	user    = flag.String("user", "", "Username")
	pass    = flag.String("pass", "", "Password")
	bind    = flag.String("bind", ":2008", "Address to bind to for rcp server. Example, :1234, to bind to tcp port 1234, or : to bind to random port.  127.0.0.1:2008 would make it only accessable from local host")
	// Default to test system, since this is an example code.
	baseUrl = flag.String("url", "https://api.test.nordnet.se/next", "The base URL. If you are running on prod system, set to "+api.NNBASEURL)

	pemData []byte
)

func init() {
	flag.Parse()

	file, err := os.Open(*pemFile)
	if err != nil {
		log.Fatal(err)
	}
	pemData, err = ioutil.ReadAll(file)
	if err != nil {
		log.Fatal(err)
	}

	if *user == "" || *pass == "" || len(pemData) == 0 {
		flag.PrintDefaults()
		os.Exit(3)
	}
}

func credentials() string {
	cred, err := util.GenerateCredentials([]byte(*user), []byte(*pass), pemData)
	if err != nil {
		log.Fatal(err)
	}
	return cred
}

func main() {
	var tranportFn api.TransportFn
	httpTransport := api.NewHttpTransport(*baseUrl, api.NNAPIVERSION, api.NNSERVICE, credentials)
	tranportFn = func(method, path string, params *api.Params, res interface{}) error {
		log.Printf("Calling %s, %s, %+v", method, path, params)
		err := httpTransport.Perform(method, path, params, res)
		if err != nil {
			if apie, ok := err.(api.APIError); ok && apie.Code == "NEXT_INVALID_SESSION" {
				login := &models.Login{}
				err2 := httpTransport.Perform("POST", "login", nil, login)
				if err2 == nil {
					// Retry
					return httpTransport.Perform(method, path, params, res)
				}
			}
		}
		return err
	}

	l, _ := net.Listen("tcp", *bind)
	srv := rpc.NewRpcTransportServer(l, tranportFn)
	defer srv.Close()

	c := make(chan interface{})
	log.Print(<-c)
}
