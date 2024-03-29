go test -run InitialElection | dslogs

go test -run InitialElection > output.log

dslogs output.log -c 3 -i TIMR

VERBOSE=1 go test -run 2A 


VERBOSE=1 go test -run ManyElections > output.log

VERBOSE=1 go test -run ReElection > output.log

dstest -p 5 ManyElections

dslogs output.log -c 7 -i TIMR


dstest  -p 5 -n 30 -o .run ManyElections 





cat test_test.go|grep 'func Test'> test.txt


dstest  -p 32 -n 100 -o run_logsA TestInitialElection2A TestReElection2A TestManyElections2A

dstest  -p 100 -n 1000 -o run_logsA TestInitialElection2A

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

VERBOSE=1 go test -run TestInitialElection2A > output.log

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

dstest -p 32 -n 100 -o run_logsB TestBasicAgree2B

go test -timeout 2s -run=TestBasicAgree2B


dstest -p 50 -n 1000 -o run_logs2B TestBackup2B

dstest  -p 100 -n 1000 -o run_logs2B TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B TestFailAgree2B TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B

dstest  -p 50 -n 1000 -o run_logsAB TestInitialElection2A TestReElection2A TestManyElections2A TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B TestFailAgree2B TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B


dstest  -p 50 -n 100 -o run_logs TestManyElections2A

go test -race -run TestManyElections2A

dstest -r -p 50 -l -o run_logs TestManyElections2A

#### some
dstest  -p 50 -n 100 -o run_logsB  TestFailAgree2B TestFailNoAgree2B TestRejoin2B


## 2C

### Task List

TestPersist12C OK
TestPersist22C OK
TestPersist32C OK

TestFigure82C
dstest  -p 100 -n 200 -o run_logs3 TestFigure82C


TestUnreliableAgree2C OK


TestFigure8Unreliable2C
dstest  -p 100 -n 3000 -o run_logs2 TestFigure8Unreliable2C

TestReliableChurn2C
TestUnreliableChurn2C 


dstest  -p 32 -n 1000 -o run_logsC TestPersist12C  TestPersist22C  TestPersist32C TestFigure82C TestUnreliableAgree2C

dstest  -p 32 -n 2000 -o run_logsC TestUnreliableChurn2C

#### All
dstest  -p 50 -n 50 -o run_logsAll TestInitialElection2A TestReElection2A TestManyElections2A TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B TestFailAgree2B TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B  TestPersist12C  TestPersist22C  TestPersist32C  TestFigure82C TestUnreliableAgree2C TestFigure8Unreliable2C  TestReliableChurn2C TestUnreliableChurn2C



dstest  -p 50 -n 500 -o run_logsAll TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B TestFailAgree2B TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B  TestPersist12C  TestPersist22C  TestPersist32C  TestFigure82C TestUnreliableAgree2C TestFigure8Unreliable2C  TestReliableChurn2C TestUnreliableChurn2C




## 2D
### TaskList

TestSnapshotBasic2D

VERBOSE=1 go test -run TestSnapshotBasic2D > output.log

dstest  -p 32 -n 960 -o run_logsD TestSnapshotBasic2D

dstest  -p 50 -n 960 -o run_logsD TestSnapshotBasic2D

TestSnapshotInstall2D
VERBOSE=1 go test -run TestSnapshotInstall2D > output.log

dstest -p 10 -n 10 -o run_logsD TestSnapshotInstall2D


TestSnapshotInstallUnreliable2D
dstest -p 50 -n 100 -o run_logsD TestSnapshotInstallUnreliable2D

TestSnapshotInstallCrash2D
VERBOSE=1 go test -run TestSnapshotInstallCrash2D > output.log
TestSnapshotInstallUnCrash2D
TestSnapshotAllCrash2D
TestSnapshotInit2D



dstest  -p 50 -n 100 -o run_logs2D TestSnapshotBasic2D TestSnapshotInstall2D TestSnapshotInstallUnreliable2D TestSnapshotInstallCrash2D TestSnapshotInstallUnCrash2D TestSnapshotAllCrash2D TestSnapshotInit2D

dstest  -p 36 -n 100 -o run_logs2CD TestPersist12C  TestPersist22C  TestPersist32C  TestFigure82C TestUnreliableAgree2C TestFigure8Unreliable2C  TestReliableChurn2C TestUnreliableChurn2C TestSnapshotBasic2D TestSnapshotInstall2D TestSnapshotInstallUnreliable2D TestSnapshotInstallCrash2D TestSnapshotInstallUnCrash2D TestSnapshotAllCrash2D TestSnapshotInit2D

dstest  -p 36 -n 20 -o run_logs2CD TestPersist12C  TestPersist22C  TestPersist32C  TestFigure82C TestUnreliableAgree2C TestFigure8Unreliable2C  TestReliableChurn2C TestUnreliableChurn2C TestSnapshotBasic2D TestSnapshotInstall2D TestSnapshotInstallUnreliable2D TestSnapshotInstallCrash2D TestSnapshotInstallUnCrash2D TestSnapshotAllCrash2D TestSnapshotInit2D


