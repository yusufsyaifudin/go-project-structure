# Golang Project Structure

## Features

* [x] Config from environment variable and .env
* [x] OpenTelemetry
  * [x] Tracing
  * [ ] Metric 
  * [ ] Logging - Not supported by Go client. Instead, we write our own Logging that support context, so you can trace the log and group them by tracer id.
* [x] Kubernetes YAML file
* [x] Prometheus /metrics endpoint
* [ ] Statsd metric

## Setup

Clone this repository, then find and replace string `github.com/yusufsyaifudin/go-project-structure` to your new package name.

Replace all `K8S_SERVICE_NAME` inside `deployment/k8s/*` directory to your service name.
For example, to `user-authentication-server` if your application name is that.



## Code Design Consideration

### Why using `net/http` middleware in `server/main.go` instead of using Echo middleware?

This project structure try to not forcing you to use Echo as router framework, instead, it just an example.
If you want to use another routing framework, then you only need to change the `transport/restapi/http.go`
and the existing router (Logging and Tracing) will still be available out of the box for you as long as you 
implement the interface `http.Handler` in struct `restapi.HTTP`.





