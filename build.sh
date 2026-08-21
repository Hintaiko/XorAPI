#!/usr/bin/env bash
# 一键构建脚本：构建前端 -> 编译后端（含多平台二进制）
set -e
cd "$(dirname "$0")"

echo "==> [1/3] 构建前端..."
cd frontend
pnpm install --frozen-lockfile || pnpm install
pnpm build
cd ..

echo "==> [1.5/3] 同步前端产物到后端嵌入目录..."
rm -rf backend/internal/web/dist
mkdir -p backend/internal/web/dist
cp -r frontend/dist/* backend/internal/web/dist/
# go:embed 要求目录非空，克隆后未构建前端时保留占位文件以通过编译
if [ -z "$(ls -A backend/internal/web/dist)" ]; then touch backend/internal/web/dist/.gitkeep; fi

echo "==> [2/3] 编译后端..."
cd backend
go build -o xorapi.exe ./cmd/server

echo "==> [3/3] 交叉编译 Linux amd64..."
GOOS=linux GOARCH=amd64 go build -o ../release/xorapi-linux-amd64 ./cmd/server
GOOS=linux GOARCH=arm64 go build -o ../release/xorapi-linux-arm64 ./cmd/server

echo "完成！二进制位于 backend/xorapi.exe 与 release/ 目录"
