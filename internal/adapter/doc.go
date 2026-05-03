// Package adapter groups the driver/driven adapters that connect the
// services to external systems:
//
//   - httpapi/   — driving (REST controllers + middleware)
//   - natsbridge/ — driven (NATS streams, subjects, payloads)
//   - pgrepo/    — driven (Postgres-backed repositories)
//   - redisrepo/ — driven (Redis-backed slot index, ticket store)
//
// Driving adapters depend on services. Driven adapters fulfil interfaces
// declared by the services that consume them (so services do not depend
// back).
package adapter
