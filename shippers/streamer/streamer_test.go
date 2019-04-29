package streamer

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/silverstagtech/gotracer"
)

type NetServer struct {
	addr      string
	port      int
	tcpServer net.Listener
	udpServer *net.UDPConn
	protocol  Protocol
	tracer    *gotracer.Tracer
	t         *testing.T
}

func NewServer(t *testing.T, p Protocol) (*NetServer, error) {
	port := rand.Intn(29999) + 10000
	addr := "127.0.0.1"
	ok := false
	switch p {
	case TCP:
		ok = true
	case UDP:
		ok = true
	}

	if ok {
		return &NetServer{
				addr:     addr,
				port:     port,
				tracer:   gotracer.New(),
				protocol: p,
				t:        t,
			},
			nil
	}

	return nil, errors.New("Invalid protocol given")
}

// Run starts the TCP Server.
func (ns *NetServer) Run() chan struct{} {
	var err error
	srvStart := make(chan struct{}, 1)
	switch ns.protocol {
	case TCP:
		ns.tcpServer, err = net.Listen(string(ns.protocol), fmt.Sprintf("%s:%d", ns.addr, ns.port))
		if err != nil {
			ns.t.Logf("Failed to create the server. Error: %s", err)
			ns.t.FailNow()
		}

		go func() {
			close(srvStart)
			err := ns.tcpHandleConnections()
			if err != nil {
				ns.t.Logf("%s", err)
				ns.t.FailNow()
			}
		}()
	case UDP:
		udpaddr := &net.UDPAddr{Port: ns.port, IP: net.ParseIP(ns.addr)}
		ns.udpServer, err = net.ListenUDP("udp", udpaddr)
		if err != nil {
			ns.t.Logf("Failed to create the server. Error: %s", err)
			ns.t.FailNow()
		}
		go func() {
			close(srvStart)
			ns.udpHandlePackets()
		}()
	}
	return srvStart
}

func (ns *NetServer) tcpHandleConnections() (err error) {
	for {
		conn, err := ns.tcpServer.Accept()
		if err != nil || conn == nil {
			err = errors.New("could not accept connection")
			break
		}

		go ns.tcpHandleConnection(conn)
	}
	return
}

func (ns *NetServer) tcpHandleConnection(conn net.Conn) {
	defer conn.Close()

	rc := bufio.NewReader(conn)
	for {
		req, err := rc.ReadBytes(10)
		if err != nil && err != io.EOF {
			ns.t.Logf("Failed to read data on the stream. Error: %s", err)
			ns.t.Fail()
			break
		}
		if len(req) == 0 {
			break
		}

		ns.tracer.SendBytes(req)
	}
	rc.Reset(rc)
}

func (ns *NetServer) udpHandlePackets() {
	for {
		buf := make([]byte, 65508)
		n, _, err := ns.udpServer.ReadFrom(buf)
		if err != nil {
			return
		}
		if n == 0 {
			continue
		}
		ns.tracer.Send(string(buf[:n]))
	}
}

func (ns *NetServer) Messages() []string {
	m := ns.tracer.Show()
	ns.tracer.Reset()
	return m
}

// Close shuts down the TCP Server
func (ns *NetServer) Close() (err error) {
	time.Sleep(time.Millisecond * 10)
	switch ns.protocol {
	case TCP:
		return ns.tcpServer.Close()
	case UDP:
		return ns.udpServer.Close()
	}
	return nil
}

