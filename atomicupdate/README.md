
# Benchmarking Some Serialized Data Patterns

I was curious about the performance of some different ways of serializing
reads/writes on a Go data structure. What follows is a description of some
different ways to do this; and is followed by some benchmarking under different
usage scenarios.


## Introduction

(todo [bs]: Let's add one-or-two sentences here about the need for
synchronization; just that the CPU needs some special guidelines when data is
accessed between threads. Goal isn't to create a treatise on the fundamentals of
concurrency; but should at least introduce th e need for this in broad strokes.
Also may wish to tweak the existing intro paragraph as part of this)

A common multi-threading problem: there is some data in your application.
Readers and writers will need to access and modify the data from multiple
threads. How is this access made to be safe?

I often find myself following this basic pattern to guarantee thread safety:

- Treat the data itself as immutable. It can be fetched and set, but all
  subfields and data structures can never be modified.

- Whenever any part of the data needs to be updated, create a complete copy of
  the original with whatever updated values are needed.

- Perform updates and gets within a lock to ensure multiple threads can get
  access. As the data is immutable, simply return a reference and let any
  readonly processing happen outside of the lock.

Of course, there are plenty of situational alterations to this pattern; but it's
a good starting point when dealing with sharing data between threads. For the
sake of this experiment, we'll be using the following toy interface:

```go
// ThreadsafeArray is an interface that represents an integer array of fixed
// length.
type ThreadsafeArray interface {
 	// Get returns the underlying array. Note that this should be treated as
 	// immutable.
 	Get() []int

	// Add increments every value in the array by the given amount.
	Add(amt int)
}
```

This is a comically underpowered interface without real world value. It is
acting as a stand-in for the basic type of concurrency under test here:
immutable data, cheap reads, and expensive writes. The expense can
be dynamically varied by changing the size of the array.


### Locks

The most common way to deal with this is with a lock (or "mutex"). Here's an
implementation using a plain go mutex:

```go
// MutexArray implements ThreadsafeArray with a single mutex.
type MutexArray struct {
	l sync.Mutex
  a []int
}

func (ma *MutexArray) Get() []int {
	ma.l.Lock()
	defer ma.l.Unlock()
	return ma.a
}

func (ma *MutexArray) Add(amt int) {
	ma.l.Lock()
	defer ma.l.Unlock()
	newA := make([]int, len(ma.a))
	for i, v := range ma.a {
		newA[i] = v + amt
	}
	ma.a = newA
}
```

(todo [bs]: let's try the benchmarks without defer as well; that seems fair.)

This is as simple as it gets. If the data needs to be read or written, a lock is
first required. Anyone else who needs to act on the data will try to acquire the
same lock, and wait until the lock is free before taking action.

A common variant of this is to use a read-write lock rather than just a plain
lock. A read-write lock let's multiple readers access the data at the same time,
but ensures if anyone is writing to the data everyone else has to wait.


### Atomics

Atomics can be used as an alternative to locks. They use special CPU
instructions for writing/reading to avoid the need for a lock. This can avoid
blocking, but at the expense of being more limited. They can only write/read a
few bytes. Caution is needed to ensure correctness of any solution.

Consider (RW)MutexArray - while the writer is preparing an update, no reader can
proceed. The writer isn't modifying any existing data up until the end; most of
the work is spent creating a new version of the data.

Instead of a lock, we could replace the value with an atomic reference. A read
can just get this value reliably with less overhead. A write can first perform
an atomic read, perform any necessary changes on the data copy, then perform an
atomic write.

But: this is too simplistic. What happens if two writes happen at the same time?
They can both make a copy of the same base array, create a derived version, then
perform the write. Since they both were working on the same array, one will
_overwrite_ the other. We need these writes, updates and all, to happen
in-sequence.

If you wish to make it truly lockless, a _compare and swap_ (CAS) can be
used. This will conditionally write the value if it hasn't changed.

1. Atomically read the value
2. Perform the update
3. Compare-and-swap the value with the new one
4. If the swap succeeds; return. Otherwise, go back to 1.

As this experiment specifically is for cases with cheap reads and expensive
writes, CAS operations aren't ideal. If write density gets too high, then you
will have to repeat the write over-and-over before it succeeds.

This experiment includes a hybrid implementation, where the value is atomic but
a lock must be acquired to update the value. This allows reads to always proceed
with zero contention, even during an update, but ensures all updates happen
serially -

```go
type SemiAtomicArray struct {
	updateL sync.Mutex
	v       atomic.Value
}

func (ma *SemiAtomicArray) Get() []int {
	return ma.v.Load().([]int)
}

func (ma *SemiAtomicArray) Add(amt int) {
	ma.updateL.Lock()
	defer ma.updateL.Unlock()
	oldA := ma.v.Load().([]int)
	newA := make([]int, len(oldA))
	for i, v := range oldA {
		newA[i] = v + amt
	}
	ma.v.Store(newA)
}
```


(aside [bs]: consider trying to extract and understand the details of
AtomicValue. Particularly, it has some eccentricities when initializing type
that are a little odd and aren't really necessary here. Also; I wouldn't mind
trying a fully lockless implementation as well via unsafe.Pointer. After reading
the code I suspect that'd be doable without a ton of effort).


## Benchmarking the Implementations

So, with all that introducing out of the way, let's take a look at how these
perform under different conditions. We'll consider four main variables:

- The size of the array, which is a stand-in for how expensive the update is.

- Number writes per second. If this is high enough, the data will be constantly
  written to.

- The number of reader and writer goroutines. Each goroutine will occasionally
  preempt itself by taking nanosecond sleeps at occasional intervals.

In addition to the mutex, rwmutex, and semi-atomic implementations, a "no-op"
implementation is included as a baseline. It does nothing - it returns a nil
slice and performs no updates.

This is solely being tested on my old mac mini, with a 2012 i7. The total number
of active goroutines is generally limited to 8, as that is the number of
threads.

Without further adieu - let's dive into some benchmarks.


### Plain vs RW Mutex







(aside [bs]: the rwmutex performs badly here in large part because no real work
is done on the lock. It'd be worth clarifying that this situation isn't
necessarily typical; and in cases where you just need to derive some data it may
be better to use hidden, mutable data and more complicated processing methods
that hold the read lock)


## Conclusions

(todo [bs]: this was written off of some preliminary results; should definitely
revisit it later)

In sum: in basic cases that don't involve huge amounts of contention, a basic
lock will do fine. Correctness is always a more pressing concern than
performance, and the average app doesn't need to worry about this. When there is
the need for a high amount of quick reads and expensive updates, it might be
worth doing the split atomic/lock technique.


