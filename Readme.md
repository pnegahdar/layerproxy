## layerproxy: A layered cache proxy

A library and cli to cache requests to s3 via cache layers. The default layers arguments

- S3 Api
- Local FS cache
- Local in memory cache

If `--watch` is enabled a background loop monitors s3 for changes and pushes them up to the local cache layers  

#### Installation:

Grab the right precompiled bin from github releases and put it in your path. Don't forget to `chmod +x` the bin.

OSX:

    curl -SL https://github.com/pnegahdar/layerproxy/releases/download/0.1.0/layerproxy_0.1.0_darwin_amd64.tar.gz \
        | tar -xzC /usr/local/bin --strip 1 && chmod +x /usr/local/bin/layerproxy

Nix:

    curl -SL https://github.com/pnegahdar/layerproxy/releases/download/0.1.0/layerproxy_0.1.0_linux_amd64.tar.gz \
        | tar -xzC /usr/local/bin --strip 1 && chmod +x /usr/local/bin/layerproxy


#### CLI Usage:

    NAME:
       layerproxy run - Run the proxy

    USAGE:
       layerproxy run [command options] [arguments...]

    OPTIONS:
       --listen ":8009"					The bind string e.g. :8009 [$LAYERPROXY_LISTEN]
       --aws-region "us-east-1"				The aws region [$LAYERPROXY_AWS_REGION]
       --on-dne 						Key to use if file does not exist. [$LAYERPROXY_ON_DNE]
       --cache-size "1000000000"				How big (bytes) to make the in memory cache [$LAYERPROXY_CACHE_BYTES]
       --watch "0"						Seconds to check for new files [$LAYERPROXY_WATHC_DELAY]
       --prefetch [--prefetch option --prefetch option]	A list of key prefixes to prefetch. e.g 201501_ [$LAYERPROXY_PREFETCH]

Sample run command:

    AWS_ACCESS_KEY_ID=<MY_KEY> AWS_SECRET_KEY=<MY_SECRET> layerproxy run --watch 10 --on-dne my_bucket/404.html --listen :8009

test it out:

    curl localhost:8009/my_bucket/my_key


#### Lib usage (will eventually be documented)


    bucket := stores.NewS3Store(c.String(FlagAWSRegion.Name))
    fscache := stores.NewFSCache()
    memcache := stores.NewCache(c.Int(FlagMemCacheBytes.Name))
    manager := stores.NewManager(c.String(FlagOnDne.Name))
    manager.AddLayer("S3", bucket, false)
    manager.AddLayer("FS", fscache, true)
    manager.AddLayer("Memory", memcache, true)
    manager.Run(c.String(FlagListen.Name), c.StringSlice(FlagPrefetchPrefixes.Name), c.Int(FlagWatchDelay.Name))
