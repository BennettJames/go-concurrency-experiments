//
// Package brun offers a few helper utilities to manage groups of goroutines.
// There are three main utilities:
//
// - Group. This is designed for a set of long-running goroutines that are
// expected to run in concert. It ensures a uniform runtime with safe shutdown.
//
// - Batch. This is for ensuring a set of short lived tasks are all executed and
// completed together, complete with error management.
//
// - Set. This is for dynamic sets of tasks that may come and go. Has not
// intrinsic error handling of the goroutines; as they are inherently
// independent.
//
package brun
