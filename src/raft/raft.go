package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	. "6.5840/debug"
	//	"6.5840/labgob"
	"6.5840/labrpc"
)

// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

type ServerState int

const (
	Follower  ServerState = 0
	Candidate ServerState = 1
	Leader    ServerState = 2
)

type Event int

const (
	Reset         Event = 0
	WinInElection Event = 1
	GrantReset    Event = 2
)

const HeartbeatInterval = 150 * time.Millisecond

type WithMutex[T any] struct {
	mu   sync.Mutex
	data T
}

type LogEntry struct {
	Term    int
	Command interface{}
}

// A Go object implementing a single Raft peer.
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	currentState ServerState

	currentTerm int        // latest term server has seen (initialized to 0 on first boot, increases monotonically)
	votedFor    int        // candidateId that received vote in current term (or null if none)
	log         []LogEntry // log entries; each entry contains command for state machine, and term when entry was received by leader (first index is 1)

	commitIndex int // index of highest log entry known to be committed (initialized to 0, increases monotonically)
	lastApplied int // index of highest log entry applied to state machine (initialized to 0, increases monotonically)

	nextIndex  []int // for each server, index of the next log entry to send to that server (initialized to leader last log index + 1)
	matchIndex []int // for each server, index of highest log entry known to be replicated on server (initialized to 0, increases monotonically)

	majority int

	notifyCh chan Event
}

// GetState return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isLeader bool
	// Your code here (2A).

	//rf.mu.Lock()
	//defer rf.mu.Unlock()
	term = rf.currentTerm
	isLeader = rf.currentState == Leader
	return term, isLeader
}

// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
// before you've implemented snapshots, you should pass nil as the
// second argument to persister.Save().
// after you've implemented snapshots, pass the current snapshot
// (or nil if there's not yet a snapshot).
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// raftstate := w.Bytes()
	// rf.persister.Save(raftstate, nil)
}

// restore previously persisted state.
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).

}

// RequestVoteArgs example RequestVote RPC arguments structure.
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int // candidate's term
	CandidateId  int // candidate requesting vote
	LastLogIndex int // index of candidate's last log entry
	LastLogTerm  int // term of candidate's last log entry
}

// RequestVoteReply example RequestVote RPC reply structure.
// field names must start with capital letters!
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int  // currentTerm, for candidate to update itself
	VoteGranted bool // true means candidate received vote
}

// RequestVote RPC handler.
// Receiver implementation of RequestVote RPC.
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

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

	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		// TODO: check if candidate's log is at least as up-to-date as receiver's log
		reply.Term = rf.currentTerm
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		rf.debug(DVote, "votes for S%d\n", args.CandidateId)
		rf.notifyCh <- GrantReset
		return
	} else {
		reply.Term = rf.currentTerm
		reply.VoteGranted = false
		return
	}

}

// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

// AppendEntries Part

type AppendEntriesArgs struct {
	Term         int        // leader's term
	LeaderId     int        // so follower can redirect clients
	PrevLogIndex int        // index of log entry immediately preceding new ones
	PrevLogTerm  int        // term of prevLogIndex entry
	Entries      []LogEntry // log entries to store (empty for heartbeat; may send more than one for efficiency)
	LeaderCommit int        // leader's commitIndex
}

type AppendEntriesReply struct {
	Term    int  // currentTerm, for leader to update itself
	Success bool // true if follower contained entry matching prevLogIndex and prevLogTerm
}

// AppendEntries RPC handler.
// Receiver implementation of AppendEntries RPC.
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.votedFor = -1
		if rf.currentState == Leader {
			rf.debug(DVote, "stepping down from leader in T%d\n", rf.currentTerm)
		}
		rf.currentState = Follower
	}

	if rf.log[args.PrevLogIndex].Term != args.PrevLogTerm {
		reply.Term = rf.currentTerm
		reply.Success = false
		return
	}
	// 3,4,5 not implemented

	// heartbeat
	if args.Entries == nil {
		reply.Term = rf.currentTerm
		reply.Success = true
		rf.notifyCh <- Reset
		return
	}

}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

// AppendEntries Part End

type VoteCount struct {
	mu          sync.Mutex
	data        int
	isTriggered bool // to avoid multiple triggers
}

