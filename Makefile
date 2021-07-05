PKGS=$(shell ls -d *)

check:
	@echo "Checking...\n"
	gocyclo -over 15 . || echo -n ""
	@echo ""
	golangci-lint run -E misspell -E depguard -E dupl -E goconst -E gocyclo -E ifshort -E predeclared -E tagliatelle
	golint -min_confidence 0.21 -set_exit_status ./...
	@echo "\nAll ok!"