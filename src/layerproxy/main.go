package main

import (
	"github.com/codegangsta/cli"
	"os"
	"os/signal"
	"runtime"
)

var FlagListen = cli.StringFlag{Name: "listen", Value: ":8009", EnvVar: "LAYERPROXY_LISTEN", Usage: "The bind string e.g. :8009"}
var FlagAWSRegion = cli.StringFlag{Name: "aws-region", Value: "us-east-1", EnvVar: "LAYERPROXY_AWS_REGION", Usage: "The aws region"}
var FlagOnDne = cli.StringFlag{Name: "on-dne", Value: "", EnvVar: "LAYERPROXY_ON_DNE", Usage: "Key to use if file does not exist."}
var FlagMemCacheBytes = cli.IntFlag{Name: "cache-size", Value: int(1E9), EnvVar: "LAYERPROXY_CACHE_BYTES", Usage: "How big (bytes) to make the in memory cache"}
var FlagWatchDelay = cli.IntFlag{Name: "watch", Value: 0, EnvVar: "LAYERPROXY_WATHC_DELAY", Usage: "Seconds to check for new files"}
var FlagPrefetchPrefixes = cli.StringSliceFlag{Name: "prefetch", Value: &cli.StringSlice{}, EnvVar: "LAYERPROXY_PREFETCH", Usage: "A list of key prefixes to prefetch. e.g 201501_"}

func main() {
	if os.Getenv("GO_DEBUG") != "" {
		go func() {
			sigs := make(chan os.Signal, 1)
			signal.Notify(sigs, os.Interrupt, os.Kill)
			<-sigs
			buf := make([]byte, 1<<20)
			runtime.Stack(buf, true)
			println(string(buf))
			os.Exit(1)
		}()
	}

	app := cli.NewApp()
	app.Name = "layerproxy"
	app.Usage = "kill it"
	app.Commands = []cli.Command{
		{
			Name:    "run",
			Aliases: []string{"r"},
			Usage:   "Run the proxy",
			Flags:   []cli.Flag{FlagListen, FlagAWSRegion, FlagOnDne, FlagMemCacheBytes, FlagWatchDelay, FlagPrefetchPrefixes},
			Action: func(c *cli.Context) {
				bucket := NewS3Store(c.String(FlagAWSRegion.Name))
				fscache := NewFSCache()
				memcache := NewCache(c.Int(FlagMemCacheBytes.Name))
				manager := NewManager(c.String(FlagOnDne.Name))
				manager.AddLayer("S3", bucket, false)
				manager.AddLayer("FS", fscache, true)
				manager.AddLayer("Memory", memcache, true)
				manager.Run(c.String(FlagListen.Name), c.StringSlice(FlagPrefetchPrefixes.Name), c.Int(FlagWatchDelay.Name))
			},
		},
	}

	app.Run(os.Args)

}
