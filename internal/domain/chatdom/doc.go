// Package chatdom is the domain model for in-match chat. It owns the
// scope grammar (world / coalition / dm), the [Message] aggregate and the
// validation rules every chat write must pass. Persistence lives in
// [pgrepo], live fan-out lives in [natsbridge], and the use case
// orchestration lives in [chatsvc] — this package only knows about
// pure-Go types.
package chatdom
