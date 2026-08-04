SHELL := /bin/sh

.DEFAULT_GOAL := help

.PHONY: help init dev up down logs test check fmt api web db

help: ## 显示可用命令
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z_-]+:.*## / {printf "  %-12s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

init: ## 创建本地配置并安装依赖
	@test -f .env || cp .env.example .env
	cd server && go mod download
	cd web && npm ci

dev: ## 一键启动开发环境（Vue 热更新 + Go + PostgreSQL + Redis）
	docker compose -f docker-compose.dev.yml up --build

up: ## 构建并启动生产形态容器
	docker compose up --build -d

down: ## 停止开发及生产容器
	docker compose -f docker-compose.dev.yml down --remove-orphans
	docker compose down --remove-orphans

logs: ## 跟踪生产形态容器日志
	docker compose logs -f

test: ## 运行后端测试与前端类型/构建检查
	cd server && go test -race ./...
	cd web && npm run build

check: ## 运行 Go 静态检查与前端类型检查
	cd server && go vet ./...
	cd web && npm run typecheck

fmt: ## 格式化 Go 源码
	cd server && gofmt -w .

api: ## 在宿主机运行 API（数据库与 Redis 需已启动）
	cd server && go run .

web: ## 在宿主机运行 Vue 开发服务器
	cd web && npm run dev

db: ## 仅启动 PostgreSQL 与 Redis
	docker compose -f docker-compose.dev.yml up -d postgres redis
