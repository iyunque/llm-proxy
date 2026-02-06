@echo off
echo Testing API endpoint...
echo.
echo Testing /taluoaiqing endpoint:
curl.exe -X POST http://localhost:8080/taluoaiqing -H "X-API-Key: 123456789" -H "Content-Type: application/json" --data "{\"content\": \"塔罗牌解读测试\"}" -v
echo.
echo.
pause