// A simple DNS stress tool to build robust DNS servers
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

func queryStress(rs *net.Resolver, host string, countChannel chan struct{}, wg *sync.WaitGroup, showResult bool) {
	countChannel <- struct{}{}
	wg.Add(1)
	go func(h string) {
		addrs, err := rs.LookupHost(context.Background(), h)
		if err != nil {
			fmt.Println(err)
		} else if showResult {
			fmt.Println("IPs:", addrs)
		}
		<-countChannel
		wg.Done()
	}(host)
}

func main() {
	var (
		Network     string
		Resolver    string
		InFile      string
		Timeout     time.Duration
		Concurrency int
		Total       int
		Verbose     bool
	)

	flag.Usage = func() {
		fmt.Println("A simple DNS stress tool - v0.1.0\n\nUsage: dnsstress [options] [host]\n\nOptions:")
		flag.PrintDefaults()
	}
	flag.StringVar(&Network, "p", "udp", "network type")
	flag.StringVar(&Resolver, "s", "127.0.0.1:53", "DNS server (ip:port)")
	flag.StringVar(&InFile, "i", "", "host list file, discards host")
	flag.DurationVar(&Timeout, "t", 5*time.Second, "dial timeout")
	flag.IntVar(&Concurrency, "c", 1, "concurrency")
	flag.IntVar(&Total, "n", 1, "total queries, ignored in list file mode")
	flag.BoolVar(&Verbose, "v", false, "verbose")
	flag.Parse()

	rs := &net.Resolver{}
	if len(Resolver) > 0 {
		rs.PreferGo = true // required
		rs.Dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
			d := &net.Dialer{Timeout: Timeout}
			return d.DialContext(ctx, Network, Resolver)
		}
	}

	countChannel := make(chan struct{}, Concurrency)
	var wg sync.WaitGroup

	if InFile != "" {
		in, err := os.ReadFile(InFile)
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
		lines := strings.Split(string(in), "\n")
		for _, line := range lines {
			dm := strings.Fields(line)
			if len(dm) == 0 {
				continue
			}
			if dm[0][0] == '#' {
				continue
			}
			queryStress(rs, dm[0], countChannel, &wg, Verbose)
		}
	} else if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	} else {
		for i := 0; i < Total; i++ {
			queryStress(rs, flag.Arg(0), countChannel, &wg, Verbose)
		}
	}

	wg.Wait()
}
