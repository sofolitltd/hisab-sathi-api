PORT ?= 8080

.PHONY: run

# পুরোনো সার্ভার পোর্টে চালু থাকলে আগে বন্ধ করে
run:
	-@lsof -ti tcp:$(PORT) -sTCP:LISTEN | xargs kill 2>/dev/null; sleep 0.5
	go run .
