
format:
	for dir in `find . -type f -name '*.go' -printf '%h\n' |uniq | grep -v '/vendor/'`; do ( cd "$$dir"; go fmt; ); done;

test:
	for dir in `find . -type f -name '*_test.go' -printf '%h\n' |uniq | grep -v '/vendor/'`; do test -d "$$dir" && ( cd "$$dir"; go test -cover; ); done;
