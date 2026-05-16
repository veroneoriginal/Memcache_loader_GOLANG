.PHONY: up down run dry reset proto build docker-run

# Генерация кода из .proto файла
proto:
	mkdir -p src/appsinstalled
	protoc \
	  --go_out=src/appsinstalled \
	  --go_opt=paths=source_relative \
	  appsinstalled.proto

# Сборка бинарника локально
build: proto
	go build -o memc_loader ./src

# Поднять только memcache-кластер
up:
	docker compose up -d memcache_idfa memcache_gaid memcache_adid memcache_dvid

# Остановить всё
down:
	docker compose down

# Dry-run локально (без реальной записи)
dry: build
	./memc_loader --dry --pattern="data/*.tsv.gz"

# Запуск локально с реальной записью
run: build
	./memc_loader \
		--pattern="data/*.tsv.gz" \
		--idfa="127.0.0.1:33013" \
		--gaid="127.0.0.1:33014" \
		--adid="127.0.0.1:33015" \
		--dvid="127.0.0.1:33016"

# Запуск всего в Docker
docker-run:
	docker compose up --build

# Восстановить обработанные файлы для повторного запуска
reset:
	@cd data && for f in .*.tsv.gz; do [ -f "$$f" ] && mv "$$f" "$${f#.}"; done; true
	@echo "Files in data/ restored"
