// Package errs is the project-wide error catalogue. Every layer wraps its
// errors with one of the [Code] values declared here so the HTTP renderer
// (and future grpc/log middleware) can map a single error type onto the
// right transport-level signal — HTTP status, gRPC code, etc.
//
// Usage:
//
//	if user == nil {
//	    return errs.New(errs.NotFound, "user not found")
//	}
//	if err := repo.Save(ctx, user); err != nil {
//	    return errs.Wrap(err, errs.Internal, "saving user")
//	}
package errs
