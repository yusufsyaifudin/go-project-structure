# Golang Project Structure

## Background

When we want to create new project in Golang there is always "opinionated project structure style".

This "go-project-structure" is another "opinionated" project structure boiler plate in Golang. 
But, instead of forcing you to use and implements set of contract (interface), we need leave the business logic implementation up to you.
You can put your business logic in `internal/` or place it anywhere, we don't force you to put it somewhere.

Different from the another framework, all code in "go-project-structure" will become your own code to manage once you `git clone` this repository.
The thing you need **to worry** is if the 3rd party library in `go.mod` is update their version, you need to update and upgrade it manually: 
and if some API of the 3rd party library has breaking changes, then you need to update it on your own.

What we want to achieve is, when you setup new project you will always get the Logging, Tracing and Metric out of the box, 
without writing the middlewares over again. 

* Logging is component that MUST EXISTS in every system, but sometimes developer think that printing some message is enough to be called as "logging".
  In this repo, we try to force you to use JSON log, because it easy to parse when we need some specific log filter, for example: filtering by message or `trace_id`.
* Metric is something that sometimes not added until it needed. But, some basic Metric should exist in the application: Golang runtime stat and latency of the request.
  We added some metrics backend, so you only need to pick where to push (or pull) your metrics.
* Tracing is rarely be considered in new system, it usually comes later. But, by adding OpenTelemetry in this system you are adviced to always write the 
  tracing Span and propagate to next function call using context. If your company or team doesn't have enough resource to deploy tracing infrastructure, then leave it using `NOOP` 
  or set it to `STDOUT` so it will be printed as log. But, when your company's budget ready, just enable it using `JAEGER` or `OTLP` exporter.


We add [`docker-compose.yaml`](/deployment/development/docker-compose.yml) for you to setup Prometheus, Grafana (for metric) and OpenTelemetry + Jaeger (for tracing)
if you want to run it locally and then see the visualization of your metric and tracing.

We skip logging infrastructure in Docker Compose example because it depends on your deployment: you can pipe it to any log system from stdout.
We encourage you to write log to `stdout` since logging should never break your system, but writing to file will add I/O process in your system.

## Features

* [x] Config from environment variable and .env
* [x] OpenTelemetry
  * [x] Tracing
  * [ ] Metric 
  * [ ] Logging - Not supported by Go client. Instead, we write our own Logging that support context, so you can trace the log and group them by tracer id.
* [x] Kubernetes YAML file
* [x] Prometheus /metrics endpoint
* [x] Logging with context. We can track and grouping logs per `trace_id` and `span_id`. For example, for typical API you can get specific logs only by filtering `trace_id`.
* [ ] Statsd metric

## Setup

Clone this repository, then find and replace string `github.com/yusufsyaifudin/go-project-structure` to your new package name.

Replace all `K8S_SERVICE_NAME` inside `deployment/k8s/*` directory to your service name.
For example, to `user-authentication-server` if your application name is that.

### Show me bash script

Before you run this Bash script, ensure you already installed:

* `git`
* `sed`

```shell
# for example you want to put your new project to "Project" directory
cd ~/Project 

# you want to name your new application as "my-project"
git clone git@github.com:yusufsyaifudin/go-project-structure.git my-project
cd my-project

# you want to rename the package to "github.com/your-username/my-project"
# https://stackoverflow.com/a/29687544/5489910
find . -type f -print |
while IFS= read -r file
do
   sed -i -- 's,github.com/yusufsyaifudin/go-project-structure,github.com/your-username/my-project,g' $file
done

# delete git file and create your own git
rm -rf .git
git init && git add . && git commit -m "first init"

# unit test should run normally
make test
```

## Code Design Consideration

### Why using `net/http` middleware in `server/main.go` instead of using Echo middleware?

This project structure try to not forcing you to use Echo as router framework, instead, it just an example.
If you want to use another routing framework, then you only need to change the `transport/restapi/http.go`
and the existing router (Logging and Tracing) will still be available out of the box for you as long as you 
implement the interface `http.Handler` in struct `restapi.HTTP`.
