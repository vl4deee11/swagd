# swagd
# Swagger decomposition util

Утилита для переноса path с parameters и definitions в разные файлы

Run:

    user@user swag % ./swag.bin -in=../../../swagger-doc/swagger.yml -out-dir=./spec/swagger -title=service -path-index=1 -auto-split=false
    2022/11/07 14:40:04 HANDLER = /v1/users/{id}
    WRITE NAME OF FILE FOR THIS HANDLER (OR 'exit' TO SAVE AND STOP)...
    user-swagger <-- enter file name
    2022/11/07 14:40:11 WRITE TO = user-swagger.yml


