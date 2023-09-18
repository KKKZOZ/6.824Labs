# Failed Reason

## 2A-ManyElection-1.log

由于 select 和 channel 的阻塞，引起了死锁

## 2B-TestFailNoAgree2-1.log

由于 select 和 channel 的阻塞，引起了死锁

## TestFailAgree2B_86.log

Leader 变为 Follower 后，重新提交了之前已经提交过的日志
具体原因是在 commit checker 中，commit 后没有把 lastApplied 加一

## TestFailAgree2B_151.log

只要 Term 一致，不管 consistency check 是否通过，都应该重置 Timer

## TestRejoin2B_228.log

在发送RPC时，应该对整个发送过程加锁，而不是只对组装 args 的过程加锁，因为在顺序发送的 for 中解锁，可能会让其他地方抢到锁，修改 args 中的参数，比如这个 log 中的：
![image-20230918132605650](https://kkkzoz-1304409899.cos.ap-chengdu.myqcloud.com/img/image-20230918132605650.png)

## TestConcurrentStarts2B_1.log

没有在 Start() 函数中加锁，导致并发执行时函数的返回值错乱