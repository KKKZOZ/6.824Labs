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
	"bytes"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"6.5840/labgob"

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
	Reset          Event = 0
	WinInElection  Event = 1
	GrantReset     Event = 2
	HeartbeatReset Event = 3
)

const HeartbeatInterval = 50 * time.Millisecond

type LogEntry struct {
	Index   int
	Term    int
	Command interface{}
}

func (l LogEntry) String() string {
	return fmt.Sprintf("{%d %v %d}", l.Term, Trunc(l.Command, 3), l.Index)
}

// Raft A Go object implementing a single Raft peer.
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

	majority      int
	lastResetTime time.Time

	notifyCh    chan Event
	heartbeatCh chan Event

	applyChan chan ApplyMsg
	inputChan chan ApplyMsg
	snapshot  Snapshot
}

type PersistentState struct {
	CurrentTerm       int
	VotedFor          int
	Log               []LogEntry
	LastIncludedIndex int
	LastIncludedTerm  int
}

func (p PersistentState) String() string {
	return fmt.Sprintf("{T:%d VF:%d LogLength:%d}", p.CurrentTerm, p.VotedFor, len(p.Log))
}

type Snapshot struct {
	LastIncludedIndex int
	LastIncludedTerm  int
	Data              []byte
}

func (sp Snapshot) String() string {
	return fmt.Sprintf("{LastIncludedIndex:%d LastIncludedTerm:%d}", sp.LastIncludedIndex, sp.LastIncludedTerm)
}

// GetState return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {

	var term int
	var isLeader bool
	// Your code here (2A).

	rf.mu.Lock()
	term = rf.currentTerm
	isLeader = rf.currentState == Leader
	rf.mu.Unlock()
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
	state := PersistentState{
		CurrentTerm:       rf.currentTerm,
		VotedFor:          rf.votedFor,
		Log:               rf.log,
		LastIncludedIndex: rf.snapshot.LastIncludedIndex,
		LastIncludedTerm:  rf.snapshot.LastIncludedTerm,
	}
	rf.debug(
		DPersist, "persisting state in T%d: %v, rf.getFirstLogIndex:%d\n", rf.currentTerm, state, rf.getFirstLogIndex(),
	)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	err := e.Encode(state)
	if err != nil {
		return
	}
	raftState := w.Bytes()
	if rf.snapshot.LastIncludedIndex != 0 {
		rf.persister.Save(raftState, rf.snapshot.Data)
	} else {
		rf.persister.Save(raftState, nil)
	}

}

// restore previously persisted state.
func (rf *Raft) readPersist(raftState []byte, spBytes []byte) {
	if raftState == nil || len(raftState) < 1 { // bootstrap without any state?
		return
	}
	r := bytes.NewBuffer(raftState)
	d := labgob.NewDecoder(r)
	var state PersistentState
	if d.Decode(&state) != nil {
		rf.debug(DError, "Error: Raft %d failed to read raftState\n", rf.me)
	} else {
		rf.mu.Lock()
		rf.currentTerm = state.CurrentTerm
		rf.votedFor = state.VotedFor
		rf.log = state.Log
		rf.commitIndex = state.LastIncludedIndex
		rf.lastApplied = state.LastIncludedIndex
		rf.snapshot.LastIncludedIndex = state.LastIncludedIndex
		rf.snapshot.LastIncludedTerm = state.LastIncludedTerm
		rf.debug(DPersist, "read persist in T%d: %v\n", rf.currentTerm, state)
		rf.mu.Unlock()
	}

	if spBytes != nil && len(spBytes) > 0 {
		rf.mu.Lock()
		rf.snapshot.Data = spBytes
		rf.inputChan <- ApplyMsg{
			SnapshotValid: true,
			Snapshot:      rf.snapshot.Data,
			SnapshotTerm:  rf.snapshot.LastIncludedTerm,
			SnapshotIndex: rf.snapshot.LastIncludedIndex,
		}
		rf.mu.Unlock()
	}

}

