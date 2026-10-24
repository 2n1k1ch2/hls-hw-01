#!/usr/bin/env bash

BASE="http://localhost:8080"
COUNT="${1:-2000000}"

NAMES=("Морковка" "Картошка" "Огурец" "Помидор" "Лук" "Капуста")
IDS=()

request() {
    local method="$1"
    local url="$2"
    local data="${3:-}"

    if [[ -n "$data" ]]; then
        curl -sS -X "$method" "$url" \
            -H "Content-Type: application/json" \
            -d "$data" -w $'\n%{http_code}'
    else
        curl -sS -X "$method" "$url" -w $'\n%{http_code}'
    fi
}

create_item() {
    local name price count result body status id
    name="${NAMES[$((RANDOM % ${#NAMES[@]}))]}"
    price=$((RANDOM % 500 + 10))
    count=$((RANDOM % 100 + 1))
    result=$(request POST "$BASE/item" "{\"name\":\"$name\",\"price\":$price,\"count\":$count}")
    status="${result##*$'\n'}"
    body="${result%$'\n'*}"

    if [[ "$status" == "201" ]]; then
        id=$(printf '%s' "$body" |
            grep -oE '"id"[[:space:]]*:[[:space:]]*"[^"]+"' |
            head -n1 | cut -d '"' -f4)

        if [[ -n "$id" ]]; then
            IDS+=("$id")
            echo "POST /item: создан $name ($id)"
            return
        fi
    fi

    echo "POST /item: HTTP $status — $body"
}

random_id() {
    echo "${IDS[$((RANDOM % ${#IDS[@]}))]}"
}

remove_id() {
    local target="$1"
    local new_ids=()
    local id

    for id in "${IDS[@]}"; do
        [[ "$id" == "$target" ]] || new_ids+=("$id")
    done
    IDS=("${new_ids[@]}")
}

echo "Запуcк $COUNT случайных запросов к $BASE"

for ((i = 1; i <= COUNT; i++)); do
    route=$((RANDOM % 4))
    if (( RANDOM % 4 == 0 )); then
        fail=1
    else
        fail=0
    fi

    case "$route" in
        0)
            result=$(request GET "$BASE/items")
            status="${result##*$'\n'}"
            echo "GET /items: HTTP $status"
            ;;

        1)
            if (( fail )); then
                result=$(request GET "$BASE/item?id=invalid-id")
            elif (( ${#IDS[@]} > 0 )); then
                id=$(random_id)
                result=$(request GET "$BASE/item?id=$id")
            else
                result=$(request GET "$BASE/item?id=invalid-id")
            fi
            status="${result##*$'\n'}"
            echo "GET /item: HTTP $status"
            ;;

        2)
            if (( fail )); then
                result=$(request POST "$BASE/item" '{"invalid-json":')
                status="${result##*$'\n'}"
                echo "POST /item (ошибка): HTTP $status"
            else
                create_item
            fi
            ;;

        3)
            if (( fail )); then
                result=$(request DELETE "$BASE/item?id=invalid-id")
                status="${result##*$'\n'}"
                echo "DELETE /item (ошибка): HTTP $status"
            elif (( ${#IDS[@]} > 0 )); then
                id=$(random_id)
                result=$(request DELETE "$BASE/item?id=$id")
                status="${result##*$'\n'}"
                echo "DELETE /item: HTTP $status"

                if [[ "$status" == "204" ]]; then
                    remove_id "$id"
                fi
            else
                result=$(request DELETE "$BASE/item?id=invalid-id")
                status="${result##*$'\n'}"
                echo "DELETE /item: HTTP $status"
            fi
            ;;
    esac


    sleep 0.01
done

