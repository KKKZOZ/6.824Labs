package kvraft

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"

	. "6.5840/debug"
	"6.5840/labrpc"
)

type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	mu           sync.Mutex
	lastLeaderId int
	clientId     int64
	seqNum       int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.clientId = nrand()
	ck.lastLeaderId = -1
	return ck
}

func (ck *Clerk) sendRequest(method string, argsT any) Reply {
	//ck.mu.Lock()
	var lastLeaderId int
	if ck.lastLeaderId != -1 {
		lastLeaderId = ck.lastLeaderId
	} else {
		lastLeaderId = 0
	}
	//ck.mu.Unlock()
	var replyT Reply

	for {
		var ok bool
		if method == "Get" {
			args, _ := argsT.(GetArgs)
			reply := GetReply{}
			ok = ck.servers[lastLeaderId].Call("KVServer."+method, &args, &reply)
			replyT = reply
		} else {
			args, _ := argsT.(PutAppendArgs)
			reply := PutAppendReply{}
			ok = ck.servers[lastLeaderId].Call("KVServer."+method, &args, &reply)
			replyT = reply
		}

		if !ok ||
			replyT.Error() == ErrWrongLeader ||
			replyT.Error() == ErrTimeout {
			ck.debug(SRequest, "failed (method: %v, args: %v) because of %v, resending\n", replyT.Error(), method, argsT)
			lastLeaderId = (lastLeaderId + 1) % len(ck.servers)
		} else {
			//ck.mu.Lock()
			ck.lastLeaderId = lastLeaderId
			//ck.mu.Unlock()
			break
		}
	}

	return replyT
}

// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) Get(key string) string {
	// You will have to modify this function.
	args := GetArgs{ck.makeInfo(), key}
	// reply := ck.sendRequest("Get", args).(GetReply)
	reply := GetReply{}

	reply = ck.sendRequest("Get", args).(GetReply)

	if reply.Err == OK {
		return reply.Value
	} else {
		return ""
	}
}

// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.
	args := PutAppendArgs{ck.makeInfo(), key, value, op}
	_ = ck.sendRequest("PutAppend", args).(PutAppendReply)

}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}

func (ck *Clerk) makeInfo() ClientInfo {
	ck.mu.Lock()
	defer ck.mu.Unlock()

	ck.seqNum++
	return ClientInfo{ck.clientId, ck.seqNum}
}

func (ck *Clerk) debug(topic LogTopic, format string, a ...interface{}) {
	prefix := fmt.Sprintf("S%d ", ck.clientId)
	Debug(topic, prefix+format, a...)
}