// Snapshot the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()

	if index <= rf.snapshot.LastIncludedIndex {
		return
	}

	rf.snapshot = Snapshot{
		LastIncludedIndex: index,
		LastIncludedTerm:  rf.log[index-rf.getFirstLogIndex()].Term,
		Data:              snapshot,
	}
	rf.log = rf.log[index+1-rf.getFirstLogIndex():]
	rf.debug(DSnap, "snapshotting in T%d: %v\n", rf.currentTerm, rf.snapshot)
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
	defer rf.persist()

	if args.Term < rf.currentTerm {
		reply.Term, reply.VoteGranted = rf.currentTerm, false
		return
	}

	if args.Term > rf.currentTerm {
		rf.currentTerm, rf.votedFor = args.Term, -1
		rf.convertToFollower()
	}

	if rf.votedFor == -1 || rf.votedFor == args.CandidateId {
		// check if candidate's log is at least as up-to-date as receiver's log
		if args.LastLogTerm > rf.getTermByIndex(len(rf.log)-1) ||
			(args.LastLogTerm == rf.getTermByIndex(len(rf.log)-1) && args.LastLogIndex >= rf.getLastLogIndex()) {
			rf.debug(DVote, "votes for S%d in T%d\n", args.CandidateId, rf.currentTerm)
			reply.Term, reply.VoteGranted = rf.currentTerm, true
			rf.votedFor = args.CandidateId
			rf.lastResetTime = time.Now()
			go func() { rf.notifyCh <- GrantReset }()
			return
		} else {
			reply.Term, reply.VoteGranted = rf.currentTerm, false
			rf.debug(DVote, "rejects voting S%d in T%d", args.CandidateId, rf.currentTerm)
			return
		}
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
// handler function on the server side does not return, thus there
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

type RejectionDetail struct {
	XTerm  int // term in the conflicting entry (if any)
	XIndex int // index of first entry with that term (if any)
	XLen   int // log length
}

type AppendEntriesReply struct {
	Term          int  // currentTerm, for leader to update itself
	Success       bool // true if follower contained entry matching prevLogIndex and prevLogTerm
	RejectionInfo RejectionDetail
}

// AppendEntries RPC handler.
// Receiver implementation of AppendEntries RPC.
func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()

	if args.Term > rf.currentTerm {
		rf.currentTerm, rf.votedFor = args.Term, -1
		rf.convertToFollower()
	}

	if args.Term == rf.currentTerm && rf.currentState == Candidate {
		rf.convertToFollower()
	}

	if args.Term < rf.currentTerm {
		reply.Term, reply.Success = rf.currentTerm, false
		return
	}

	// check whether preLogIndex is larger than lastLogIndex
	// follower's log is too short
	if args.PrevLogIndex > rf.getLastLogIndex() {
		reply.Term, reply.Success = rf.currentTerm, false
		reply.RejectionInfo = RejectionDetail{
			XTerm:  -1,
			XIndex: -1,
			XLen:   rf.getLastLogIndex(),
		}
		rf.lastResetTime = time.Now()
		go func() { rf.notifyCh <- Reset }()
		rf.debug(
			DLog2, "Rejected AE RPC in T%d (from S%d), because args.PrevLogIndex:%d > rf.lastLogIndex:%d",
			rf.currentTerm, args.LeaderId, args.PrevLogIndex, rf.getLastLogIndex(),
		)
		return
	}

	// TODO: ???
	if args.PrevLogIndex+1 < rf.getFirstLogIndex() {
		reply.Term, reply.Success = rf.currentTerm, false
		reply.RejectionInfo = RejectionDetail{
			XTerm:  -1,
			XIndex: -1,
			XLen:   rf.getLastLogIndex(),
		}
		rf.lastResetTime = time.Now()
		go func() { rf.notifyCh <- Reset }()
		rf.debug(
			DLog2, "Rejected AE RPC in T%d (from S%d), because args.PrevLogIndex:%d < rf.firstLogIndex:%d",
			rf.currentTerm, args.LeaderId, args.PrevLogIndex, rf.getFirstLogIndex(),
		)
		return
	}

	// rf.log[args.PrevLogIndex-rf.getFirstLogIndex()].Term
	if rf.getTermByIndex(args.PrevLogIndex-rf.getFirstLogIndex()) != args.PrevLogTerm {
		reply.Term, reply.Success = rf.currentTerm, false
		var rejectInfo RejectionDetail

		firstIndex := args.PrevLogIndex
		for ; firstIndex-rf.getFirstLogIndex() >= 0; firstIndex-- {
			if rf.getTermByIndex(firstIndex-rf.getFirstLogIndex()) != rf.getTermByIndex(args.PrevLogIndex-rf.getFirstLogIndex()) {
				break
			}
		}
		firstIndex++
		rejectInfo = RejectionDetail{
			XTerm:  rf.log[args.PrevLogIndex-rf.getFirstLogIndex()].Term,
			XIndex: firstIndex,
			XLen:   rf.getLastLogIndex(),
		}

		reply.RejectionInfo = rejectInfo
		rf.lastResetTime = time.Now()
		go func() { rf.notifyCh <- Reset }()
		rf.debug(
			DLog2, "Rejected AE RPC (from S%d), Logs: %v in T%d\n",
			args.LeaderId, rf.log, rf.currentTerm,
		)
		rf.debug(
			DLog2, "rf.log[args.PrevLogIndex].Term: %d, args.PrevLogTerm: %d\n",
			rf.log[args.PrevLogIndex-rf.getFirstLogIndex()].Term, args.PrevLogTerm,
		)
		return
	}

	// for log entries
	if args.Entries != nil {
		// If an existing entry conflicts with a new one (same index but different terms)
		// delete the existing entry and all that follow it

		//if rf.log[args.PrevLogIndex+1] != args.Entries[0] {
		//	// rf.log = rf.log[:args.PrevLogIndex+1]
		//	rf.lastLogIndex = args.PrevLogIndex
		//}

		// Append any new entries not already in the log
		for i := 0; i < len(args.Entries); i++ {

			if args.PrevLogIndex+i+1 <= rf.getLastLogIndex() {
				if rf.log[args.PrevLogIndex+i+1-rf.getFirstLogIndex()].Term != args.Entries[i].Term {
					rf.log = rf.log[:args.PrevLogIndex+i+1-rf.getFirstLogIndex()]
					rf.log = append(rf.log, args.Entries[i])
				}
			} else {
				rf.log = append(rf.log, args.Entries[i])
			}

			//if rf.log[args.PrevLogIndex+i+1] != args.Entries[i] {
			//	rf.log[args.PrevLogIndex+i+1] = args.Entries[i]
			//}
			//rf.lastLogIndex++
			//rf.log[rf.lastLogIndex] = args.Entries[i]
		}
		//rf.lastLogIndex = args.PrevLogIndex + len(args.Entries)
		rf.debug(DLog2, " After appended, lastLogIndex:%d, Logs: %v", rf.getLastLogIndex(), rf.log)
	}

	// reply
	reply.Term = rf.currentTerm
	reply.Success = true
	rf.debug(DCommit, "accepted AE RPC in T%d\n", rf.currentTerm)

	if args.LeaderCommit > rf.commitIndex {
		rf.commitIndex = min(args.LeaderCommit, rf.getLastLogIndex())
		for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
			rf.debug(
				DCommit, "applies log as a Follower at %d %v in T%d\n",
				i, rf.log[i-rf.getFirstLogIndex()], rf.currentTerm,
			)
			rf.inputChan <- ApplyMsg{
				CommandValid: true,
				Command:      rf.log[i-rf.getFirstLogIndex()].Command,
				CommandIndex: i,
			}
			rf.lastApplied = i
		}
	}
	rf.lastResetTime = time.Now()
	go func() { rf.notifyCh <- Reset }()
	rf.debug(DInfo, "accepted AE RPC (from S%d) in T%d\n", args.LeaderId, rf.currentTerm)
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
	rf.currentState = Candidate
	rf.currentTerm += 1
	rf.debug(DLeader, "timeout, starts a new election for T%d\n", rf.currentTerm)

	rf.votedFor = rf.me

	var preLogTerm int
	if len(rf.log) == 0 {
		preLogTerm = rf.snapshot.LastIncludedTerm
	} else {

		preLogTerm = rf.getTermByIndex(rf.getLastLogIndex() - rf.getFirstLogIndex())
	}

	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: rf.getLastLogIndex(),
		LastLogTerm:  preLogTerm,
	}

	voteCount := &VoteCount{data: 1, isTriggered: false}

	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			go func(i int) {
				reply := RequestVoteReply{}
				rf.sendRequestVote(i, &args, &reply)

				rf.mu.Lock()
				validVote := true
				if reply.Term < rf.currentTerm {
					validVote = false
				}
				if reply.Term > rf.currentTerm {
					rf.convertToFollower()
				}

				if validVote && reply.VoteGranted {
					voteCount.mu.Lock()
					rf.debug(DVote, "received vote from S%d (for T%d) in T%d\n", i, reply.Term, rf.currentTerm)
					voteCount.data += 1
					if voteCount.data >= rf.majority {
						if !voteCount.isTriggered {
							voteCount.isTriggered = true
							rf.debug(
								DLeader, "achieved Majority for T%d (%d), converting to leader\n", rf.currentTerm,
								rf.majority,
							)
							if rf.currentState == Candidate {
								voteCount.mu.Unlock()
								rf.mu.Unlock()
								rf.becomeLeader()
								return
							}
						}
					}
					voteCount.mu.Unlock()
				}
				rf.mu.Unlock()
			}(i)
		}
	}
}

