AJAIB_ID="${1:-130010505}"
HEADER=$(printf '{}' | base64 | tr '+/' '-_' | tr -d '=\n')
PAYLOAD=$(printf '{"sub":"%s"}' "$AJAIB_ID" | base64 | tr '+/' '-_' | tr -d '=\n')

echo
echo "Ajaib ID: $AJAIB_ID"
echo "Token:    $HEADER.$PAYLOAD.fake"
