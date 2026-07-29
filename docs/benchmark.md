## benchmark

```bash
go test ./internal/vector \
    -run='^$' \
    -bench='TopK' \
    -benchmem \
    -count 3
```
- `-run='^$'` 不运行普通的单元测试
- `-bench 'TopK'` 运行名称包含 TopK 的 Bnechmark
- `-benchmem`  查看内存分配情况（在代码中添加`b.ReportAllocs()`也可以）
- `-count 3` 运行 3 次 benchmark 将每次的结果都报告

输出类似：
`BenchmarkTopKBySort-16    120000    9500 ns/op    16384 B/op    2 allocs/op`
- BenchmarkTopKBySort-16    名称以及使用的 GOMAXPROCS
- 120000                    Benchmark 执行次数
- 9500 ns/op                每次操作平均耗时
- 16384 B/op                每次操作累计分配的字节数(堆分配的内存)
- 2 allocs/op               每次操作发生的堆分配次数
- B/op 是累计分配量，不等于程序的峰值内存占用.


注意：
- benchmark 中不要使用 `fmt.Println`，会严重影响 benchmark 的性能
- `-race` 检测数据竞争，benchmark 中不要使用 `-race`，会严重影响 benchmark 的性能

`b.Run` 创建一个子 benchmark

b.Loop() 会返回 true 共 b.N 次，因此循环体总共执行 b.N 次（为了测试更可靠）。
循环结束后可以通过 b.N 获取实际迭代次数。, 之前的初始化不会记录到 benchmark 测试中

**使用 b.Loop() 时，每个 Benchmark/子 Benchmark 函数在一次测量中通常只进入一次，而 b.Loop() 的循环体会执行很多次。**
所以说 BenchmarkMemoryStoreInsertBatch 函数 创建 store 在 b.Loop()中


为什么 memory_benchmark_test.go 中 BenchmarkMemoryStoreSearch 分配的字节数只和 N 相关

1. 数据插入是在 `b.Loop()` 之前完成的，所以不计入 benchmark 测试中
2. 这个统计的是每次操作累计分配的字节数(堆分配的内存)，对于 search 函数 results 会进行堆内存分配, 堆排序过程中分配的 大小为 limit 的堆同样占用内存

