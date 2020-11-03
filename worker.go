package httpproxy

import (
	"io"
	"log"
	"sync"

	"github.com/fatih/color"
)

// Worker is a single https worker, it does everything up to the piping
type Worker struct {
	Conn   io.ReadWriteCloser
	Buffer []byte
}

// Pool is a pool of reusable https workers
type Pool struct {
	Workers []*Worker
	Mux     sync.Mutex
}

// NewPool creates and returns a new Pool of socks5 workers
func NewPool() *Pool {
	return &Pool{}
}

// GetWorker returns a socks5 worker from the workerpool
func (p *Pool) GetWorker() *Worker {
	p.Mux.Lock()
	defer p.Mux.Unlock()

	var w *Worker
	if len(p.Workers) > 0 {
		w, p.Workers = p.Workers[0], p.Workers[1:]
		log.Println(color.GreenString("Existing http worker"))
		return w
	}
	log.Println(color.RedString("New worker"))
	return &Worker{
		Conn:   nil,
		Buffer: make([]byte, 2048),
	}
}
