package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"time"

	"net/http"

	//including gorilla mux and handlers packages for HTTP routing and CORS support

	"github.com/gorilla/mux"
)

var ver string = "v1.1"

func root(w http.ResponseWriter, req *http.Request) {
	bodyBytes, err := ioutil.ReadAll(req.Body)

	if err != nil {
		fmt.Println("error reading body...")
	}

	bodyString := string(bodyBytes)

	fmt.Println(bodyString)

	fmt.Fprintf(w, "OK")
	return
}

func hello(w http.ResponseWriter, req *http.Request) {
	sender := getEnv("SENDER", "CloudAcademy")
	fmt.Fprintf(w, "Hello from: %s\n", sender)
	return
}

func cputhrash(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "OK")

	done := make(chan int)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				select {
				case <-done:
					return
				default:
				}
			}
		}()
	}

	time.Sleep(time.Second * 10)
	close(done)

	return
}

func version(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "%s\n", ver)
	return
}

func getEnv(key, fallback string) string {
	value, exists := os.LookupEnv(key)
	if !exists {
		value = fallback
	}
	return value
}

func main() {
	fmt.Println("starting hello service...")

	router := mux.NewRouter()

	//setup routes
	router.HandleFunc("/", root).Methods("POST")
	router.HandleFunc("/hello", hello).Methods("GET")
	router.HandleFunc("/cputhrash", cputhrash).Methods("GET")
	router.HandleFunc("/version", version).Methods("GET")

	http.ListenAndServe(":8080", (router))
}
