
# Brun; Shieldmaiden of Goroutines

_**Author's note**: this is an experimental, actively developed library that's been
made public to solicit some feedback. Comments/suggestions/insults are
welcome._

## Super-Short Introduction

Goroutines are an easy way to introduce concurrency and concurrency-related bugs
to a program. Goroutines can become blocked; they can leak and run way past
their initial scope; panics immediately crash the program; errors can be dropped
and missed.

This library has a simple utility, `brun.Group`, that manages long-lived
goroutines and their control flow. Here it is in action:

```go
func runTheThing(ctx context.Context) error {
  sg := &brun.Group{}

  sg.Add(func(ctx context.Context) error {
    for {
      // do stuff
      select {
        // handle events
        case <-ctx.Done():
          return ctx.Err()
      }
    }
  })

  sg.Add(func(ctx context.Context) error {
    for {
      // do stuff
      select {
        // handle events
        case <-ctx.Done():
          return ctx.Err()
      }
    }
  })

  return sg.Run(ctx)
}
```

Here's what it does:

- Each added sub-function is run in it's own goroutine when `Run` is called.

-  `Run` will then block until it is cancelled, or one of the sub-functions
   returns. It will then cancel every other sub-function, and wait for them to
   complete.

- If a sub-function returns an error, that error is returned by `Run` after all
  other sub-functions have completed.

- If a sub-function panics, then that panic is sent back by `Run`.

By managing goroutines in groups, their behavior is more obvious and many bugs
can be avoided. See the [docs][brun-docs] for more details on the API, it's
usage, and helpers. To understand the why and how of this library, read on for a
detailed look at some potential concurrency pitfalls and how to avoid them.


## Extended Introduction

### The "Blocking Run" Pattern

Before we look at coordinating multiple long-running services, let's consider
how to handle just one long-running service via a context. A useful pattern is
to have a function that accepts a context and a callback, and blocks until it's
either cancelled or it encounters a fatal error - that is, a "blocking run".
Signatures typically will look like this:

```go
func run(ctx context.Context, callback func(message interface{})) error
```

