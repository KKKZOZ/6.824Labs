package kvraft

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	. "6.5840/debug"
	"6.5840/labgob"
	"6.5840/labrpc"
	"6.5840/raft"
)

type OpType string

const (
	GET    OpType = "Get"
	PUT    OpType = "Put"
	APPEND OpType = "Append"
)

const TIMEOUT = 3000

type Op struct {
	OpType   OpType
	Key      string
	Value    string
	ClientId int64
	SeqNum   int
}

type Snapshot struct {
	KVMap    map[string]string
	DupTable map[int64]OperationContext
}

type OperationContext struct {
	SeqNum int
	Value  string
}

type ResultMsg struct {
	Index  int
	Result string
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	persister      *raft.Persister
	kvMap          map[string]string
	broker         *Broker[raft.ApplyMsg]
	lastOperations map[int64]OperationContext
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	kv.mu.Lock()

	if kv.isDuplicate(args.ClientId, args.SeqNum) {
		reply.Err, reply.Value = OK, kv.lastOperations[args.ClientId].Value
		kv.mu.Unlock()
		kv.debug(
			SResponse, "[Duplicate] Get response (ClientInfo: %v): [K: %v V: %v]\n", args.ClientInfo, args.Key,
			reply.Value,
		)
		return
	}
	kv.mu.Unlock()

	command := Op{OpType: GET, Key: args.Key, ClientId: args.ClientId, SeqNum: args.SeqNum}
	index, curTerm, isLeader := kv.rf.Start(command)

	kv.debug(SRequest, "starts Get request (ClientInfo: %v): [K: %v]\n", args.ClientInfo, args.Key)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	msgChan := kv.broker.Subscribe()
	defer kv.broker.Unsubscribe(msgChan)

	for {
		select {
		case <-time.After(TIMEOUT * time.Millisecond):
			reply.Err = ErrTimeout
			return
		case applyMsg := <-msgChan:
			if !applyMsg.CommandValid || applyMsg.CommandIndex != index {
				continue
			}
			if applyMsg.CommandTerm != curTerm {
				reply.Err = ErrWrongLeader
				return
			}

			kv.mu.Lock()
			op := applyMsg.Command.(Op)
			reply.Err = OK
			reply.Value = kv.kvMap[op.Key]
			kv.debug(
				SResponse, "Get response (ClientInfo: %v): [K: %v V: %v]\n", args.ClientInfo, args.Key, reply.Value,
			)
			kv.mu.Unlock()
			return

		}
	}

}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	kv.mu.Lock()
	if kv.isDuplicate(args.ClientId, args.SeqNum) {
		reply.Err = OK
		kv.mu.Unlock()
		kv.debug(
			SResponse, "[Duplicate] PutAppend response (ClientInfo: %v): %v, %v\n", args.ClientInfo, args.Key,
			args.Value,
		)
		return
	}
	kv.mu.Unlock()

	command := Op{
		OpType: OpType(args.Op),
		Key:    args.Key, Value: args.Value,
		ClientId: args.ClientId, SeqNum: args.SeqNum,
	}

	index, curTerm, isLeader := kv.rf.Start(command)
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}
	kv.debug(SRequest, "starts PutAppend request (ClientInfo: %v): %v\n", args.ClientInfo, args.Key)

	msgChan := kv.broker.Subscribe()
	defer kv.broker.Unsubscribe(msgChan)

	for {
		select {
		case <-time.After(TIMEOUT * time.Millisecond):
			reply.Err = ErrTimeout
			return
		case applyMsg := <-msgChan:
			if !applyMsg.CommandValid || applyMsg.CommandIndex != index {
				continue
			}
			if applyMsg.CommandTerm != curTerm {
				reply.Err = ErrWrongLeader
				return
			}

			reply.Err = OK
			kv.debug(SResponse, "PutAppend response (ClientInfo: %v): %v\n", args.ClientInfo, args.Key)
			return
		}
	}
}

// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

func (kv *KVServer) applyPub() {
	for !kv.killed() {
		applyMsg := <-kv.applyCh

		if applyMsg.SnapshotValid {
			kv.restoreSnapshot(applyMsg.Snapshot)
			continue
		}

		kv.mu.Lock()
		// var response string
		op := applyMsg.Command.(Op)
		// if it is a duplicate message
		// return as it has been processed in the first time
		if !applyMsg.CommandValid || kv.isDuplicate(op.ClientId, op.SeqNum) {
			kv.mu.Unlock()
			// response = kv.lastOperations[op.ClientId].Value
			kv.broker.Publish(applyMsg)
			continue
		} else {
			_ = kv.processCommand(op)
		}

		if curTerm, isLeader := kv.rf.GetState(); isLeader &&
			applyMsg.CommandTerm == curTerm {
			kv.broker.Publish(applyMsg)
		}
		kv.checkRaftStateSize(applyMsg.CommandIndex)
		kv.mu.Unlock()
	}
}

func (kv *KVServer) processCommand(op Op) string {
	switch op.OpType {
	case GET:
	case PUT:
		kv.kvMap[op.Key] = op.Value
	case APPEND:
		kv.kvMap[op.Key] += op.Value
	}
	kv.lastOperations[op.ClientId] = OperationContext{SeqNum: op.SeqNum, Value: kv.kvMap[op.Key]}
	return kv.kvMap[op.Key]
}

func (kv *KVServer) restoreSnapshot(data []byte) {
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var snapshot Snapshot
	if err := d.Decode(&snapshot); err != nil {
		panic(err)
	} else {
		kv.mu.Lock()
		kv.kvMap = snapshot.KVMap
		kv.lastOperations = snapshot.DupTable
		kv.mu.Unlock()
	}
}

func (kv *KVServer) checkRaftStateSize(index int) {
	if kv.maxraftstate != -1 && kv.persister.RaftStateSize() >= kv.maxraftstate {
		snapshot := &Snapshot{kv.kvMap, kv.lastOperations}

		w := new(bytes.Buffer)
		e := labgob.NewEncoder(w)
		err := e.Encode(snapshot)
		if err != nil {
			panic(err)
		}
		data := w.Bytes()
		kv.rf.Snapshot(index, data)
	}
}

// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.kvMap = make(map[string]string)
	kv.broker = NewBroker[raft.ApplyMsg]()
	kv.lastOperations = make(map[int64]OperationContext)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	kv.persister = persister

	// You may need initialization code here.

	spBytes := persister.ReadSnapshot()
	if len(spBytes) > 0 {
		r := bytes.NewBuffer(spBytes)
		d := labgob.NewDecoder(r)
		var snapshot Snapshot
		if err := d.Decode(&snapshot); err != nil {
			panic(err)
		} else {
			kv.kvMap = snapshot.KVMap
			kv.lastOperations = snapshot.DupTable
		}
	}

	go kv.broker.Start()
	go kv.applyPub()

	return kv
}

func (kv *KVServer) isDuplicate(clientId int64, seqNum int) bool {
	entry, ok := kv.lastOperations[clientId]
	if ok && entry.SeqNum >= seqNum {
		return true
	}
	return false
}

func (kv *KVServer) debug(topic LogTopic, format string, a ...interface{}) {
	prefix := fmt.Sprintf("S%d ", kv.me)
	Debug(topic, prefix+format, a...)
}
