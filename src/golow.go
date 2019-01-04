package main

/**
 * GoLow
 * Lightweight HTTP Server that can Redirect Incomming Requests from a config
 *  file with MUX based rules as well as direct URL paths.
 *
 * @link https://github.com/gorilla/mux
 *
 * @author Noah Halstead <nhalstead00@gmail.com>
 * @license MIT
 */

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"reflect"
	"time"

	"github.com/gorilla/mux"
)

// Config: The Config file that Gets Loaded on Start
type Config struct {
	FinalRedirect string    `json:"defaultRedirect"`
	RedirectRules []URLRule `json:"redirects"`
}

// URLRule: Control Redirects in the Config File
type URLRule struct {
	Path            string   `json:"rule"`
	URL             string   `json:"url"`
	RedirectOptions struct{} `json:"options"`
}

func main() {
	var wait time.Duration
	flag.DurationVar(&wait, "graceful-timeout", time.Second*15, "the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m")
	flag.Parse()

	conf := Config{FinalRedirect: "https://example.com", RedirectRules: []URLRule{URLRule{Path: "/go", URL: "https://golang.org"}, URLRule{Path: "/nh", URL: "https://nhalstead.me"}}}

	r := mux.NewRouter()

	for _, v := range conf.RedirectRules {
		if v.Path != "" && v.URL != "" {

			// Path can be `/` or `/word*`
			path := v.Path
			url := v.URL
			options := v.RedirectOptions

			r.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				// Default Redirect Method, 307
				statusCode := http.StatusTemporaryRedirect

				// Loop through the given Struct and give the key and values
				fields := reflect.TypeOf(options)
				values := reflect.ValueOf(options)
				num := fields.NumField()
				for i := 0; i < num; i++ {
					field := fields.Field(i)
					value := values.Field(i)

					// Set the Header value in the Request
					// field.Name, value.String()
					if field.Name == "permanently" && value.Bool() == true {
						statusCode = http.StatusTemporaryRedirect
					}
				}

				// http.StatusTemporaryRedirect, 307
				// http.StatusMovedPermanently, 301/302
				log.Println("Redirected User Rule Based: ", url)
				http.Redirect(w, r, url, statusCode)
			}) // Close Anonymous function registration for the Method.

		}
	}

	// Default 404 Route, Redirect using Default URL
	r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("Redirected User with Default: ", conf.FinalRedirect)
		http.Redirect(w, r, conf.FinalRedirect, http.StatusTemporaryRedirect)
	})

	srv := &http.Server{
		Addr:         ":80",
		Handler:      r,
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	// Run our server in a goroutine so that it doesn't block.
	go func() {
		log.Println("Server Started")
		if err := srv.ListenAndServe(); err != nil {
			log.Println(err)
		}
	}()

	/**
	 * This section of code is from the MUX docs for a graceful shtudown.
	 * @link https://github.com/gorilla/mux#graceful-shutdown
	 */
	c := make(chan os.Signal, 1)
	// We'll accept graceful shutdowns when quit via SIGINT (Ctrl+C)
	// SIGKILL, SIGQUIT or SIGTERM (Ctrl+/) will not be caught.
	signal.Notify(c, os.Interrupt)

	// Block until we receive our signal.
	<-c

	// Create a deadline to wait for.
	ctx, cancel := context.WithTimeout(context.Background(), wait)
	defer cancel()
	// Doesn't block if no connections, but will otherwise wait
	// until the timeout deadline.
	srv.Shutdown(ctx)
	// Optionally, you could run srv.Shutdown in a goroutine and block on
	// <-ctx.Done() if your application should wait for other services
	// to finalize based on context cancellation.
	log.Println("Shutting Down.")
	os.Exit(0)
}