func (rf *Raft) becomeLeader() {
	rf.mu.Lock()
	rf.currentState = Leader
	// reinitialize after election
	for i := 0; i < len(rf.peers); i++ {
		rf.nextIndex[i] = rf.getLastLogIndex() + 1
		rf.matchIndex[i] = 0
	}
	rf.mu.Unlock()
	rf.notifyCh <- WinInElection
	rf.heartbeatCh <- HeartbeatReset

}

func (rf *Raft) sendHeartBeat() {
	rf.debug(DTimer, "Leader, checking heartbeats in T%d\n", rf.currentTerm)
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			var args AppendEntriesArgs
			// check its log is up-to-date
			var preLogTerm int

			// if nextIndex is smaller than the rf.getFirstLogIndex()
			// send InstallSnapshot request
			if rf.nextIndex[i] < rf.getFirstLogIndex() {
				rf.debug(
					DSnap, "nextIndex[%d]:%d, firstLogIndex:%d, sending snapshot to S%d in T%d\n",
					i, rf.nextIndex[i], rf.getFirstLogIndex(), i, rf.currentTerm,
				)
				args := InstallSnapshotArgs{
					Term:              rf.currentTerm,
					LeaderId:          rf.me,
					LastIncludedIndex: rf.snapshot.LastIncludedIndex,
					LastIncludedTerm:  rf.snapshot.LastIncludedTerm,
					Snapshot:          rf.snapshot,
				}

				reply := InstallSnapshotReply{}

				go func(i int, args InstallSnapshotArgs, nextIndex int) {
					rf.sendInstallSnapshot(i, &args, &reply)
					rf.mu.Lock()
					defer rf.mu.Unlock()
					defer rf.persist()
					if reply.Term > rf.currentTerm {
						rf.convertToFollower()
						return
					}
					rf.nextIndex[i] = max(rf.nextIndex[i], nextIndex)
					rf.debug(DSnap, "reset nextIndex[%d] to %d\n", i, rf.nextIndex[i])
					go func() { rf.notifyCh <- Reset }()
					return
				}(i, args, args.LastIncludedIndex+1)
				continue
			}

			if len(rf.log) == 0 {
				preLogTerm = rf.snapshot.LastIncludedTerm
			} else {
				rf.debug(
					DTest, "len(rf.log):%d, rf.nextIndex[%d]:%d, rf.getFirstLogIndex():%d",
					len(rf.log), i, rf.nextIndex[i], rf.getFirstLogIndex(),
				)
				preLogTerm = rf.getTermByIndex(rf.nextIndex[i] - 1 - rf.getFirstLogIndex())
			}

			if rf.getLastLogIndex() >= rf.nextIndex[i] {
				// send AppendEntries RPC with log entries starting at nextIndex
				args = AppendEntriesArgs{
					Term:         rf.currentTerm,
					LeaderId:     rf.me,
					PrevLogIndex: rf.nextIndex[i] - 1,
					PrevLogTerm:  preLogTerm,
					// TODO: is correct?
					Entries:      rf.log[rf.nextIndex[i]-rf.getFirstLogIndex():],
					LeaderCommit: rf.commitIndex,
				}
			} else {
				// a normal heartbeat
				args = AppendEntriesArgs{
					Term:         rf.currentTerm,
					LeaderId:     rf.me,
					PrevLogIndex: rf.nextIndex[i] - 1,
					PrevLogTerm:  preLogTerm,
					Entries:      nil,
					LeaderCommit: rf.commitIndex,
				}
			}

			if args.Entries == nil {
				rf.debug(
					DLog, "-> S%d Sending heartbeat in T%d {T:%d preLogIdx:%d preLogT:%d len:%d lCommit:%d}",
					i, rf.currentTerm, args.Term, args.PrevLogIndex, args.PrevLogTerm, len(args.Entries),
					args.LeaderCommit,
				)
			} else {
				rf.debug(
					DLog, "-> S%d Sending (with heartbeat) in T%d {T:%d preLogIdx:%d preLogT:%d len:%d lCommit:%d}",
					i, rf.currentTerm, args.Term, args.PrevLogIndex, args.PrevLogTerm, len(args.Entries),
					args.LeaderCommit,
				)
			}

			go func(i int, args AppendEntriesArgs) {
				reply := AppendEntriesReply{}
				ok := rf.sendAppendEntries(i, &args, &reply)
				if ok {
					rf.mu.Lock()
					if reply.Term > rf.currentTerm {
						rf.currentTerm = reply.Term
						rf.votedFor = -1
						rf.convertToFollower()
						rf.persist()
						rf.mu.Unlock()
						rf.notifyCh <- Reset
						return
					}

					if reply.Success {
						rf.nextIndex[i] = args.PrevLogIndex + len(args.Entries) + 1
						rf.matchIndex[i] = rf.nextIndex[i] - 1
					} else {
						// optimization that backs up nextIndex by more than one entry at a time
						rejectInfo := reply.RejectionInfo
						// Case 1: follower's log is too short:
						if reply.RejectionInfo.XTerm == -1 {
							rf.nextIndex[i] = max(1, rejectInfo.XLen)
						} else {
							// Case 2: leader doesn't have XTerm:
							if !rf.containsXTerm(rejectInfo.XTerm) {
								rf.nextIndex[i] = max(1, rejectInfo.XIndex)
							} else {
								// Case 3: leader has XTerm:
								for j := len(rf.log) - 1; j >= 0; j-- {
									if rf.log[j].Term == rejectInfo.XTerm {
										rf.nextIndex[i] = j
										break
									}
								}
								rf.nextIndex[i] = max(1, rf.nextIndex[i])
							}
						}
					}
					rf.mu.Unlock()
				}
			}(i, args)
		}
	}
}