If it needs to pass values back to the caller, `callback` should be invoked with
it (a channel can also be used - that's a matter of taste). If a fatal error is
encountered, then an error will be returned, and all behavior within the
function cancelled. This removes the need to juggle different channels,
callbacks, and defers that are needed to manage the lifetime of a service, and
simplifies it down to just a single explicit function call.

Here's a very simple example of a blocking run service. An object is created
with a polling function, and the object has a `run` method that continuously
polls and notifies the caller of changes:

```go
type struct Foo {}

type FooStreamer interface {
  Run(ctx context.Context, onFoo func(f Foo)) error
}

type SingleFooStreamer struct {
  fetch func(ctx context.Context) (*Foo, error)
}

func NewSingleFooStreamer(
  fetch(ctx context.Context) (*Foo, error),
) *FooStreamerImpl {
  return *FooStreamerImpl{
    fetch: fetch,
  }
}

func (s *SingleFooStreamer) Run(ctx context.Context, onFoo func(f Foo)) error {
  var lastFoo Foo
  for {
    newFoo, err := s.fetch(ctx)
    if err != nil {
      fmt.Println("error fetching data in foo streamer: ", err)
    } else if isDifferent(lastFoo, newFoo) {
      lastFoo = newFoo
      onFoo(newFoo)
    }
    select {
      case <- time.After(5 * time.Second):
        continue
      case <-ctx.Done():
        return ctx.Err()
    }
  }
}
```

In this case, using a full object/interface pattern is excessive; but is useful
in many real-world scenarios when a service needs to perform I/O and make
additional methods available to the rest of the program.


### Composing Blocking Runs With Goroutines

Blocking runs don't get you very far just by themselves. As they, well, _block_,
the caller, to perform actual concurrency there needs to still be a way to run
them in parallel. Let's try to compose multiple blocking runs together with
goroutines:

```go
type DoubleFooStreamer struct {
  stream1, stream2 SingleFooStreamer
}

func NewDoubleFooStreamer(
  stream1, stream2 SingleFooStreamer,
) *FooStreamerImpl {
  return *FooStreamerImpl{
    stream1: stream1,
    stream2: stream2,
  }
}

func (s *DoubleFooStreamer) Run(ctx context.Context, onFoo func(f Foo)) error {
  ctx, cancel := context.WithCancel(ctx)
  defer cancel()

  msgChan := make(chan Foo, 1)
  errChan := make(chan err, 2)

  forward := func(f Foo) {
    select {
    case msgChan <- f:
    case <-ctx.Done():
    }
  }

  go func() {
    errChan <- s.stream1.Run(ctx, forward)
  }()

  go func() {
    errChan <- s.stream2.Run(ctx, forward)
  }()

  for {
    select {
    case msg := <-msgChan:
      onFoo(msg)
    case err := <-errChan:
      return err
    }
  }
}
```

Each blocking run is put into it's own goroutine, and the parent becomes it's
own blocking run. The lifetimes and messages of the sub-routines are managed by
the parent. Messages are read into a channel to ensure the callback is only
invoked one-at-a-time. If either stream encounters a failure, the error is
returned, the local context cancelled, and the other stream will subsequently
halt.

On the surface, this might feel like an inversion of the typical way go handles
cleanup. Often, a resource is created, and cleanup is put in a `defer
resource.Close()` block. That works great in simple procedural cases - a set of
resources is created, inspected as necessary, then released once the function
returns.

That pattern doesn't quite work when you're coordinating multiple concurrent
routines though - they each need their own scopes to manage processing and
behavior. Blocking runs are in a sense a way to still manage lifetimes together
for concurrent processes - each lifetime is unified together in a single
function scope, and each sub function is able to handle the distinct behaviors
of a single routine.

________

Honestly, the above code is a decent way to handle concurrent blocking runs
as-is. If you just want to use that pattern without any library, that'd be a
fine way to structure concurrency.

There are one or two possible pitfalls. It does rely on the error channel always
being used correctly within goroutines - otherwise, a goroutine could silently
exit. Also, if the size or the goroutine is too small, it's possible that on
cancellation a goroutine might be blocked writing to a channel no one is reading
from.

If those aren't a concern for you, great - stop reading here. Keep reading if
you're interested in some thin wrappers that can take care of that for you.



### Composing Blocking Runs With brun.Group

Parallelizing blocking runs followed the same basic pattern. Put each run in a
goroutine, instrument them with a shared context, make sure to send a signal
back on error/completion, cancel and return on the first error encountered, make
sure that internal channels are large enough to not inadvertently block. That's
a good deal of common bookkeeping that can be abstracted.

And with that, we return to the `Group` that was outlined in the introduction.
Rather than self-manage goroutines and error channels, we can create a helper
that does it for us. Let's take another look at `DoubleFooStreamer`, but with
service groups:

```go

func (s *DoubleFooStreamer) Run(ctx context.Context, onFoo func(f Foo)) error {
  ctx, cancel := context.WithCancel(ctx)
  defer cancel()

  msgChan := make(chan Foo, 1)
  forward := func(f Foo) {
    select {
    case msgChan <- f:
    case <-ctx.Done():
    }
  }

  return brun.GroupRun(
    ctx,
    func(ctx context.Context) error {
      return s.stream1.Run(ctx, forward)
    },
    func(ctx context.Context) error {
      return s.stream2.Run(ctx, forward)
    },
    func(ctx context.Context) error {
      for {
        select {
        case msg := <-msgChan:
          onFoo(msg)
        case <-ctx.Done():
          return ctx.Err()
        }
      }
    })
}
```

Let's try to simplify a little further by re-arranging some arguments. Rather
than have direct `Run` functions that block when executed, we can return a `Run`
function:

```go

type FooStreamer interface {
  Runner(func(Foo)) func(context.Context) error
}

func (s *DoubleFooStreamer) Run(onFoo func(Foo)) func(context.Context) error {
  msgChan := make(chan Foo, 1)
  forward := func(f Foo) {
    select {
    case msgChan <- f:
    case <-ctx.Done():
    }
  }

  return brun.GroupRunner(
    s.stream1.Runner(forward),
    s.stream2.Runner(forward),
    func(ctx context.Context) error {
      for {
        select {
        case msg := <-msgChan:
          onFoo(msg)
        case <-ctx.Done():
          return ctx.Err()
        }
      }
    })
}
```

This leads to a fairly lean, simple API that doesn't even require anything from
`brun` be exported as part of your API. Through simple function composition,
goroutines can be made safer and their runtimes made more obvious with very
little added complexity.

Which is the essence of what blocking runs accomplish. They are a formal,
minimalist way to make the control flow between goroutines as obvious as
possible. By keeping all goroutines in a strict hierarchy of groups that each
have a specific, defined policy for fanout, cancellation, and errors, reasoning
about the runtimes and behavior of concurrency is made easy by obviousness -
which, really, is the heart of good Go design.
