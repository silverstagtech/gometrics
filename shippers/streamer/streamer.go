// Streamer is a network stream that will send bytes in on the network via TCP or UDP.
// Buffers
// The Streamer manages 2 in memory buffers, the packet and the metrics buffer.
// Is how many measurement samples we can buffer up in before flushing.
// These are stored as [][]byte and is measured in bytes. ie you could store
// 100kb or 5mb.
// When a measurement sample comes in that will make the buffer breach its limit a
// flush is triggered.
// A flush is also triggered by a timer to make sure that measurements are always fresh.
//
// Max Packet Size
// If UDP is chosen then it will respect the max packet size to stop packets from
// being fragmented (Which would likely mean your measurements get lost).
// If it is a TCP connection it will just send when the buffer is full.
//
// Flush Interval
// The flush interval is used to keep metrics out the buffer if there is not enough
// to fill the buffer and trigger a flush.

package streamer

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/silverstagtech/gometrics/shippers"
)

// Protocol is a networking protocol that is supported by the streamer
type Protocol string

const (
	// TCP is for TCP
	TCP Protocol = "tcp"
	// UDP is for UDP
	UDP Protocol = "udp"
	// DefaultMaxPacketSize is the default packet size for UDP messages.
	// 1500bytes - 8 byte UDP header + 20 byte IP header
	DefaultMaxPacketSize = 1500 - 28
	// DefaultMaxBuffer is the default size for the in memory buffer. Default is 1mb.
	DefaultMaxBuffer = 1024 * 1024
	// DefaultTickerTime is how often the flush will happen in ms. Default is 2000ms
	DefaultTickerTime = 2000
	// DefaultConcatinator is used to join []bytes received by Ship when sending on the network
	DefaultConcatinator = byte(10)
	// DefaultReconnectionAttempts will be used when trying to reconnect after a failed write
	// After this many reconnections the data is dropped.
	DefaultReconnectionAttempts = 3
	// DefaultReconnectionWaitMS how long to wait between each attempt
	DefaultReconnectionWaitMS = 250
	// EOF End of File
	EOF = byte(0)
)

// Option is a function that will apply options to the streamer
type Option func(*Streamer)

// Streamer is used to send messages on the network as a stream of data via UDP or TCP
type Streamer struct {
	address              string
	packetSize           int
	protocol             Protocol
	connected            bool
	buffer               *buffer
	ticker               *time.Ticker
	shutdown             chan struct{}
	conn                 net.Conn
	joint                []byte
	onErrfunc            func(error)
	ratedLimitErrorFunc  func(error)
	udpSendOversize      bool
	reconnectionAttempts int
	reconnectionWaitMS   time.Duration
}

// New create a streamer applies the options and then returns a pointer to the new streamer.
func New(options ...Option) *Streamer {
	streamer := &Streamer{
		packetSize:           DefaultMaxPacketSize,
		joint:                []byte{DefaultConcatinator},
		reconnectionAttempts: DefaultReconnectionAttempts,
		reconnectionWaitMS:   time.Microsecond * DefaultReconnectionWaitMS,
		buffer:               newBuffer(),
		protocol:             UDP,
		shutdown:             make(chan struct{}, 1),
		udpSendOversize:      true,
	}

	for _, option := range options {
		option(streamer)
	}

	if streamer.ticker == nil {
		streamer.ticker = time.NewTicker(time.Millisecond * DefaultTickerTime)
	}
	return streamer
}

// Connect will connect the streamer.
func (stream *Streamer) Connect() error {
	c, err := stream.connect()
	if err != nil {
		return err
	}

	stream.conn = c
	stream.startSending()
	stream.buffer.init()
	return nil
}

func (stream *Streamer) connect() (net.Conn, error) {
	return net.Dial(string(stream.protocol), stream.address)
}

func (stream *Streamer) startSending() {
	// Start the flush trigger
	go func() {
		for {
			select {
			case <-stream.ticker.C:
				select {
				case stream.buffer.trigger <- struct{}{}:
				}
			}
		}
	}()
	// Start the correct shipper
	if stream.protocol == TCP {
		go stream.sendTCP()
		return
	}
	go stream.sendUDP()
}

