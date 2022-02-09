# naive-http-relay

http traffic mitm

## usage

start relay:

```
$ go run . [<port number>] | tee ~/traffic.log
```

then send http requests to:

```
http://0.0.0.0:<port number>/<url>
```
