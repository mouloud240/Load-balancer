package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"
)
type Logger struct {
	file *os.File
	mu   sync.Mutex
	buf  *bufio.Writer
}

func NewLogger(port int) *Logger {
	file, err := os.OpenFile(
		fmt.Sprintf("dump/main_%d.log", port),
		os.O_APPEND|os.O_CREATE|os.O_WRONLY,
		0644,
	)
	if err != nil {
		panic(err)
	}

	return &Logger{
		file: file,
		buf:  bufio.NewWriterSize(file, 64*1024),
	}
}

func (l *Logger) Write(r *http.Request, port int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	fmt.Fprintf(
		l.buf,
		"%s %s %s\n",
		r.Method,
		r.URL.String(),
		r.RemoteAddr,
	)
}

func (l *Logger) Close() {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.buf.Flush()
	l.file.Close()
}
func handler(port int,loggger *Logger) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {
		go loggger.Write(r,port)
		surname := r.URL.Query().Get("surname")
		switch r.Method {
		case http.MethodGet:
			// GET: only use the surname query parameter
			fmt.Fprintf(w, "hello %s from port %d\n", surname, port)

		case http.MethodPost:
			// POST: use both the body name and query surname
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "failed to read body", http.StatusBadRequest)
				return
			}

			values, err := url.ParseQuery(string(body))
			if err != nil {
				http.Error(w, "invalid body", http.StatusBadRequest)
				return
			}

			name := values.Get("name")

			fmt.Fprintf(w, "hello %s %s from port %d\n", name, surname, port)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func startServer(port int) {
	loggger := NewLogger(port)
	mux := http.NewServeMux()
		mux.HandleFunc("/hello", handler(port, loggger))
mux.HandleFunc("/",func (w http.ResponseWriter, r *http.Request){

			fmt.Fprintf(w, "Wrong bozo from port %d\n", port)
	},)


	fmt.Printf("Server running on :%d\n", port)

	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), mux); err != nil {
		fmt.Printf("server %d stopped: %v\n", port, err)
	}
}

func main() {
	for port := 3000; port <= 3003; port++ {
		go startServer(port)
	}

	select {}
}
