go test -run InitialElection | dslogs

go test -run InitialElection > output.log

dslogs output.log -c 3 -i TIMR

VERBOSE=1 go test -run 2A 


VERBOSE=1 go test -run ManyElections > output.log

VERBOSE=1 go test -run ReElection > output.log

dstest -p 5 ManyElections

dslogs output.log -c 7 -i TIMR


dstest  -p 5 -n 30 -o .run ManyElections 


cat test_test.go| grep 2A | sed 's\(\ \g'|awk '/func/ {printf "%s ",$2;}' | xargs dstest -p 4 -o .run -v 1 -r  -s


cat test_test.go|grep 'func Test'> test.txt


dstest  -p 50 -n 100 -o .run TestInitialElection2A TestReElection2A TestManyElections2A


## 2B

### Task List

TestBasicAgree2B
TestRPCBytes2B
TestFollowerFailure2B
TestLeaderFailure2B
TestFailAgree2B
TestFailNoAgree2B
TestConcurrentStarts2B
TestRejoin2B
TestBackup2B
TestCount2B

VERBOSE=1 go test -run TestBasicAgree2B > output.log

go test -run TestBasicAgree2B

dslogs output.log -c 3

go test -run 2B

go test -run TestBasicAgree2B OK

go test -run TestRPCBytes2B OK

go test -run TestFollowerFailure2B OK

go test -run TestLeaderFailure2B OK

go test -run TestFailAgree2B OK
VERBOSE=1 go test -run TestFailAgree2B > output.log

dstest  -p 50 -n 1000 -o .run  TestFailAgree2B

go test -run TestFailNoAgree2B OK

go test -run TestConcurrentStarts2B
VERBOSE=1 go test -run TestConcurrentStarts2B > output.log
dstest  -p 50 -n 100 -o .run TestConcurrentStarts2B

go test -run TestRejoin2B OK
dstest  -p 50 -n 1000 -o .run TestRejoin2B

go test -run TestBackup2B OK
dstest  -p 50 -n 100 -o .run TestBackup2B

go test -run TestCount2B OK

### For dstest

#### All

dstest  -p 50 -n 100 -o run_logs TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B TestFailAgree2B TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B

dstest  -p 50 -n 100 -o run_logs TestInitialElection2A TestReElection2A TestManyElections2A TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B TestFailAgree2B TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B


dstest  -p 50 -n 100 -o run_logs TestManyElections2A

go test -race -run TestManyElections2A

dstest -r -p 50 -l -o run_logs TestManyElections2A

#### some
dstest  -p 50 -n 100 -o run_logs  TestFailAgree2B TestFailNoAgree2B TestRejoin2B


Failed test TestManyElections2A - run_logs/TestManyElections2A_224.log
Failed test TestManyElections2A - run_logs/TestManyElections2A_266.log
Failed test TestManyElections2A - run_logs/TestManyElections2A_405.log
Failed test TestManyElections2A - run_logs/TestManyElections2A_478.log
Failed test TestManyElections2A - run_logs/TestManyElections2A_506.log
Failed test TestManyElections2A - run_logs/TestManyElections2A_507.log
Failed test TestManyElections2A - run_logs/TestManyElections2A_511.log
Failed test TestManyElections2A - run_logs/TestManyElections2A_515.log
Failed test TestManyElections2A - run_logs/TestManyElections2A_517.log
