package main

import (
	"sync"
	"sync/atomic"
)

type (
	// ThreadsafeArray is a simplified object meant to model and benchmark
	// different threadsafe strategies for serializing read/writes on a set of
	// data.
	ThreadsafeArray interface {
		Get() []int
		Add(amt int)
	}

	// ThreadsafeArrayFactory is a basic type that should instantiate a
	// ThreadsafeArray of the given length.
	ThreadsafeArrayFactory func(len int) ThreadsafeArray

	// MutexArray implements ThreadsafeArray with a single mutex.
	MutexArray struct {
		l sync.Mutex
		a []int
	}

	// RWMutexArray implements ThreadsafeArray with a RWMutex.
	RWMutexArray struct {
		l sync.RWMutex
		a []int
	}

	// SemiAtomicArray implements ThreadsafeArray with two levels of concurrency: a
	// lock that is required to be held by updates, and the value is held in an
	// atomic pointer. That allows writers to safely update the value, and removes
	// the need for a lock for readers.
	SemiAtomicArray struct {
		updateL sync.Mutex
		v       atomic.Value
	}

	// NoOpArray returns no array and performs no updates. This is useful as a
	// "baseline" to measure the cost of benchmarking itself.
	NoOpArray struct{}
)

func NewMutexArray(len int) ThreadsafeArray {
	return &MutexArray{
		a: make([]int, len),
	}
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

func NewRWMutexArray(len int) ThreadsafeArray {
	return &RWMutexArray{
		a: make([]int, len),
	}
}

func (ma *RWMutexArray) Get() []int {
	ma.l.RLock()
	defer ma.l.RUnlock()
	return ma.a
}

func (ma *RWMutexArray) Add(amt int) {
	ma.l.Lock()
	defer ma.l.Unlock()
	newA := make([]int, len(ma.a))
	for i, v := range ma.a {
		newA[i] = v + amt
	}
	ma.a = newA
}

func NewSemiAtomicArray(len int) ThreadsafeArray {
	aa := &SemiAtomicArray{}
	aa.v.Store(make([]int, len))
	return aa
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

func NewNoOpArray(len int) ThreadsafeArray {
	return &NoOpArray{}
}

func (a *NoOpArray) Get() []int {
	return nil
}

func (a *NoOpArray) Add(amt int) {
}
