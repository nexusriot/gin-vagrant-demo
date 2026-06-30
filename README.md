# gin-vagrant-demo

A small demo of a [Gin](https://github.com/gin-gonic/gin) HTTP service built into a
Docker image and deployed inside an Ubuntu 24.04 VirtualBox VM provisioned with Ansible.

The VM is brought up with Vagrant, provisioned via `ansible_local` (installs Go and
Docker CE), and the service runs as a container. A single `vm.sh` driver wraps the
common lifecycle commands.

## Requirements

On the host:

- [Vagrant](https://www.vagrantup.com/)
- [VirtualBox](https://www.virtualbox.org/)
- The `cloud-image/ubuntu-24.04` box (Vagrant fetches it on first `up`)

Ansible, Go, and Docker are installed **inside** the VM during provisioning — you do not
need them on the host.

For local development (outside the VM) you only need Go 1.22+.

## Quick start

```sh
./vm.sh start     # vagrant up + provision (installs Go + Docker in the VM)
./vm.sh build     # build the Docker image and run the container inside the VM
./vm.sh test      # go vet + go test, then curl the /health endpoint
```

The service is published on host port `8080` (forwarded from the guest):

```sh
curl http://localhost:8080/
curl http://localhost:8080/health
curl http://localhost:8080/version
```

## `vm.sh` commands

| Command          | Description                                              |
| ---------------- | -------------------------------------------------------- |
| `start`          | `vagrant up --provider=virtualbox`                       |
| `stop`           | `vagrant halt`                                           |
| `destroy`        | `vagrant destroy -f`                                     |
| `build`          | Build the image and (re)run the container inside the VM  |
| `test`           | Run `go vet` + `go test` and health-check the endpoint   |
| `svc-start`      | `docker start gin-demo`                                  |
| `svc-stop`       | `docker stop gin-demo`                                   |
| `svc-logs`       | Follow the container logs                                |
| `version`        | Print the running service's build info (`/version`)      |

## HTTP endpoints

| Method | Path        | Response                                                        |
| ------ | ----------- | -------------------------------------------------------------- |
| `GET`  | `/`         | `{"message":"hello from gin in vagrant VM"}`                   |
| `GET`  | `/health`   | `{"status":"ok"}`                                              |
| `GET`  | `/version`  | `{"version":"…","commit":"…","build_time":"…"}`                |
| _any_  | _other_     | `404` `{"error":"not found","path":"…"}`                       |

The server listens on `0.0.0.0:$PORT` (default `8080`) and shuts down gracefully on
`SIGINT`/`SIGTERM`, draining in-flight requests with a 10s timeout.

## Local development

```sh
# Run directly
go run ./cmd/gin-demo

# Run tests
go test ./...

# Build with version metadata injected via ldflags
go build \
  -ldflags="-X main.version=1.2.3 -X main.commit=$(git rev-parse --short HEAD) -X main.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
  -o gin-server ./cmd/gin-demo
```

Override the listen port with the `PORT` environment variable:

```sh
PORT=9090 go run ./cmd/gin-demo
```

## Docker

The image is a multi-stage build: dependencies are downloaded in a cacheable layer
(`go.mod` + `go.sum` → `go mod download`), the binary is built statically with version
ldflags, and the runtime stage is
[`distroless/static`](https://github.com/GoogleContainerTools/distroless) running as a
non-root user (no shell, no package manager).

```sh
docker build \
  --build-arg VERSION=$(git describe --tags --always --dirty) \
  --build-arg COMMIT=$(git rev-parse --short HEAD) \
  --build-arg BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
  -t gin-demo:latest .

docker run -d --name gin-demo -p 8080:8080 gin-demo:latest
```

The image declares a `HEALTHCHECK` that runs the binary's built-in `-healthcheck` flag,
which probes the local `/health` endpoint and exits non-zero on failure — so no `curl`
or `wget` is needed inside the minimal runtime image.

## Layout

```
cmd/gin-demo/main.go          Entrypoint: server bootstrap, graceful shutdown, healthcheck flag
internal/server/server.go     Router and handlers
internal/server/server_test.go
Dockerfile                    Multi-stage build → distroless runtime
Vagrantfile                   Ubuntu 24.04 VM + ansible_local provisioning
provisioning/site.yml         Ansible playbook (installs Go + Docker CE)
scripts/build_and_deploy.sh   Build image + run container (executed inside the VM)
scripts/run_tests.sh          go vet + go test -race (executed inside the VM)
vm.sh                         Host-side lifecycle driver
```
