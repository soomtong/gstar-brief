#!/bin/bash

TOTAL=3272
BATCH=5
FILES=$(((TOTAL + BATCH - 1) / BATCH))

mkdir -p output

echo "총 저장소: $TOTAL, 배치 크기: $BATCH, 생성 파일 수: $FILES"

for i in $(seq 0 $((FILES - 1))); do
	offset=$((i * BATCH))
	file=$(printf "output/%04d.md" $((i + 1)))

	if [ -f "$file" ]; then
		echo "[SKIP] $file"
		continue
	fi

	echo "[$(date '+%H:%M:%S')] $(printf '%04d' $((i + 1)))/$FILES — offset=$offset → $file"
	./gstar-brief report --offset "$offset" --limit "$BATCH" -o "$file" 2>&1 | grep -v '^$'
	sleep 60
done

echo "완료!"
