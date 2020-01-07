# Streamer

Streamer is a network streamer that sends bytes over TCP and UDP.

Streamer is build using the same ethos as gometrics itself. It trades speed for use ability. In 99% of use cases it will do the job.

It plenty fast enough...

## Available options

* TCP Mode
* UDP Buffered
* UDP FireStream 🔥

### TCP Mode

TCP mode will flush all the bytes out the buffer and send it in a stream. Slower than UDP but reliable.

### UDP Buffered

UDP buffered mode tries its hardest to respect packet sizes. However if a message is too large for a packet then it tried to send it anyway as the other choice is to drop it.

The default packet size is 1472 bytes
Default LAN MTU 1500 bytes - 8 byte UDP header + 20 byte IP header

If you increase it remember that the 28bytes will still be added by the OS so make sure you factor it in.

### UDP Fire Stream 🔥

Setting the max packet size to 0 will enable FireStream mode. Basically the message will get shipped as the come out of the shipper buffer by triggers from the interval flusher or by simply filling it up.

It gains speed by disabling the packet buffer manager. So your receiver needs to be fast enough to handle your messages.
Generally this will be one measurement per message.

## Using Streamer

```go
  stream := New(
    SetAddress(fmt.Sprintf("%s:%d", srv.addr, srv.port)),
    SetProtocol(UDP),
    SetFlushInterval(time.Millisecond * 1500),
    SetOnError(func(err error) { fmt.Println(err)),
  )
```
