// Package render is the single source of truth for HTTP responses across
// the API. It produces consistent JSON envelopes for both success and error
// payloads so the frontend never has to care which controller answered.
package render
