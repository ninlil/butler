PKGS=$(shell ls -d *)
TAG=$(shell cat release.txt)

check:
	@echo "Checking...\n"
	gocyclo -over 15 . || echo -n ""
	@echo ""
	golangci-lint run -E misspell -E depguard -E dupl -E goconst -E gocyclo -E predeclared -E tagliatelle -E errorlint -E godox -E unparam
#	golint -min_confidence 0.21 -set_exit_status ./...
	@echo "\nAll ok!"

release:
	gh release create $(TAG) -t $(TAG)

check2:
	@echo "Checking...\n"
	gocyclo -over 15 . || echo -n ""
	@echo ""
	golint -min_confidence 0.21 -set_exit_status ./...
	@echo "\nAll ok!"

check2:
	golangci-lint run -E misspell -E depguard -E dupl -E goconst -E gocyclo -E predeclared -E tagliatelle -E errorlint -E godox -D structcheck
	@echo ""

updates:
	go list -m -u all