## profile
### cpu profile
```
go test ./internal/store \
  -run '^$' \
  -bench '^BenchmarkMemoryStoreInsertBatch/N=1000/D=384$' \
  -benchtime=3s \
  -cpuprofile=/tmp/insert-cpu.out

go tool pprof -top /tmp/insert-cpu.out
```
输出：
- flat 函数自身直接消耗的 CPU
- cum 函数自身消耗的 CPU + 他调用的其他函数消耗的 CPU
```
File: store.test
Build ID: b4de9eee8ff93bba83c312d81c1b2e6a33f5ae07
Type: cpu
Time: 2026-07-29 18:53:43 CST
Duration: 3.58s, Total samples = 4.10s (114.44%)
Showing nodes accounting for 3.62s, 88.29% of 4.10s total
Dropped 121 nodes (cum <= 0.02s)
      flat  flat%   sum%        cum   cum%
     0.57s 13.90% 13.90%      3.48s 84.88%  github.com/mohae/deepcopy.copyRecursive
     0.38s  9.27% 23.17%      0.66s 16.10%  runtime.mallocgcTiny
     0.29s  7.07% 30.24%      0.39s  9.51%  reflect.Value.Index
     0.22s  5.37% 35.61%      1.01s 24.63%  runtime.mallocgc
     0.21s  5.12% 40.73%      1.36s 33.17%  reflect.packEfaceData
     0.17s  4.15% 44.88%      0.28s  6.83%  reflect.Value.assignTo
     0.17s  4.15% 49.02%      0.29s  7.07%  runtime.typedmemmove
     0.14s  3.41% 52.44%      0.14s  3.41%  runtime.nextFreeFast (inline)
     0.12s  2.93% 55.37%      0.70s 17.07%  reflect.Value.Set
     0.12s  2.93% 58.29%      0.41s 10.00%  reflect.typedmemmove
     0.12s  2.93% 61.22%      0.12s  2.93%  runtime.memmove
     0.09s  2.20% 63.41%      0.99s 24.15%  reflect.unsafe_New
     0.08s  1.95% 65.37%      0.08s  1.95%  reflect.Value.CanInterface (inline)
     0.08s  1.95% 67.32%      0.08s  1.95%  reflect.flag.mustBeAssignable (inline)
     0.08s  1.95% 69.27%      1.51s 36.83%  reflect.valueInterface
     0.07s  1.71% 70.98%      0.07s  1.71%  reflect.directlyAssignable
     0.07s  1.71% 72.68%      0.07s  1.71%  runtime.getMCache (inline)
     0.06s  1.46% 74.15%      0.06s  1.46%  internal/abi.(*Type).Kind (inline)
     0.06s  1.46% 75.61%      0.06s  1.46%  reflect.flag.kind (inline)
     0.05s  1.22% 76.83%      0.05s  1.22%  reflect.flag.ro (inline)
     0.04s  0.98% 77.80%      3.55s 86.59%  github.com/pfyao1101/miniRAG/internal/store.(*MemoryStore).Insert
     0.04s  0.98% 78.78%      1.40s 34.15%  reflect.packEface (inline)
     0.04s  0.98% 79.76%      0.06s  1.46%  runtime.scanObjectsSmall
     0.03s  0.73% 80.49%      0.03s  0.73%  github.com/pfyao1101/miniRAG/internal/store.makeBenchmarkVector (inline)
     0.03s  0.73% 81.22%      0.06s  1.46%  runtime.(*sweepLocked).sweep
     0.03s  0.73% 81.95%      0.04s  0.98%  runtime.findObject
     0.03s  0.73% 82.68%      0.03s  0.73%  runtime.futex
     0.03s  0.73% 83.41%      0.03s  0.73%  runtime.getempty
     0.03s  0.73% 84.15%      0.03s  0.73%  runtime.releasem (inline)
     0.03s  0.73% 84.88%      0.04s  0.98%  runtime.tryDeferToSpanScan
     0.02s  0.49% 85.37%      0.05s  1.22%  runtime.scanblock
     0.02s  0.49% 85.85%      0.06s  1.46%  runtime.stealWork
     0.02s  0.49% 86.34%      0.04s  0.98%  runtime.typePointers.next
     0.01s  0.24% 86.59%      1.52s 37.07%  reflect.Value.Interface (inline)
     0.01s  0.24% 86.83%      0.06s  1.46%  runtime.(*mcache).nextFree
     0.01s  0.24% 87.07%      0.05s  1.22%  runtime.(*mcentral).cacheSpan
     0.01s  0.24% 87.32%      0.25s  6.10%  runtime.gcDrainMarkWorkerDedicated (inline)
     0.01s  0.24% 87.56%      0.07s  1.71%  runtime.mallocgcSmallScanNoHeader
     0.01s  0.24% 87.80%      0.03s  0.73%  runtime.newMarkBits
     0.01s  0.24% 88.05%      0.08s  1.95%  runtime.newarray
     0.01s  0.24% 88.29%      0.08s  1.95%  runtime.scanObject
         0     0% 88.29%      3.49s 85.12%  github.com/mohae/deepcopy.Copy
         0     0% 88.29%      0.03s  0.73%  github.com/pfyao1101/miniRAG/internal/store.BenchmarkMemoryStoreInsertBatch
         0     0% 88.29%      3.55s 86.59%  github.com/pfyao1101/miniRAG/internal/store.BenchmarkMemoryStoreInsertBatch.func1
         0     0% 88.29%      3.50s 85.37%  github.com/pfyao1101/miniRAG/internal/store.cloneRecord (inline)
         0     0% 88.29%      0.03s  0.73%  github.com/pfyao1101/miniRAG/internal/store.makeBenchmarkRecord (inline)
         0     0% 88.29%      0.09s  2.20%  reflect.MakeSlice
         0     0% 88.29%      0.03s  0.73%  reflect.New
         0     0% 88.29%      0.07s  1.71%  reflect.unsafe_NewArray
         0     0% 88.29%      0.05s  1.22%  runtime.(*mcache).refill
         0     0% 88.29%      0.04s  0.98%  runtime.(*mcentral).grow
         0     0% 88.29%      0.04s  0.98%  runtime.(*mheap).alloc
         0     0% 88.29%      0.04s  0.98%  runtime.(*mheap).alloc.func1
         0     0% 88.29%      0.04s  0.98%  runtime.(*mheap).allocSpan
         0     0% 88.29%      0.07s  1.71%  runtime.bgsweep
         0     0% 88.29%      0.11s  2.68%  runtime.findRunnable
         0     0% 88.29%      0.30s  7.32%  runtime.gcBgMarkWorker
         0     0% 88.29%      0.28s  6.83%  runtime.gcBgMarkWorker.func2
         0     0% 88.29%      0.26s  6.34%  runtime.gcDrain
         0     0% 88.29%      0.05s  1.22%  runtime.mallocgcSmallNoscan
         0     0% 88.29%      0.09s  2.20%  runtime.markroot
         0     0% 88.29%      0.08s  1.95%  runtime.markroot.func1
         0     0% 88.29%      0.11s  2.68%  runtime.mcall
         0     0% 88.29%      0.11s  2.68%  runtime.park_m
         0     0% 88.29%      0.06s  1.46%  runtime.scanSpan
         0     0% 88.29%      0.06s  1.46%  runtime.scanframeworker
         0     0% 88.29%      0.08s  1.95%  runtime.scanstack
         0     0% 88.29%      0.12s  2.93%  runtime.schedule
         0     0% 88.29%      0.07s  1.71%  runtime.sweepone
         0     0% 88.29%      0.38s  9.27%  runtime.systemstack
         0     0% 88.29%      3.58s 87.32%  testing.(*B).run1.func1
         0     0% 88.29%      3.58s 87.32%  testing.(*B).runN
```