func (stream *Streamer) sendTCP() {
	tryWrite := func(b [][]byte) bool {
		_, err := stream.conn.Write(bytes.Join(b, nil))
		if err != nil {
			stream.onErrfunc(err)
			return false
		}
		return true
	}

	reconnectTCP := func() bool {
		reconnected := false
		for try := 1; try <= stream.reconnectionAttempts; try++ {
			if stream.recoverTCPConnection() {
				reconnected = true
				break
			}
			time.Sleep(stream.reconnectionWaitMS)
		}
		if !reconnected {
			stream.onErrfunc(fmt.Errorf("Failed to recover the TCP connection after 3 reconnection attempts"))
		}
		return reconnected
	}

	tcpConnectionWorking := true

	for {
		// Is our TCP connection alive and kicking or do we need to try get it back.
		if !tcpConnectionWorking {
			if !reconnectTCP() {
				continue
			}
			tcpConnectionWorking = true
		}

		// We can only try write if we know we have a working TCP connection
		select {
		case data, ok := <-stream.buffer.out:
			if !ok {
				// Nothing more to come
				err := stream.conn.Close()
				if err != nil {
					stream.onErrfunc(err)
				}
				close(stream.shutdown)
				return
			}

			if !tryWrite(data) {
				// At this point the write failed and the TCP connection has likely failed.
				// Can close the connection, and try to connect again.

				// TCP connection is considered dead
				tcpConnectionWorking = false

				// Try reconnect and send the data again. If it failed again, then it should drop
				// the data as it is a bigger problem than just reconnecting.
				// Metrics are sent on best effort so this behavior fits that model.
				// This function itself will keep trying to reconnect.
				if reconnectTCP() {
					// Once connected, try write again. If it is successful, then the TCP connection
					// will get a true showing its ok. Else it gets a false which will trigger the
					// above reconnection loop.
					tcpConnectionWorking = tryWrite(data)
				}
			}
		}
	}
}

func (stream *Streamer) recoverTCPConnection() bool {
	// If we are here then we think that the connection is dead.

	// Close it and reconnect
	// Go could think its still open so close it to release the resources.
	stream.conn.Close()
	conn, err := stream.connect()
	if err != nil {
		stream.onErrfunc(err)
		return false
	}
	stream.conn = conn
	return true
}

func (stream *Streamer) sendUDP() {
	packet := newPacket(stream.packetSize)
	for {
		select {
		case data, ok := <-stream.buffer.out:
			// Nothing more to come.
			if !ok {
				err := stream.conn.Close()
				if err != nil {
					stream.onErrfunc(err)
				}
				close(stream.shutdown)
				return
			}

			// Firestream mode.
			if stream.packetSize == 0 {
				for _, bs := range data {
					if _, err := stream.conn.Write(bs); err != nil {
						stream.onErrfunc(err)
					}
				}
				continue
			}

			// Packet Buffer mode.
			for _, bs := range data {
				err := packet.add(bs)
				if err != nil {
					switch err.(type) {
					case errOverCap:
						// Will fit but buffer too full
						toSend := packet.read()
						if len(toSend) > 0 {
							if _, err := stream.conn.Write(toSend); err != nil {
								stream.onErrfunc(err)
							}
						}
						// try again, still doesn't fit?
						if err := packet.add(bs); err != nil {
							stream.onErrfunc(err)
						}
					case errTooLarge:
						// Would never fit
						if stream.udpSendOversize {
							// go for broke and send anyway. Might work, better than dropping the data.
							_, err := stream.conn.Write(bs)
							if err != nil {
								stream.onErrfunc(err)
							}
						} else {
							stream.onErrfunc(fmt.Errorf("UDP Datagram is too large and shipper is configured to drop oversize messages. Dropping message"))
						}
					}
				}
			}
			// End of the data, flush anything left in the packet buffer
			toSend := packet.read()
			if len(toSend) == 0 {
				continue
			}
			if _, err := stream.conn.Write(toSend); err != nil {
				stream.onErrfunc(err)
			}
		}
	}
}

