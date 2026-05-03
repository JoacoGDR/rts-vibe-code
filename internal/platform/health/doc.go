// Package health provides the liveness and readiness probe handlers used
// by Kubernetes (and the local docker-compose stack). Liveness checks ask
// "is the process alive?"; readiness asks "is it ready to serve traffic?".
package health
