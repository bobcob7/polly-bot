package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/bobcob7/polly/pkg/transmission"
)

var tranmissionEndpoint string

func init() {
	flag.StringVar(&tranmissionEndpoint, "endpoint", "https://transmission.bobcob7.com", "URL where transmission RPC can be reached")
}

func main() {
	flag.Parse()
	if len(flag.Args()) != 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] magnet-link\n", flag.Args()[0])
		os.Exit(1)
	}
	ctx, done := context.WithCancel(context.Background())
	defer done()
	tr, err := transmission.New(ctx, tranmissionEndpoint)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to connect to tranmission RPC server", err.Error())
		return
	}

	err = tr.AddLink(ctx, flag.Args()[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to add link:", err.Error())
		return
	}
}