#### All
dstest  -p 50 -n 10 -o run_logsAll TestInitialElection2A TestReElection2A TestManyElections2A TestBasicAgree2B TestRPCBytes2B TestFollowerFailure2B TestLeaderFailure2B TestFailAgree2B TestFailNoAgree2B TestConcurrentStarts2B TestRejoin2B TestBackup2B TestCount2B  TestPersist12C  TestPersist22C  TestPersist32C  TestFigure82C TestUnreliableAgree2C TestFigure8Unreliable2C  TestReliableChurn2C TestUnreliableChurn2C TestSnapshotBasic2D TestSnapshotInstall2D TestSnapshotInstallUnreliable2D TestSnapshotInstallCrash2D TestSnapshotInstallUnCrash2D TestSnapshotAllCrash2D TestSnapshotInit2D 




## 3A
### TaskList
grep "^func Test" test_test.go > tasklist.txt

TestBasic3A

VERBOSE=1 go test -run TestBasic3A > output.log

dstest  -p 50 -n 100 -o run_logsA TestBasic3A

TestSpeed3A

VERBOSE=1 go test -run TestSpeed3A > output.log

dstest -p 32 -n 64 -o run_logsA TestSpeed3A


TestConcurrent3A



TestUnreliable3A

VERBOSE=1 go test -run TestUnreliable3A > output.log

TestUnreliableOneKey3A
TestOnePartition3A

VERBOSE=2 go test -run TestOnePartition3A > output.log

VERBOSE=1 go test -run TestOnePartition3A > output.log

TestManyPartitionsOneClient3A

VERBOSE=2 go test -run TestManyPartitionsOneClient3A > output.log

TestManyPartitionsManyClients3A

VERBOSE=2 go test -run TestManyPartitionsManyClients3A > output.log


TestPersistOneClient3A

VERBOSE=2 go test -run TestPersistOneClient3A > output.log

TestPersistConcurrent3A
TestPersistConcurrentUnreliable3A
TestPersistPartition3A
TestPersistPartitionUnreliable3A
TestPersistPartitionUnreliableLinearizable3A

dstest -p 32 -n 20 -o run_logs3A TestBasic3A TestSpeed3A TestConcurrent3A TestUnreliable3A TestUnreliableOneKey3A TestOnePartition3A TestManyPartitionsOneClient3A TestManyPartitionsManyClients3A TestPersistOneClient3A TestPersistConcurrent3A TestPersistConcurrentUnreliable3A TestPersistPartition3A TestPersistPartitionUnreliable3A TestPersistPartitionUnreliableLinearizable3A

dstest -p 32 -n 20 -o run_logs3A TestBasic3A TestConcurrent3A TestUnreliable3A TestUnreliableOneKey3A TestOnePartition3A TestManyPartitionsOneClient3A TestManyPartitionsManyClients3A TestPersistOneClient3A TestPersistConcurrent3A TestPersistConcurrentUnreliable3A TestPersistPartition3A TestPersistPartitionUnreliable3A TestPersistPartitionUnreliableLinearizable3A

dstest -p 32 -n 20 -o run_logs3A TestBasic3A TestConcurrent3A TestUnreliable3A TestUnreliableOneKey3A TestOnePartition3A TestManyPartitionsOneClient3A TestManyPartitionsManyClients3A 


dstest -p 64 -n 200 -o run_logs3A TestPersistConcurrent3A TestPersistPartition3A

## 3B
TestSnapshotRPC3B
VERBOSE=2 go test -run TestSnapshotRPC3B > output.log


TestSnapshotSize3B
TestSpeed3B
TestSnapshotRecover3B
TestSnapshotRecoverManyClients3B

dstest -p 32 -n 32 -o run_logsB  TestSnapshotRecover3B 
go test -run TestSnapshotRecoverManyClients3B

TestSnapshotUnreliable3B
TestSnapshotUnreliableRecover3B
TestSnapshotUnreliableRecoverConcurrentPartition3B
TestSnapshotUnreliableRecoverConcurrentPartitionLinearizable3B


dstest -p 32 -n 32 -o run_logsB TestSnapshotUnreliable3B TestSnapshotUnreliableRecover3B TestSnapshotUnreliableRecoverConcurrentPartition3B TestSnapshotUnreliableRecoverConcurrentPartitionLinearizable3B


dstest -p 32 -n 32 -o run_logsB  TestSnapshotUnreliableRecoverConcurrentPartitionLinearizable3B

dstest -p 32 -n 32 -o run_logsB  TestSnapshotSize3B

### All
dstest -p 32 -n 32 -o run_logsAll TestSnapshotRPC3B TestSnapshotSize3B TestSnapshotRecover3B TestSnapshotRecoverManyClients3B TestSnapshotUnreliable3B TestSnapshotUnreliableRecover3B TestSnapshotUnreliableRecoverConcurrentPartition3B TestSnapshotUnreliableRecoverConcurrentPartitionLinearizable3B

***exclude TestSpeed3A,TestSnapshotSize3B since it needs speed***

dstest -p 32 -n 20 -o run_logsAll TestBasic3A  TestConcurrent3A TestUnreliable3A TestUnreliableOneKey3A TestOnePartition3A TestManyPartitionsOneClient3A TestManyPartitionsManyClients3A TestPersistOneClient3A TestPersistConcurrent3A TestPersistConcurrentUnreliable3A TestPersistPartition3A TestPersistPartitionUnreliable3A TestPersistPartitionUnreliableLinearizable3A TestSnapshotRPC3B TestSnapshotRecover3B TestSnapshotRecoverManyClients3B TestSnapshotUnreliable3B TestSnapshotUnreliableRecover3B TestSnapshotUnreliableRecoverConcurrentPartition3B TestSnapshotUnreliableRecoverConcurrentPartitionLinearizable3B