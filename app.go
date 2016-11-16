package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/hpcloud/tail"
	"github.com/julienschmidt/httprouter"
)

var (
	port     = flag.String("port", "9000", "HTTP server port")
	basePath = flag.String("base_path", "/var/log/", "Log base path")
)

func main() {
	flag.Parse()

	router := httprouter.New()

	router.GET("/*path", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			log.Printf("[ERROR] Streaming unsupported!")
			http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")

		t, err := tail.TailFile(*basePath+ps.ByName("path"), tail.Config{Follow: true})

		if err != nil {
			log.Printf("[ERROR] TailFile : %s", err)
			http.Error(w, "File not found!", http.StatusNotFound)
			return
		}

		for line := range t.Lines {
			fmt.Fprintf(w, line.Text+"\n")
			flusher.Flush()
		}
	})

	log.Fatal(http.ListenAndServe(":"+*port, router))
	log.Printf("Application running on port %s", *port)
}
