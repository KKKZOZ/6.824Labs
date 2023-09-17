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


dstest  -p 100 -n 400 -o .run TestInitialElection2A TestReElection2A TestManyElections2A