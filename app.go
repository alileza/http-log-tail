package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"

	"github.com/julienschmidt/httprouter"
)

var (
	port     = flag.String("port", "9000", "HTTP server port")
	basePath = flag.String("base_path", "/var/log/", "Log base path")
)

func tail(path string) (chan []byte, chan error, error) {
	_, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}

	errChan := make(chan error)
	w := newWatcher()
	go func() {
		cmds := []string{"tail", "-f", path}
		fmt.Println("Executing : ", cmds)

		cmd := exec.Command(cmds[0], cmds[1:]...)

		cmd.Stdout = w
		cmd.Stderr = w

		if errno := cmd.Run(); errno != nil {
			errChan <- errno
		}
	}()

	return w.out, errChan, nil
}

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

		outChan, errChan, err := tail(*basePath + ps.ByName("path"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		for {
			select {
			case out := <-outChan:
				fmt.Fprintf(w, string(out))
				flusher.Flush()
			case err := <-errChan:
				http.Error(w, err.Error(), http.StatusInternalServerError)
				break
			}
		}
	})

	fmt.Printf("Application running on port %s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, router))
}

type watcher struct {
	out chan []byte
}

func newWatcher() *watcher {
	w := &watcher{
		out: make(chan []byte),
	}

	return w
}

func (c *watcher) Write(b []byte) (int, error) {
	c.out <- b
	return len(b), nil
}
