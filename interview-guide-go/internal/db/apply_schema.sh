#!/usr/bin/env bash
# 对「已有」Postgres 增量执行 schema/*.sql（compose 的 initdb.d 仅在空数据目录时跑一次）。
# 用法：export DATABASE_URL='postgres://...' && ./apply_schema.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCHEMA="$ROOT/schema"

if [[ "${1:-}" != "" ]]; then
	export DATABASE_URL="$1"
fi
: "${DATABASE_URL:?请设置 DATABASE_URL，或 ./apply_schema.sh 'postgres://...'}"

export PGCONNECT_TIMEOUT="${PGCONNECT_TIMEOUT:-15}"

# 按依赖顺序执行（勿依赖 shell 对 07_*.sql 的字典序）；见 schema/README.md。
files=(
	01_extensions.sql
	02_resumes.sql
	03_resume_analyses.sql
	04_interview.sql
	05_knowledge_bases.sql
	06_rag_chat.sql
	07_knowledge_base_chunks.sql
	08_resumes_pdf24_output_key.sql
	10_resume_interviewer_role.sql
	09_resumes_interviewer_role_no_general.sql
)

for f in "${files[@]}"; do
	fp="$SCHEMA/$f"
	if [[ ! -f "$fp" ]]; then
		echo "missing: $fp" >&2
		exit 1
	fi
	echo "==> $f"
	psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$fp"
done

echo "done."
