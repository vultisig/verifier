# adjust to point to your local go-wrappers repo
DYLD_LIBRARY=../go-wrappers/includes/darwin/:$LD_LIBRARY_PATH

up:
	@docker compose up -d --remove-orphans;

down:
	@docker compose down

# Run the verifier server
run-server:
	@DYLD_LIBRARY_PATH=$(DYLD_LIBRARY) VS_CONFIG_NAME=config go run cmd/verifier/main.go


# Run the worker process
run-worker:
	@DYLD_LIBRARY_PATH=$(DYLD_LIBRARY) VS_CONFIG_NAME=config go run cmd/worker/main.go
	
