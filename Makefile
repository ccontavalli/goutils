
format:
	for dir in *; do test -d "$$dir" && ( cd "$$dir"; go fmt; ); done;

test:
	for dir in *; do test -d "$$dir" && ( cd "$$dir"; go test -cover; ); done;
