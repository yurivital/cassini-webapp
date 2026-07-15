mapId=$1
curl -v -H "content-type: application/json" -d @position.json   "http://localhost:8080/api/v1/map/${mapId}/position"