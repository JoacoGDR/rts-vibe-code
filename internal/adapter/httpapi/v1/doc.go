// Package v1 contains the HTTP controllers behind /api/v1. Each
// controller is one small file: it parses DTOs from pkg/api, dispatches
// to a service, and renders the response via internal/adapter/httpapi/render.
//
// Controllers should not contain business logic. If you find yourself
// wanting to write a `for ... else if` here, push it down into the
// service.
package v1