func (rf *Raft) convertToFollower() {
	if rf.currentState == Candidate {
		rf.debug(DVote, "stepping down from candidate in T%d\n", rf.currentTerm)
	}
	if rf.currentState == Leader {
		rf.debug(DVote, "stepping down from leader in T%d\n", rf.currentTerm)
	}
	rf.currentState = Follower
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
	rf.mu.Lock()
	defer rf.mu.Unlock()

	index := -1
	term := -1
	isLeader := true
	// Your code here (2B).
	if rf.currentState != Leader {
		isLeader = false
		//rf.debug(DClient, "rejected Client's request in T%d\n", rf.currentTerm)
		return index, term, isLeader
	}

	rf.debug(DClient, "accepted Client's request in T%d\n", rf.currentTerm)

	//rf.lastLogIndex++
	//rf.log[rf.lastLogIndex] = LogEntry{Index: rf.lastLogIndex, Term: rf.currentTerm, Command: command}
	logEntry := LogEntry{
		Index:   rf.getLastLogIndex() + 1,
		Term:    rf.currentTerm,
		Command: command,
	}
	rf.log = append(rf.log, logEntry)

	rf.debug(DLog, "appended log entry %v at %d in T%d\n", logEntry, len(rf.log)-1, rf.currentTerm)
	rf.debug(DLog2, " Logs: %v", rf.log)
	// rf.sendLogEntries(command)
	index = rf.getLastLogIndex()
	term = rf.currentTerm
	// rf.heartbeatCh <- HeartbeatReset
	rf.persist()
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

	// Commit checker
	go func() {
		for !rf.killed() {
			time.Sleep(50 * time.Millisecond)
			rf.mu.Lock()
			if rf.currentState == Leader {
				// try to find a new commitIndex
				tempIndex := rf.findNewCommitIndex()

				if tempIndex > rf.commitIndex && rf.log[tempIndex-rf.getFirstLogIndex()].Term == rf.currentTerm {
					rf.commitIndex = tempIndex
					rf.debug(DInfo, "lastApplied:%d, commitIndex:%d\n", rf.lastApplied, rf.commitIndex)

					for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
						rf.debug(
							DCommit, "applying log as a Leader at %d %v in T%d,[%d - %d]\n",
							i, rf.log[i-rf.getFirstLogIndex()], rf.currentTerm, i, rf.getFirstLogIndex(),
						)
						rf.inputChan <- ApplyMsg{
							CommandValid: true,
							Command:      rf.log[i-rf.getFirstLogIndex()].Command,
							CommandIndex: i,
						}
						rf.lastApplied = i
					}

					//for i := rf.lastApplied + 1; i <= rf.commitIndex; i++ {
					//	rf.mu.Unlock()
					//	rf.spMux.Lock()
					//	rf.debug(
					//		DCommit, "applying log as a Leader at %d %v in T%d,[%d - %d]\n",
					//		i, rf.log[i-rf.getFirstLogIndex()], rf.currentTerm, i, rf.getFirstLogIndex(),
					//	)
					//	rf.applyChan <- ApplyMsg{
					//		CommandValid: true,
					//		Command:      rf.log[i-rf.getFirstLogIndex()].Command,
					//		CommandIndex: i,
					//	}
					//	rf.spMux.Unlock()
					//	time.Sleep(5 * time.Millisecond)
					//	rf.mu.Lock()
					//	rf.lastApplied = i
					//}
					//rf.debug(DCommit, "finishes applying in T%d\n", rf.currentTerm)
				}
			}
			rf.persist()
			rf.mu.Unlock()
		}
	}()

	// Heartbeat
	go func() {
		for !rf.killed() {
			timeout := time.After(HeartbeatInterval)
			select {
			case <-timeout:
				rf.mu.Lock()
				if rf.currentState == Leader {
					rf.sendHeartBeat()
				}
				rf.mu.Unlock()
			case event := <-rf.heartbeatCh:
				switch event {
				case HeartbeatReset:
					rf.mu.Lock()
					rf.debug(DLeader, "heartbeat reset")
					rf.sendHeartBeat()
					rf.mu.Unlock()
				}
			}
		}
	}()

	// Election Timeout
	for !rf.killed() {
		// Your code here (2A)
		// Check if a leader election should be started.
		rf.mu.Lock()
		rf.lastResetTime = time.Now()
		rf.mu.Unlock()
		interval := time.Duration(50+rand.Intn(20)) * 10 * time.Millisecond
		timeout := time.After(interval)
		select {
		case <-timeout:
			rf.mu.Lock()
			// double check
			if time.Now().Sub(rf.lastResetTime) >= interval && rf.currentState != Leader {
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
	rf.debug(DInfo, "stopped\n")
}

func (rf *Raft) findNewCommitIndex() int {
	tempIndex := rf.commitIndex
	for {
		cnt := 1
		for i := 0; i < len(rf.peers); i++ {
			if i != rf.me && rf.matchIndex[i] > tempIndex {
				cnt++
			}
		}
		if cnt >= rf.majority {
			tempIndex++
		} else {
			break
		}
	}
	return tempIndex
}

type InstallSnapshotArgs struct {
	Term              int
	LeaderId          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Snapshot          Snapshot
}

func (is *InstallSnapshotArgs) String() string {
	return fmt.Sprintf(
		"T:%d, LII:%d, LIT:%d, Snapshot:%v", is.Term, is.LastIncludedIndex, is.LastIncludedTerm,
		is.Snapshot,
	)
}

type InstallSnapshotReply struct {
	Term int
}

func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	ok := rf.peers[server].Call("Raft.InstallSnapshot", args, reply)
	return ok
}

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	defer rf.persist()

	if args.Term < rf.currentTerm {
		reply.Term = rf.currentTerm
		return
	}

	rf.debug(DSnap, "received IS RPC %v in T%d\n", args, rf.currentTerm)

	// out of order
	if args.LastIncludedIndex <= rf.snapshot.LastIncludedIndex {
		rf.debug(
			DSnap, "rejected it since args.LastIncludedIndex:%d < rf.snapshot.LastIncludedIndex:%d\n",
			args.LastIncludedIndex, rf.snapshot.LastIncludedIndex,
		)
		return
	}

	if args.LastIncludedIndex < rf.commitIndex {
		rf.debug(
			DSnap, "rejected it since args.LastIncludedIndex:%d < rf.commitIndex:%d\n", args.LastIncludedIndex,
			rf.commitIndex,
		)
		return
	}

	if args.LastIncludedIndex <= rf.getLastLogIndex() &&
		rf.getTermByIndex(args.LastIncludedIndex-rf.getFirstLogIndex()) == args.LastIncludedTerm {
		// find the log entry that has same index and term as snapshot's last included entry
		rf.debug(
			DSnap, "accepted the snapshot(args.LastIncludedIndex:%d <= rf.getLastLogIndex():%d), rf.log:%v\n",
			args.LastIncludedIndex, rf.getLastLogIndex(), rf.log,
		)

		if rf.commitIndex < args.LastIncludedIndex {
			for i := rf.commitIndex + 1; i <= args.LastIncludedIndex; i++ {
				rf.debug(
					DCommit, "applies log as a Follower at %d %v in T%d\n",
					i, rf.log[i-rf.getFirstLogIndex()], rf.currentTerm,
				)
				rf.inputChan <- ApplyMsg{
					CommandValid: true,
					Command:      rf.log[i-rf.getFirstLogIndex()].Command,
					CommandIndex: i,
				}
				rf.lastApplied = i
			}
		}

		rf.log = rf.log[args.LastIncludedIndex-rf.getFirstLogIndex()+1:]
		rf.snapshot = args.Snapshot
		// TODO: correct?
		//rf.commitIndex = args.LastIncludedIndex
		//rf.lastApplied = args.LastIncludedIndex
		go func() { rf.notifyCh <- Reset }()
		return
	} else {
		// discard the entire log
		rf.log = make([]LogEntry, 0)
		rf.snapshot = args.Snapshot
		rf.commitIndex = args.LastIncludedIndex
		rf.lastApplied = args.LastIncludedIndex
		rf.debug(DSnap, "accepted the snapshot, rf.log:%v\n", rf.log)
		rf.inputChan <- ApplyMsg{
			SnapshotValid: true,
			Snapshot:      rf.snapshot.Data,
			SnapshotIndex: rf.snapshot.LastIncludedIndex,
			SnapshotTerm:  rf.snapshot.LastIncludedTerm,
		}
		go func() { rf.notifyCh <- Reset }()
		return
	}

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
	rf := &Raft{
		peers:     peers,
		persister: persister,
		me:        me,
		dead:      0,
		applyChan: applyCh,

		// Initialization code
		currentState: Follower,
		currentTerm:  0,
		votedFor:     -1,
		log:          make([]LogEntry, 1),
		commitIndex:  0,
		lastApplied:  0,
		nextIndex:    make([]int, len(peers)),
		matchIndex:   make([]int, len(peers)),
		majority:     len(peers)/2 + 1,
		notifyCh:     make(chan Event),
		heartbeatCh:  make(chan Event),
		snapshot: Snapshot{
			LastIncludedIndex: 0,
			LastIncludedTerm:  0,
			Data:              nil,
		},
		inputChan: make(chan ApplyMsg),
	}

	rf.debug(DInfo, "starting...\n")

	// Your initialization code here (2A, 2B, 2C).
	go ApplyMsgHelper(rf.inputChan, rf.applyChan)

	rand.Seed(time.Now().Unix())

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState(), persister.ReadSnapshot())

	// start ticker goroutine to start elections
	go rf.ticker()

	return rf
}

