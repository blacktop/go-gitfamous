.PHONY: bump
bump:
	@echo "🚀 Bumping Version"
	git tag $(shell svu patch)
	git push --tags

.PHONY: build
build:
	@echo "🚀 Building Version $(shell svu current)"
	go build -o gitfamous main.go

.PHONY: vhs
vhs:
	@echo "📼 VHS Recording"
	@echo "Please ensure you have the 'vhs' command installed."
	vhs < vhs.tape

.PHONY: release
release:
	@echo "🚀 Releasing Version $(shell svu current)"
	goreleaser build --id default --clean --snapshot --single-target --output dist/gitfamous