Make sure wasmcloud is running with `--allow-file-load`

Create a named config with the desired port number, and start the provider:

```
wash config put http-server port=8080
wash start provider --config http-server file://./build/shared-http-server.par.gz http-server
```

Start a hello world component

```
wash start component ghcr.io/wasmcloud/components/http-hello-world-rust:0.1.0 hello-world
```

Create a named config and link the component & http server:

```
wash config put hello_world_http hostname=hello-world.internal
wash link put http-server hello-world wasi http --interface incoming-handler --source-config hello_world_http --link-name hello-world
```

Now you can access the component via HTTP Server, using the specified hostname:

```
❯ curl -i -H'Host: hello-world.internal' localhost:8080
HTTP/1.1 200 OK
Date: Wed, 07 Aug 2024 15:15:35 GMT
Content-Length: 17
Content-Type: text/plain; charset=utf-8

Hello from Rust!
http-hello-world ❯
```
