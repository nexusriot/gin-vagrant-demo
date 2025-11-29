#!/usr/bin/env bash
set -euo pipefail

CMD="${1:-}"

usage() {
  cat <<EOF
Usage: $0 <command>

Commands:
  start        Start VM (vagrant up with VirtualBox)
  stop         Stop VM (vagrant halt)
  destroy      Destroy VM (vagrant destroy -f)
  build        Build & deploy Dockerized Gin service inside VM
  test         Run Go tests and health-check inside VM
  svc-start    Start gin-demo container inside VM (if already built)
  svc-stop     Stop gin-demo container inside VM
  svc-logs     Show logs of gin-demo container
EOF
}

if [[ -z "$CMD" ]]; then
  usage
  exit 1
fi

case "$CMD" in
  start)
    vagrant up --provider=virtualbox
    ;;
  stop)
    vagrant halt
    ;;
  destroy)
    vagrant destroy -f
    ;;
  build)
    vagrant ssh -c "bash /vagrant/scripts/build_and_deploy.sh"
    ;;
  test)
    vagrant ssh -c "bash /vagrant/scripts/run_tests.sh"
    echo "[INFO] Checking health endpoint..."
    vagrant ssh -c "curl -sf http://localhost:8080/health || (echo 'Health check failed' && exit 1)"
    echo "[INFO] Health check OK"
    ;;
  svc-start)
    vagrant ssh -c "docker start gin-demo"
    ;;
  svc-stop)
    vagrant ssh -c "docker stop gin-demo || true"
    ;;
  svc-logs)
    vagrant ssh -c "docker logs -f gin-demo"
    ;;
  *)
    echo "Unknown command: $CMD"
    usage
    exit 1
    ;;
esac

