package kvraft

import (
	"fmt"
	"strconv"
)

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeout     = "ErrTimeout"
)

type Err string

type Args interface {
}

type ClientInfo struct {
	ClientId int64
	SeqNum   int
}

func (ci ClientInfo) String() string {
	return fmt.Sprintf("%v - %d", Trunc(ci.ClientId, 3), ci.SeqNum)
}

type Reply interface {
	Error() string
}

// Put or Append
type PutAppendArgs struct {
	ClientInfo
	Key   string
	Value string
	Op    string // "Put" or "Append"
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
}

func (pa PutAppendArgs) Args() {
}

type PutAppendReply struct {
	Err Err
}

func (pr PutAppendReply) Error() string {
	return string(pr.Err)
}

type GetArgs struct {
	ClientInfo
	Key string
	// You'll have to add definitions here.
}

func (ga GetArgs) Args() {
}

type GetReply struct {
	Err   Err
	Value string
}

func (gr GetReply) Error() string {
	return string(gr.Err)
}

func Trunc(data any, length int) string {
	truncated := ""
	count := 0
	if data == nil {
		return "<nil>"
	}
	var str string
	if i, ok := data.(int64); ok {
		str = strconv.Itoa(int(i))
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
