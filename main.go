package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
)

//  FIRST WE WILL BUILD TOR ITSELF AND SET UP N INSTANCES
//  THEN WE NEED TO FIGURE OUT HOW TO CANCEL THEM
const (
	NUM_INSTANCES = 3
)

func KillAllProxies() {
	for i := 0; i < NUM_INSTANCES; i++ {
		cmd := exec.Command(
			"docker",
			"stop",
			fmt.Sprintf("socksprox_%v", i),
		)

		err := cmd.Run()

		if err != nil {
			fmt.Println("Error stopping tor")
			os.Exit(1)
		}

		cmd2 := exec.Command(
			"docker",
			"rm",
			fmt.Sprintf("socksprox_%v", i),
		)

		err2 := cmd2.Run()

		if err2 != nil {
			fmt.Println("Error removing tor")
			os.Exit(1)
		}
	}
}

func main() {
	// WHEN YOU RUN THIS BINARY IT MUST SET UP THE INSTANCES
	// WE DONT WANNA FIGURE OUT ALL THE THINGS SO WE WILL DO THIS VIA CMD?
	// BUILD IMAGE
	cmd := exec.Command(
		"docker",
		"build",
		"-t",
		"socksprox",
		"./socksprox",
	)

	path, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting pwd")
		log.Println(err)
	}

	cmd.Dir = path

	err2 := cmd.Run()

	if err2 != nil {
		fmt.Println("Error building tor")
		os.Exit(1)
	}

	// CREATE INSTANCES
	for i := 0; i < NUM_INSTANCES; i++ {
		cmd := exec.Command(
			"docker",
			"run",
			"-d",
			"-p",
			fmt.Sprintf("500%v:9050", i),
			"--name",
			fmt.Sprintf("socksprox_%v", i),
			"socksprox",
		)

		err := cmd.Run()

		if err != nil {
			fmt.Println("Error running tor")
			os.Exit(1)
		}
	}

	// EXPOSE A PROXY THAT WHEN YOU HIT IT ROTATES THROUGH
	// so we have ten instances of tor at 127.0.0.1:5000-9
	// nowe we need te expose an http proxy that uses it on the other end
	http.HandleFunc("/", proxyAll)

	fmt.Printf("Starting server for testing HTTP POST...\n")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal(err)
	}
	// THE LIST OF PROXIES ON EVERY REQUEST AND ALSO PREPARES
	// THE NEXT PRODXY AND REPLACES IT WHEN ITS TIME
}

func proxyAll(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		proxyUrl, err1 := url.Parse("http://127.0.0.1:5000")

		if err1 != nil {
			fmt.Printf("Unhandled error %v \n", err1)
			log.Fatalln(err1)
		}

		myClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyUrl),
			},
		}

		resp, err := myClient.(r.URL.String())

		if err != nil {
			fmt.Printf("Unhandled error %v \n", err)
			log.Fatalln(err)
		}

		fmt.Printf("Response %v\n", resp)
		w.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
		var body []byte
		resp.Body.Read(body)

		w.Write(body)
	}

}