func TestTCPStream(t *testing.T) {
	srv, err := NewServer(t, TCP)
	if err != nil {
		t.Logf("Failed to create a TCP server to test. Error: %s", err)
		t.Fail()
	}
	<-srv.Run()

	tcpStreamer := New(
		SetAddress(fmt.Sprintf("%s:%d", srv.addr, srv.port)),
		SetProtocol(TCP),
		SetFlushInterval(time.Millisecond*2),
		SetOnError(func(err error) { t.Logf("Streamer Error: %s", err); t.Fail() }),
	)

	if err := tcpStreamer.Connect(); err != nil {
		t.Logf("TCP streamer failed to connect. Error: %s", err)
		t.Fail()
	}
	generate := func(size int) []byte {
		start := 48
		reset := 58
		bytecode := start
		b := make([]byte, size)
		for i := 0; i < size; i++ {
			if bytecode == reset {
				bytecode = start
			}
			b[i] = byte(bytecode)
			bytecode++
		}
		return b
	}
	wg := sync.WaitGroup{}
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 50; i++ {
				time.Sleep(time.Microsecond * 100)
				tcpStreamer.Ship(generate(250))
			}
			wg.Done()
		}()
	}

	wg.Wait()
	<-tcpStreamer.Shutdown()

	err = srv.Close()
	if err != nil {
		t.Logf("Error from closing connection. Error: %s", err)
		t.Fail()
	}
	if srv.tracer.Len() < 1 {
		t.Logf("TCP Streamer didn't get a message.")
		t.Fail()
	}
}

func TestUDPOversizeStream(t *testing.T) {
	srv, err := NewServer(t, UDP)
	if err != nil {
		t.Logf("Failed to create a TCP server to test. Error: %s", err)
		t.Fail()
	}

	<-srv.Run()

	maxpacket := 1000
	payloadsize := 3000

	udpStreamer := New(
		SetAddress(fmt.Sprintf("%s:%d", srv.addr, srv.port)),
		SetProtocol(UDP),
		SetFlushInterval(time.Millisecond*2),
		SetOnError(func(err error) { t.Logf("Streamer Error: %s", err); t.Fail() }),
		SetMaxPacketSize(maxpacket),
		SetSendOversize(true),
	)

	if err := udpStreamer.Connect(); err != nil {
		t.Logf("TCP streamer failed to connect. Error: %s", err)
		t.Fail()
	}

	payload := make([]byte, payloadsize)
	for i := 0; i < payloadsize; i++ {
		payload[i] = 'a'
	}
	udpStreamer.Ship(payload)
	<-udpStreamer.Shutdown()

	err = srv.Close()
	if err != nil {
		t.Logf("Error from closing connection. Error: %s", err)
		t.Fail()
	}

	if len(srv.tracer.Show()[0]) != payloadsize+1 {
		t.Logf("UDP Streamer didn't get correct number of bytes")
		t.Fail()
	}
	//t.Log(srv.tracer.Len())
}

func TestUDPStream(t *testing.T) {
	srv, err := NewServer(t, UDP)
	if err != nil {
		t.Logf("Failed to create a TCP server to test. Error: %s", err)
		t.Fail()
	}

	<-srv.Run()

	maxpacket := 1000
	//totalPayloadsize := 3000

	udpStreamer := New(
		SetAddress(fmt.Sprintf("%s:%d", srv.addr, srv.port)),
		SetProtocol(UDP),
		SetFlushInterval(time.Millisecond*2),
		SetOnError(func(err error) { t.Logf("Streamer Error: %s", err); t.Fail() }),
		SetMaxPacketSize(maxpacket),
	)

	if err := udpStreamer.Connect(); err != nil {
		t.Logf("TCP streamer failed to connect. Error: %s", err)
		t.Fail()
	}

	generate := func(size int) []byte {
		start := 48
		reset := 58
		bytecode := start
		b := make([]byte, size)
		for i := 0; i < size; i++ {
			if bytecode == reset {
				bytecode = start
			}
			b[i] = byte(bytecode)
			bytecode++
		}
		return b
	}
	wg := sync.WaitGroup{}
	for worker := 0; worker < 8; worker++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 50; i++ {
				time.Sleep(time.Microsecond * 100)
				udpStreamer.Ship(generate(250))
			}
			wg.Done()
		}()
	}

	wg.Wait()
	<-udpStreamer.Shutdown()

	err = srv.Close()
	if err != nil {
		t.Logf("Error from closing connection. Error: %s", err)
		t.Fail()
	}

	//t.Logf("%d", srv.tracer.Len())
	if srv.tracer.Len() < 134 {
		t.Logf("UDP Streamer didn't get enough message.")
		t.Fail()
	}
	//t.Log(srv.tracer.Show())
}

