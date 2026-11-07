PORT ?= 8080

.PHONY: run landing

# পুরোনো সার্ভার পোর্টে চালু থাকলে আগে বন্ধ করে
run:
	-@lsof -ti tcp:$(PORT) -sTCP:LISTEN | xargs kill 2>/dev/null; sleep 0.5
	go run .

# internal/landing/web/app.css বানায় (Tailwind)
landing:
	npm run landing