func (rf *Raft) debug(topic LogTopic, format string, a ...interface{}) {
	prefix := fmt.Sprintf("S%d ", rf.me)
	Debug(topic, prefix+format, a...)
}

func (rf *Raft) containsXTerm(XTerm int) bool {
	for i := 0; i < len(rf.log); i++ {
		if rf.log[i].Term == XTerm {
			return true
		}
	}
	return false
}

func (rf *Raft) getTermByIndex(index int) int {
	if index < -1 {
		log.Panicf("getTermByIndex index:%d", index)
	}
	if index == -1 || len(rf.log) == 0 {
		return rf.snapshot.LastIncludedTerm
	}
	return rf.log[index].Term
}

func (rf *Raft) getFirstLogIndex() int {
	if len(rf.log) == 0 {
		return rf.snapshot.LastIncludedIndex
	} else {
		return rf.log[0].Index
	}
}

func (rf *Raft) getLastLogIndex() int {
	if len(rf.log) == 0 {
		if rf.snapshot.LastIncludedIndex == 0 {
			// there is no snapshot
			return 0
		} else {
			return rf.snapshot.LastIncludedIndex
		}
	} else {
		return rf.log[len(rf.log)-1].Index
	}
}

func Trunc(data any, length int) string {
	truncated := ""
	count := 0
	if data == nil {
		return "<nil>"
	}
	var str string
	if i, ok := data.(int); ok {
		str = strconv.Itoa(i)
	}
	if s, ok := data.(string); ok {
		str = s
	}
	for _, char := range str {
		truncated += string(char)
		count++
		if count >= length {
			break
		}
	}
	return truncated
}

func ApplyMsgHelper(input chan ApplyMsg, output chan ApplyMsg) {
	q := make([]ApplyMsg, 0)

	for input != nil || len(q) > 0 {
		var sendOut chan ApplyMsg
		var next ApplyMsg
		if len(q) > 0 {
			sendOut = output
			next = q[0]
		}

		select {
		case msg, ok := <-input:
			if !ok {
				input = nil
				continue
			}
			q = append(q, msg)
		case sendOut <- next:
			q = q[1:]
		}
	}
}
