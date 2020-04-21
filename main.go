package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"
	"flag"
	"net"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

var DefaultListenIP4 = "0.0.0.0"

func Ping(ad string, seq int) (int, *net.IPAddr, int64, error) {
	c, err := icmp.ListenPacket("ip4:icmp", DefaultListenIP4)
	if err != nil {
        return 0, nil, 0, err
    }
	defer c.Close()

	ip, err := net.ResolveIPAddr("ip4", ad)
	if err != nil {
		panic(err)
		return 0, nil, 0, err
	}

	wm := icmp.Message{
        Type: ipv4.ICMPTypeEcho, Code: 0,
        Body: &icmp.Echo{
            ID: os.Getpid() & 0xffff, Seq: seq,
            Data: []byte(""),
        },
	}
	
	wb, err := wm.Marshal(nil)
	if err != nil {
        return 0, ip, 0, err
	}

	t := time.Now()
	n, err := c.WriteTo(wb, ip)
	if err != nil {
        return 0, ip, 0, err
    } else if n != len(wb) {
        return 0, ip, 0, fmt.Errorf("got %v; want %v", n, len(wb))
	}
	
	rb := make([]byte, 1500)
	n, peer, err := c.ReadFrom(rb)
	rm, err := icmp.ParseMessage(1, rb[:n])
	dur := time.Since(t).Milliseconds()

	switch rm.Type {
	case ipv4.ICMPTypeEchoReply:
		return n, ip, dur, err
	default:
		return 0, ip, dur, fmt.Errorf("got %+v from %v; want echo reply", rm, peer)
	}
}


func main() {
	flag.Parse()
	address := flag.Arg(0)
	if address == "" {
		fmt.Printf("Expected a hostname or ip address.\n")
		os.Exit(1)
	}
	seq := 0
	b, ip, _, err := Ping(address, seq)
	if err != nil {
		fmt.Printf("PING %s (%s): %s\n", address, ip, err)
		os.Exit(1)
	}
	fmt.Printf("PING %s (%s): %d data bytes\n", address, ip, b)
	sent := 1
	received := 1

	c := make(chan os.Signal)
    signal.Notify(c, os.Interrupt)
	go func() {
        select {
        case <-c:
			fmt.Printf("\n--- %s ping statistics ---\n", address)
			fmt.Printf("%d packets transmitted, %d packets received, %f%% packet loss\n", sent, received, float64(sent - received)*100.0/float64(received))
            os.Exit(1)
        }
    }()

	for ; ; seq++ {
		time.Sleep(1 * time.Second)
		b, ip, dur, err := Ping(address, seq)
		sent++
		if err != nil {
			fmt.Printf("%s", err)
		} else {
			received++
			fmt.Printf("%d bytes from %s: icmp_seq=%d time=%d ms\n", b, ip, seq, dur)
		}
	}
}