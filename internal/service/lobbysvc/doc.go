// Package lobbysvc owns the lobby use cases: create / join / start / get
// / list. It speaks to a MatchesRepository (Postgres impl in pgrepo) and
// hands off "match started" notifications via the natsbridge Publisher.
package lobbysvc
