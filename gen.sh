#!/bin/bash

# Скрипт для генерации кода оператора Kubernetes
set -e

echo "🚀 Запуск генерации кода оператора..."

echo "1. Генерация кода..."
make generate

echo "2. Генерация манифестов..."
make manifests

echo "3. Установка CRD в кластер..."
make install

make run

echo "✅ Все операции завершены успешно!"
