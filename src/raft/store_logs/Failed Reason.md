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

## TestInitialElection2A_31981.log
### 总结
没有在 GetState() 中加锁，导致并发执行时函数的返回值错乱，运行3000次出现一次！

具体情况是，S2 在 T1 是 Leader，然后 S1 在 T2 在重新选举时向 S2 发送了 RV RPC，S2 更新 rf.currentTerm 为 T2, 由于 GetState() 未加锁，在 S2 还未来得及将自己降级为 Follower时， GetState() 获取 S2 的状态，为 (2,True)。S1 在成为 Leader 后，也向 GetState() 返回了 (2,True)，导致了测试程序报错 “term 2 has 2 (>1) leaders”

![image-20230918173649293](https://kkkzoz-1304409899.cos.ap-chengdu.myqcloud.com/img/image-20230918173649293.png)

### 解决方法

在 `GetState()` 中加锁


## TestManyElections2A_29240.log
### 总结

事故发生因为在 T2 有 2 个 Leader:

![](https://kkkzoz-1304409899.cos.ap-chengdu.myqcloud.com/img/image-20230918233239880.png)



关键区域未加锁，如下：
```go

// in func (rf *Raft) RequestVote()
if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.votedFor = -1

		if rf.currentState == Candidate {
			rf.debug(DVote, "stepping down from candidate in T%d\n", rf.currentTerm)
		}

		if rf.currentState == Leader {
			rf.debug(DVote, "stepping down from leader in T%d\n", rf.currentTerm)
		}
		rf.currentState = Follower
	}
```

在发现 `args.Term > rf.currentTerm` 后，应该加锁并连续执行到 `rf.currentState = Follower`

错误的时间线如下：

![614A190FF61254F757F43C8700ABCEED](https://kkkzoz-1304409899.cos.ap-chengdu.myqcloud.com/img/614A190FF61254F757F43C8700ABCEED.png)

### 解决方法

在 `startElection()` 中加锁

## TestFollowerFailure2B_4505.log

### 总结

事故发生因为在 T2 有 2 个 Leader:

![image-20230920211247365](https://kkkzoz-1304409899.cos.ap-chengdu.myqcloud.com/img/image-20230920211247365.png)

原因：在一个完整的执行流程未结束的情况下，为了调用一个函数，先把锁打开，再在函数中加锁。在打开锁后，锁被其他线程抢了过去，修改了关键值：

```go
// func (rf *Raft) startElection() {
if validVote && reply.VoteGranted {
	voteCount.mu.Lock()
	rf.mu.Lock()
	rf.debug(DVote, "received vote from S%d (for T%d) in T%d\n", i, reply.Term, rf.currentTerm)
	rf.mu.Unlock()
	voteCount.data += 1
	if voteCount.data >= rf.majority {
		if !voteCount.isTriggered {
			voteCount.isTriggered = true
			rf.mu.Lock()
			rf.debug(
				DLeader, "achieved Majority for T%d (%d), converting to leader\n", rf.currentTerm,
				rf.majority,
			)
			if rf.currentState == Candidate {
				rf.mu.Unlock()
				// 这个位置解了锁，默认是 Candidate，但在进入 rf.becomeLeader() 之前可能被其他
				// 线程修改为了 Follower，此时还是会继续执行下面的操作
				rf.becomeLeader()
			} else {
				rf.mu.Unlock()
			}
		}
	}
	voteCount.mu.Unlock()
}
```
### 解决方法

要意识到同一个线程之间的函数调用是没有问题的，只要保证调用者是加了锁的，被调用函数也是有锁的，并且要保证一个动作的完整执行


## TestFailAgree2B-2.log

### 总结

没有意识到 go routine 之间执行是乱序的，Log 如下，看起来很正常：

![image-20230921112043486](https://kkkzoz-1304409899.cos.ap-chengdu.myqcloud.com/img/image-20230921112043486.png)

但是 Test 报了 `apply error: server 0 apply out of order 3` 的错，查看代码：

```go
if args.LeaderCommit > rf.commitIndex {
	rf.commitIndex = min(args.LeaderCommit, rf.lastLogIndex)
	for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
		rf.debug(DCommit, "applies log at %d (cmd: %v) in T%d\n", i, Trunc(rf.log[i].Command, 3), rf.currentTerm)
		// TODO: Correct?
		go func(ii int, cmd any) {
			rf.applyChan <- ApplyMsg{
				CommandValid: true,
				Command:      cmd,
				CommandIndex: ii,
			}
		}(i, rf.log[i].Command)
		rf.lastApplied = i
	}
}
```

当时在写的时候就意识到可能出问题了哈哈，还写了个 `TODO`，出错的原因是go routine 之间执行可能是乱序的，可能会导致 `S0 applies log at 3 (cmd: 103) in T5` 这个事件比 `S0 applies log at 2 (cmd: 102) in T5` 先执行，这对 Raft 来说是不可接受的，不符合其 Linear 的限制

### 解决方法

如果是为了避免 channel wait 带来的死锁或者阻塞，应该在外层 for 循环时开启 go routine，这样既能保证不因为 channel 阻塞，也能保证 log 是按线性顺序被 apply 的

## Deadlock Example
###

![image-20230921165543588](https://kkkzoz-1304409899.cos.ap-chengdu.myqcloud.com/img/image-20230921165543588.png)

S1 在 vote 之后直接死锁了，相关代码有两处

第一处：在 `RequestVote()` 中

```go
// rf.mu.Lock()
if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
	// check if candidate's log is at least as up-to-date as receiver's log
	if args.LastLogTerm > rf.log[rf.lastLogIndex].Term ||
		(args.LastLogTerm == rf.log[rf.lastLogIndex].Term && args.LastLogIndex >= rf.lastLogIndex) {
		rf.debug(DVote, "votes for S%d in T%d\n", args.CandidateId, rf.currentTerm)
		reply.Term = rf.currentTerm
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		
		rf.notifyCh <- GrantReset
		return
	} else {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		rf.debug(DVote, "rejects voting S%d in T%d", args.CandidateId, rf.currentTerm)
		return
}

```

第二处： `rf.ticker()` 中：

```go
// Election Timeout
	for !rf.killed() {
		// Your code here (2A)
		// Check if a leader election should be started.
		interval := time.Duration(50+rand.Intn(20)) * 10 * time.Millisecond
		timeout := time.After(interval)
		select {
		case <-timeout:
			rf.mu.Lock()
			if rf.currentState != Leader {
				rf.startElection()
			}
			rf.mu.Unlock()
		case event := <-rf.notifyCh:
			rf.mu.Lock()
			switch event {
			case Reset:
				rf.debug(DTimer, "Resetting ELT by receiving an AppEnt in T%d\n", rf.currentTerm)
			case WinInElection:
				rf.debug(DTimer, "Resetting ELT by winning an election in T%d\n", rf.currentTerm)
			case GrantReset:
				rf.debug(DTimer, "Resetting ELT by granting others in T%d\n", rf.currentTerm)
			}
			rf.mu.Unlock()
		}
	}
```

![099484DC5CB03B68EE90341FBAE801D5](https://kkkzoz-1304409899.cos.ap-chengdu.myqcloud.com/img/099484DC5CB03B68EE90341FBAE801D5.png)