
## Lab2
### A

```bash
❯ time go test -run 2A
Test (2A): initial election ...
  ... Passed --   3.4  3  118   39472    0
Test (2A): election after network failure ...
  ... Passed --   4.4  3  218   51629    0
Test (2A): multiple elections ...
  ... Passed --   5.3  7  912  219272    0
PASS
ok      6.5840/raft     13.169s
go test -run 2A  0.55s user 0.13s system 5% cpu 13.339 total
```

### B

```bash
❯ time go test -run 2B
Test (2B): basic agreement ...
  ... Passed --   0.9  3   22    6364    3
Test (2B): RPC byte count ...
  ... Passed --   1.0  3   48  117342   11
Test (2B): test progressive failure of followers ...
  ... Passed --   4.5  3  202   51160    3
Test (2B): test failure of leaders ...
  ... Passed --   4.8  3  346   86294    3
Test (2B): agreement after follower reconnects ...
  ... Passed --   3.6  3  142   44919    7
Test (2B): no agreement if too many followers disconnect ...
  ... Passed --   3.5  5  340   88992    3
Test (2B): concurrent Start()s ...
  ... Passed --   0.6  3   24    8357    6
Test (2B): rejoin of partitioned leader ...
  ... Passed --   6.0  3  346  100667    4
Test (2B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  12.8  5 2532 2342289  102
Test (2B): RPC counts aren't too high ...
  ... Passed --   2.1  3   98   37880   12
PASS
ok      6.5840/raft     39.864s
go test -run 2B  0.96s user 0.70s system 4% cpu 40.226 total
```

### C

```bash
❯ time go test -run 2C
Test (2C): basic persistence ...
  ... Passed --   3.3  3  126   38687    6
Test (2C): more persistence ...
  ... Passed --  14.7  5 1540  395910   16
Test (2C): partitioned leader and one follower crash, leader restarts ...
  ... Passed --   1.5  3   50   14901    4
Test (2C): Figure 8 ...
  ... Passed --  31.9  5 1736  445848   58
Test (2C): unreliable agreement ...
  ... Passed --   1.4  5 1000  416921  246
Test (2C): Figure 8 (unreliable) ...
  ... Passed --  29.1  5 11724 25467645   88
Test (2C): churn ...
  ... Passed --  16.1  5 3916 6684961  684
Test (2C): unreliable churn ...
  ... Passed --  16.1  5 5664 22040829 1384
PASS
ok      6.5840/raft     114.253s
go test -run 2C  7.91s user 1.93s system 8% cpu 1:54.61 total
```