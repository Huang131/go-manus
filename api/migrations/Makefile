.PHONY: migrate-up migrate-down migrate-status

# 数据库迁移脚本
# 用法:
#   make migrate-up          - 执行所有迁移
#   make migrate-down        - 回滚所有迁移
#   make migrate-status      - 查看迁移状态

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/manus?sslmode=disable
MIGRATIONS_DIR := $(shell pwd)

# 执行所有迁移 (按顺序执行 001, 002, 003...)
migrate-up:
	@echo "Running migrations..."
	@for f in $(MIGRATIONS_DIR)/*.sql; do \
		echo "Executing $$f..."; \
		psql "$(DATABASE_URL)" -f "$$f" || exit 1; \
	done
	@echo "Migrations completed."

# 回滚所有迁移 (按倒序回滚)
migrate-down:
	@echo "Rolling back migrations..."
	@echo "DROP TABLE IF EXISTS app_configs;" | psql "$(DATABASE_URL)"
	@echo "DROP TABLE IF EXISTS files;" | psql "$(DATABASE_URL)"
	@echo "DROP TABLE IF EXISTS sessions;" | psql "$(DATABASE_URL)"
	@echo "Rollback completed."

# 查看迁移状态
migrate-status:
	@echo "Migration files in $(MIGRATIONS_DIR):"
	@ls -1 $(MIGRATIONS_DIR)/*.sql 2>/dev/null || echo "No migration files found."