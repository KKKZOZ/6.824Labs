package shardctrler

import (
	"fmt"
	"sort"
	"strconv"
	"sync"

	. "6.5840/debug"
	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raft"
)

type TableEntry struct {
	SeqNum int
	Value  string
}

type ShardCtrler struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	// Your data here.

	configs []Config // indexed by config num

	broker   *Broker[raft.ApplyMsg]
	dupTable map[int64]TableEntry
}

type Op struct {
	ClientId int64
	SeqNum   int
	Servers  map[int][]string // Join
	GIDs     []int            // Leave
	Shard    int              // Move
	GID      int              // Move
	Num      int              // Query
}

func (sc *ShardCtrler) Join(args *JoinArgs, reply *JoinReply) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.isDuplicate(args.ClientId, args.SeqNum) {
		reply.Err = OK
		sc.debug(SResponse, "[Duplicate] Move request")
		return
	}

}

func (sc *ShardCtrler) Leave(args *LeaveArgs, reply *LeaveReply) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.isDuplicate(args.ClientId, args.SeqNum) {
		reply.Err = OK
		sc.debug(SResponse, "[Duplicate] Move request")
		return
	}
}

func (sc *ShardCtrler) Move(args *MoveArgs, reply *MoveReply) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.isDuplicate(args.ClientId, args.SeqNum) {
		reply.Err = OK
		sc.debug(SResponse, "[Duplicate] Move request")
		return
	}

	// lastConfig := sc.configs[len(sc.configs)-1]
	// newConfig := Config{len(sc.configs), lastConfig.Shards, deepCopy(lastConfig.Groups)}

	// newConfig.Shards[args.Shard] = args.GID
	// sc.configs = append(sc.configs, newConfig)

	// sc.debug(SConfig, "New config: %v\n", newConfig)
}

func (sc *ShardCtrler) Query(args *QueryArgs, reply *QueryReply) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.isDuplicate(args.ClientId, args.SeqNum) {
		reply.Err = OK
		index, err := strconv.Atoi(sc.dupTable[args.ClientId].Value)
		if err != nil {
			sc.debug(SResponse, "[Duplicate] Query request Error: %v\n", err)
		}
		reply.Config = sc.configs[index]
		sc.debug(SResponse, "[Duplicate] Query request\n")
		return
	}

	if args.Num >= 0 && args.Num < len(sc.configs) {
		reply.Config = sc.configs[args.Num]
	} else {
		reply.Config = sc.configs[len(sc.configs)-1]
	}
	reply.Err = OK
	sc.debug(SResponse, "Query response: %v\n", reply.Config)
}

func GetGIDWithMinimumShards(s2g map[int][]int) int {
	var keys []int
	for k := range s2g {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	// find GID with minimum shards
	index, min := -1, NShards+1
	for _, gid := range keys {
		if gid != 0 && len(s2g[gid]) < min {
			index, min = gid, len(s2g[gid])
		}
	}
	return index
}

func GetGIDWithMaximumShards(s2g map[int][]int) int {
	// always choose gid 0 if there is any
	if shards, ok := s2g[0]; ok && len(shards) > 0 {
		return 0
	}
	var keys []int
	for k := range s2g {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	// find GID with maximum shards
	index, max := -1, -1
	for _, gid := range keys {
		if len(s2g[gid]) > max {
			index, max = gid, len(s2g[gid])
		}
	}
	return index
}

// the tester calls Kill() when a ShardCtrler instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
func (sc *ShardCtrler) Kill() {
	sc.rf.Kill()
	// Your code here, if desired.
}

// needed by shardkv tester
func (sc *ShardCtrler) Raft() *raft.Raft {
	return sc.rf
}

// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant shardctrler service.
// me is the index of the current server in servers[].
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardCtrler {
	sc := new(ShardCtrler)
	sc.me = me

	sc.configs = make([]Config, 1)
	sc.configs[0].Groups = map[int][]string{}

	labgob.Register(Op{})
	sc.applyCh = make(chan raft.ApplyMsg)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh)

	// Your code here.
	sc.broker = NewBroker[raft.ApplyMsg]()
	sc.dupTable = make(map[int64]TableEntry)

	go sc.broker.Start()
	// go sc.applyPub()

	return sc
}

func (sc *ShardCtrler) isDuplicate(clientId int64, seqNum int) bool {
	entry, ok := sc.dupTable[clientId]
	if ok && entry.SeqNum >= seqNum {
		return true
	}
	return false
}

func (sc *ShardCtrler) debug(topic LogTopic, format string, a ...interface{}) {
	prefix := fmt.Sprintf("S%d ", sc.me)
	Debug(topic, prefix+format, a...)
}

func deepCopy(groups map[int][]string) map[int][]string {
	newGroups := make(map[int][]string)
	for k, v := range groups {
		newSlice := make([]string, len(v))
		copy(newSlice, v)
		newGroups[k] = newSlice
	}
	return newGroups
}
