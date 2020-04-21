package main

// Mostly based on https://github.com/golang/net/blob/master/icmp/ping_test.go
// All ye beware, there be dragons below...

import (
    "time"
    "os"
    "log"
    "fmt"
    "net"
    "golang.org/x/net/icmp"
    "golang.org/x/net/ipv6"
)

const (
    // Stolen from https://godoc.org/golang.org/x/net/internal/iana,
    // can't import "internal" packages
    ProtocolICMP = 58
    //ProtocolIPv6ICMP = 58
)

// Default to listen on all IPv4 interfaces
var ListenAddr = "::"

func Ping(addr string) (*net.IPAddr, time.Duration, error) {
    // Start listening for icmp replies
    c, err := icmp.ListenPacket("ip6:ipv6-icmp", ListenAddr)
    if err != nil {
        return nil, 0, err
    }
    defer c.Close()

    // Resolve any DNS (if used) and get the real IP of the target
    dst, err := net.ResolveIPAddr("ip6", addr)
    if err != nil {
        panic(err)
        return nil, 0, err
    }

    // Make a new ICMP message
    m := icmp.Message{
        Type: ipv6.ICMPTypeEchoRequest, Code: 0,
        Body: &icmp.Echo{
            ID: os.Getpid() & 0xffff, Seq: 1, //<< uint(seq), // TODO
            Data: []byte(""),
        },
    }
    b, err := m.Marshal(nil)
    if err != nil {
        return dst, 0, err
    }

    // Send it
    start := time.Now()
    n, err := c.WriteTo(b, dst)
    if err != nil {
        return dst, 0, err
    } else if n != len(b) {
        return dst, 0, fmt.Errorf("got %v; want %v", n, len(b))
    }

    // Wait for a reply
    reply := make([]byte, 1500)
    err = c.SetReadDeadline(time.Now().Add(10 * time.Second))
    if err != nil {
        return dst, 0, err
    }
    n, peer, err := c.ReadFrom(reply)
    if err != nil {
        return dst, 0, err
    }
    duration := time.Since(start)

    // Pack it up boys, we're done here
    rm, err := icmp.ParseMessage(ProtocolICMP, reply[:n])
    if err != nil {
        return dst, 0, err
    }
    switch rm.Type {
    case ipv6.ICMPTypeEchoReply:
        return dst, duration, nil
    default:
        return dst, 0, fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
    }
}

func main() {
    p := func(addr string){
        dst, dur, err := Ping(addr)
        if err != nil {
            log.Printf("Ping %s (%s): %s\n", addr, dst, err)
            return
        }
        log.Printf("Ping %s (%s): %s\n", addr, dst, dur)
    }
    p("127.0.0.1")
    p("172.27.0.1")
    p("google.com")
    p("reddit.com")
    p("www.gp.se")

    //for {
    //    p("google.com")
    //    time.Sleep(1 * time.Second)
    //}
}