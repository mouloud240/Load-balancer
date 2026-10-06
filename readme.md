# Load Balancer

A simple L7 HTTP load balancer in Go using round-robin. Educational project, not for production.

## Current state

- Round-robin request distribution across configured upstreams.
- YAML file configuration (`config.yaml`, see `config.example.yaml`).
- Basic circuit breaking per upstream (closed / open / half-open).
- Docker image + compose setup (`Dockerfile`, `docker-compose.yml`).

## Setup

Prerequisites: Go 1.26+ (or Docker).

```sh
git clone https://github.com/mouloud240/Load-balancer.git
cd Load-balancer
```

Copy the example config and adjust upstreams:

```sh
cp config.example.yaml config.yaml
```

Run the load balancer:

```sh
make run
```

In another terminal, start the dummy backend servers for testing:

```sh
make run_dummy_servers
```

Or run everything with Docker:

```sh
docker compose up --build
```

The balancer listens on `:8080`. Optional helpers: `make load_test` (autocannon run) and `make view_distribution` (request counts per backend).

## Roadmap

Tracked as [GitHub issues](https://github.com/mouloud240/Load-balancer/issues).