func (rf *Raft) startElection() {
	rf.mu.Lock()
	rf.currentState = Candidate
	rf.currentTerm += 1

	rf.debug(DLeader, "timeout, starts election T%d\n", rf.currentTerm)

	rf.votedFor = rf.me
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: len(rf.log) - 1,
		LastLogTerm:  rf.log[len(rf.log)-1].Term,
	}

	rf.mu.Unlock()

	voteCount := &VoteCount{data: 1, isTriggered: false}

	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			go func(i int) {
				reply := RequestVoteReply{}
				rf.sendRequestVote(i, &args, &reply)

				if reply.VoteGranted {
					voteCount.mu.Lock()
					voteCount.data += 1
					if voteCount.data >= rf.majority {
						if !voteCount.isTriggered {
							voteCount.isTriggered = true
							rf.debug(DLeader, "achieved Majority for T%d (%d), converting to leader\n", rf.currentTerm, rf.majority)
							if rf.currentState == Candidate {
								rf.becomeLeader()
							}
						}
					}
					voteCount.mu.Unlock()
				}
			}(i)
		}
	}
}

func (rf *Raft) becomeLeader() {
	rf.mu.Lock()
	rf.currentState = Leader
	rf.mu.Unlock()
	rf.notifyCh <- WinInElection
	for i := 0; i < len(rf.peers); i++ {
		rf.nextIndex[i] = len(rf.log)
		rf.matchIndex[i] = 0
	}
	rf.sendHeartBeat()
}

func (rf *Raft) sendHeartBeat() {
	rf.debug(DTimer, "Leader, checking heartbeats in T%d\n", rf.currentTerm)

	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			go func(i int) {
				rf.mu.Lock()
				args := AppendEntriesArgs{
					Term:         rf.currentTerm,
					LeaderId:     rf.me,
					PrevLogIndex: rf.nextIndex[i] - 1,
					PrevLogTerm:  rf.log[rf.nextIndex[i]-1].Term,
					Entries:      nil,
					LeaderCommit: rf.commitIndex,
				}
				rf.mu.Unlock()
				reply := AppendEntriesReply{}
				rf.sendAppendEntries(i, &args, &reply)
			}(i)
		}
	}
}

// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	index := -1
	term := -1
	isLeader := true

	// Your code here (2B).

	return index, term, isLeader
}

// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

func (rf *Raft) ticker() {

	go func() {
		for !rf.killed() {
			time.Sleep(HeartbeatInterval)

			rf.mu.Lock()
			flag := rf.currentState == Leader
			rf.mu.Unlock()

			if flag {
				rf.sendHeartBeat()
			}
		}
	}()

	for !rf.killed() {

		// Your code here (2A)
		// Check if a leader election should be started.
		interval := time.Duration(60+rand.Intn(30)) * 10 * time.Millisecond
		timeout := time.After(interval)
		select {
		case <-timeout:
			rf.mu.Lock()
			flag := rf.currentState == Leader
			rf.mu.Unlock()
			if !flag {
				go rf.startElection()
			}

		case event := <-rf.notifyCh:

			switch event {
			case Reset:
				// 这里加锁会死锁!
				//rf.mu.Lock()
				rf.debug(DTimer, "Resetting ELT, received AppEnt T%d\n", rf.currentTerm)
				//rf.mu.Unlock()
			case WinInElection:
			case GrantReset:
				rf.debug(DTimer, "Resetting ELT by granting others in T%d\n", rf.currentTerm)
			}

		}

		// pause for a random amount of time between 50 and 350
		// milliseconds.
		//ms := 50 + (rand.Int63() % 300)
		//time.Sleep(time.Duration(ms) * time.Millisecond)
	}
	rf.debug(DInfo, "stopped\n")
}

// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me

	// Your initialization code here (2A, 2B, 2C).
	rf.currentState = Follower
	rf.currentTerm = 0
	rf.votedFor = -1
	rf.log = make([]LogEntry, 1000)
	rf.log = append(rf.log, LogEntry{Term: 0, Command: nil})
	rf.commitIndex = 0
	rf.lastApplied = 0
	rf.nextIndex = make([]int, len(rf.peers))
	rf.matchIndex = make([]int, len(rf.peers))
	rf.majority = len(peers)/2 + 1
	rf.notifyCh = make(chan Event, 10)

	rand.Seed(time.Now().Unix())

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}

func (rf *Raft) debug(topic LogTopic, format string, a ...interface{}) {
	prefix := fmt.Sprintf("S%d ", rf.me)
	Debug(topic, prefix+format, a...)
}