// Ship accepts []byte and adds it to the buffer
func (stream *Streamer) Ship(b []byte) {
	select {
	case stream.buffer.in <- append(b, stream.joint...):
	default:
		stream.ratedLimitErrorFunc(fmt.Errorf("Buffer is full or closed, can't send measurement"))
	}
}

// Shutdown will close the network connections after flushing the buffers and
// close the returned channel.
func (stream *Streamer) Shutdown() chan struct{} {
	close(stream.buffer.in)
	return stream.shutdown
}

// SetProtocol is an option to set the protocol
func SetProtocol(p Protocol) func(*Streamer) {
	return func(s *Streamer) {
		s.protocol = p
	}
}

// SetMaxPacketSize is an option to set the max packet size for UDP streaming
func SetMaxPacketSize(p int) func(*Streamer) {
	return func(s *Streamer) {
		s.packetSize = p
	}
}

// SetFlushInterval is an option to set the flush interval
func SetFlushInterval(t time.Duration) func(*Streamer) {
	return func(s *Streamer) {
		s.ticker = time.NewTicker(t)
	}
}

// SetMaxBufferSize is an option to set the max buffer size
func SetMaxBufferSize(bs int) func(*Streamer) {
	return func(s *Streamer) {
		s.buffer.maxSize = bs
	}
}

// SetAddress is an option to set the address of the receiving end.
// it should be in the format of "address:port"
func SetAddress(add string) func(*Streamer) {
	return func(s *Streamer) {
		s.address = add
	}
}

// SetConcatinator is an option that tells the streamer what charactor it should
// use when joining measurements up before sending on the network.
func SetConcatinator(c string) func(*Streamer) {
	return func(s *Streamer) {
		s.joint = []byte(c)
	}
}

// SetOnError set the error function that is called when the shipper has an error.
// Consider using this for logging or signalling a termination if you feel want.
func SetOnError(errFunc func(error)) func(*Streamer) {
	return func(s *Streamer) {
		s.onErrfunc = errFunc
		s.ratedLimitErrorFunc = shippers.RateLimitedErrors(
			time.Second*3,
			3,
			"Rate limit for errors triggered, suppression future messages for 3 seconds. Error: ",
			errFunc,
		)
	}
}

// SetOnErrorRateLimited override the default ratelimited error messages.
// Used in places where errors are able to flood the system when things go bad.
func SetOnErrorRateLimited(timeLimit time.Duration, maxErrors int, suppressionMessage string, errFunc func(error)) func(*Streamer) {
	return func(s *Streamer) {
		s.ratedLimitErrorFunc = shippers.RateLimitedErrors(
			timeLimit,
			maxErrors,
			suppressionMessage,
			errFunc,
		)
	}
}

// SetSendOversize tells the streamer to send oversize udp packets if encountered.
// The default is true. The alternative is to drop packets that are too large.
// Setting to false will cause the streamer to drop the measurement.
func SetSendOversize(toggle bool) func(*Streamer) {
	return func(s *Streamer) {
		s.udpSendOversize = toggle
	}
}

// SetReconnectionAttempts sets Reconnection Attempts on the streamer. Used when connections
// make use of TCP connections. If a TCP connection breaks then the streamer will try to
// reconnect before dropping the data it was trying to write when it discovered the stream
// was broken. If this number of reconnection attempts is reached the data is dropped
// and the streamer will keep trying to connect before attempting to write more data.
// In this situation the buffers holding metrics will fill up and start dropping data
// until the connection is established again.
func SetReconnectionAttempts(attempts int) func(*Streamer) {
	return func(s *Streamer) {
		s.reconnectionAttempts = attempts
	}
}

// SetReconnectionAttemptWait sets how long to wait between attempts to reconnect.
// See SetReconnectionAttemptWait for more details
func SetReconnectionAttemptWait(milliseconds int) func(*Streamer) {
	return func(s *Streamer) {
		s.reconnectionWaitMS = time.Millisecond * time.Duration(milliseconds)
	}
}