func TestUDPFireStream(t *testing.T) {
	srv, err := NewServer(t, UDP)
	if err != nil {
		t.Logf("Failed to create a TCP server to test. Error: %s", err)
		t.Fail()
	}

	<-srv.Run()

	maxpacket := 0
	//totalPayloadsize := 3000

	udpStreamer := New(
		SetAddress(fmt.Sprintf("%s:%d", srv.addr, srv.port)),
		SetProtocol(UDP),
		SetFlushInterval(time.Millisecond*2),
		SetOnError(func(err error) { t.Logf("Streamer Error: %s", err); t.Fail() }),
		SetMaxPacketSize(maxpacket),
	)

	if err := udpStreamer.Connect(); err != nil {
		t.Logf("TCP streamer failed to connect. Error: %s", err)
		t.Fail()
	}

	generate := func(size int) []byte {
		start := 48
		reset := 58
		bytecode := start
		b := make([]byte, size)
		for i := 0; i < size; i++ {
			if bytecode == reset {
				bytecode = start
			}
			b[i] = byte(bytecode)
			bytecode++
		}
		return b
	}
	wg := sync.WaitGroup{}
	for worker := 0; worker < 3; worker++ {
		wg.Add(1)
		go func() {
			for i := 0; i < 500; i++ {
				time.Sleep(time.Microsecond * 5)
				udpStreamer.Ship(generate(50))
			}
			wg.Done()
		}()
	}

	wg.Wait()
	<-udpStreamer.Shutdown()

	err = srv.Close()
	if err != nil {
		t.Logf("Error from closing connection. Error: %s", err)
		t.Fail()
	}

	//t.Logf("%d", srv.tracer.Len())
	if srv.tracer.Len() < 134 {
		t.Logf("UDP Streamer didn't get enough message.")
		t.Fail()
	}
}

func TestIntervalFlush(t *testing.T) {
	srv, err := NewServer(t, TCP)
	if err != nil {
		t.Logf("Failed to create a TCP server to test. Error: %s", err)
		t.Fail()
	}
	<-srv.Run()

	tcpStreamer := New(
		SetAddress(fmt.Sprintf("%s:%d", srv.addr, srv.port)),
		SetProtocol(TCP),
		SetFlushInterval(time.Millisecond*2),
		SetOnError(func(err error) { t.Logf("Streamer Error: %s", err); t.Fail() }),
	)

	if err := tcpStreamer.Connect(); err != nil {
		t.Logf("TCP streamer failed to connect. Error: %s", err)
		t.Fail()
	}

	tcpStreamer.Ship([]byte("Hello World 1"))
	tcpStreamer.Ship([]byte("Hello World 2"))
	tcpStreamer.Ship([]byte("Hello World 3"))
	tcpStreamer.Ship([]byte("Hello World 4"))

	time.Sleep(time.Millisecond * 10)
	if srv.tracer.Len() != 4 {
		t.Logf("TCP Streamer didn't get enough message when waiting for triggers.")
		t.Fail()
	}

	<-tcpStreamer.Shutdown()

	err = srv.Close()
	if err != nil {
		t.Logf("Error from closing connection. Error: %s", err)
		t.Fail()
	}

	t.Logf("triggered output: %s", srv.tracer.Show())
	t.Logf("triggered length: %d", srv.tracer.Len())
}

func TestErrors(t *testing.T) {
	tracing := gotracer.New()
	errFunc := func(err error) {
		tracing.SendInterface(err)
	}
	stream := New(
		SetOnError(errFunc),
		SetOnErrorRateLimited(time.Millisecond*3, 3, "Rate limit hit. Error:", errFunc),
	)

	shipJunk := func() {
		for i := 0; i < 1000; i++ {
			stream.Ship([]byte("Test metric that will fail"))
		}
	}

	// Try and ship with no collection out of the buffer
	// We will get errors stating that the buffer is full.
	// They must be rate limited.
	shipJunk()
	time.Sleep(time.Millisecond * 3)
	shipJunk()

	// We expect 9 messages after sending 2 times with a sleep in the middle.
	if tracing.Len() > 9 {
		t.Logf("Errors are flooding out. Errors traced:")
		tracing.PrintlnT(t)
		t.Fail()
	}
}
