
# Go Concurrency Experiments

This is a multimodule Go project that I use to host some quick, one-off
experiments with go concurrency. Click through to any subdirectory to see a
project.


# Projects

- **[atomicupdate][atomicupdate]** tries out a few different approaches to
  handling high concurrent read/write access to data. The tl;dr is I suspect
  RWMutexes aren't great for a lot of workloads and have better alternatives.

- **[brun][brun]** is a proposed library and style of structuring goroutines to
  make parallelism safer and easier to manager.

[atomicupdate]:https://github.com/BennettJames/go-concurrency-experiments/tree/master/atomicupdate
[brun]:https://github.com/BennettJames/go-concurrency-experiments/tree/master/brun
