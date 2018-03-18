package syncbus_test

import (
	"fmt"
	"sync"
	"time"

	"github.com/aryszka/syncbus"
)

type Server struct {
	resource int
	mx       sync.Mutex
	testBus  *syncbus.SyncBus
}

func (s *Server) AsyncInit() {
	go func() {
		s.mx.Lock()
		defer s.mx.Unlock()
		s.resource = 42
		s.testBus.Signal("initialized")
	}()
}

func (s *Server) Resource() int {
	s.mx.Lock()
	defer s.mx.Unlock()
	return s.resource
}

func Example() {
	s := &Server{}
	s.testBus = syncbus.New(120 * time.Millisecond)

	s.AsyncInit()
	if err := s.testBus.Wait("initialized"); err != nil {
		fmt.Println("failed:", err)
	}

	fmt.Println(s.Resource())

	// Output:
	// 42
}