显式调用关系
```
go tool pprof -tree /tmp/insert-cpu.out
```

### mem profile
```
go test ./internal/store \
  -run '^$' \
  -bench '^BenchmarkMemoryStoreInsertBatch/N=1000/D=384$' \
  -benchtime=3s \
  -memprofile=/tmp/insert-mem.out

go tool pprof -top -alloc_space /tmp/insert-mem.out
```
输出：(总共分配的内存)
```
File: store.test
Build ID: b4de9eee8ff93bba83c312d81c1b2e6a33f5ae07
Type: alloc_space
Time: 2026-07-29 18:54:50 CST
Showing nodes accounting for 486.94MB, 99.90% of 487.45MB total
Dropped 34 nodes (cum <= 2.44MB)
      flat  flat%   sum%        cum   cum%
  201.50MB 41.34% 41.34%   201.50MB 41.34%  reflect.unsafe_New
  175.26MB 35.95% 77.29%   175.26MB 35.95%  reflect.unsafe_NewArray
   37.61MB  7.72% 85.01%   470.38MB 96.50%  github.com/pfyao1101/miniRAG/internal/store.(*MemoryStore).Insert
   33.51MB  6.87% 91.88%    33.51MB  6.87%  reflect.mapassign_faststr0
   12.05MB  2.47% 94.35%    12.05MB  2.47%  github.com/pfyao1101/miniRAG/internal/store.makeBenchmarkVector (inline)
       8MB  1.64% 95.99%   432.77MB 88.78%  github.com/pfyao1101/miniRAG/internal/store.cloneRecord (inline)
       8MB  1.64% 97.64%        8MB  1.64%  reflect.makemap
    4.02MB  0.82% 98.46%     4.02MB  0.82%  runtime.mallocgc
    3.50MB  0.72% 99.18%   178.76MB 36.67%  reflect.MakeSlice
       3MB  0.62% 99.79%     4.50MB  0.92%  reflect.Value.MapKeys
    0.50MB   0.1% 99.90%    12.55MB  2.57%  github.com/pfyao1101/miniRAG/internal/store.makeBenchmarkRecord (inline)
         0     0% 99.90%   424.77MB 87.14%  github.com/mohae/deepcopy.Copy
         0     0% 99.90%   410.27MB 84.17%  github.com/mohae/deepcopy.copyRecursive
         0     0% 99.90%    13.05MB  2.68%  github.com/pfyao1101/miniRAG/internal/store.BenchmarkMemoryStoreInsertBatch
         0     0% 99.90%   470.38MB 96.50%  github.com/pfyao1101/miniRAG/internal/store.BenchmarkMemoryStoreInsertBatch.func1
         0     0% 99.90%        8MB  1.64%  reflect.MakeMap (inline)
         0     0% 99.90%        8MB  1.64%  reflect.MakeMapWithSize
         0     0% 99.90%    10.50MB  2.15%  reflect.New
         0     0% 99.90%   187.50MB 38.47%  reflect.Value.Interface (inline)
         0     0% 99.90%    33.51MB  6.87%  reflect.Value.SetMapIndex
         0     0% 99.90%     3.50MB  0.72%  reflect.copyVal
         0     0% 99.90%    33.51MB  6.87%  reflect.mapassign_faststr
         0     0% 99.90%   187.50MB 38.47%  reflect.packEface (inline)
         0     0% 99.90%   187.50MB 38.47%  reflect.packEfaceData
         0     0% 99.90%   187.50MB 38.47%  reflect.valueInterface
         0     0% 99.90%     3.02MB  0.62%  runtime.newobject
         0     0% 99.90%   483.43MB 99.18%  testing.(*B).run1.func1
         0     0% 99.90%   483.43MB 99.18%  testing.(*B).runN
```


## 问题
commit: 11154fd30ba293aa56d786139747f77bbc436c9a 之后
通过 profile 发现，MemoryStore.Insert 函数中调用 deepcopy.Copy 函数占用大量 CPU 和内存，主要是因为每次插入都需要对 record 进行深拷贝(深拷贝使用反射，会导致多余的内存分配)。

后续直接通过 `record.Clone()` 进行浅拷贝，减少 CPU 和内存占用。

优化前：allocs/record ≈ dimension + 13
主要是 deepcopy 针对 []float32数组的反射和装箱处理以及 metadata 的递归复制

优化后：allocs/record ≈ 3
1. slices.Clone 为 Vector 创建一个新底层数组；
2. maps.Clone 创建新的 map 结构；
3. maps.Clone 为这个 map 分配存储 bucket。
具体是否将 map 结构与 bucket 合并分配属于 Go Runtime 的实现细节，但当前结果显示，每条记录大约是三次堆分配。