# Telegram Captcha Bot
Этот бот отправляет новым участникам чата сообщение с кнопками и предлагает выбрать верный вариант ответа. 
Если будет выбрана кнопка "Спам", участник будет переведен в readonly режим.

## Основные функции
- Отправка приветственного сообщения новым участникам чата.
- Предоставление кнопок для выбора цели участия в чате.
- Ограничение отправки сообщений от участника, если выбран вариант "Спам"
- Удаление сообщений новых пользователей, которые не подтвердили выбор


## Установка и запуск
1. Добавить бота как администратора в группу, в которую нуно добавить проверку
2. Создать файл `.env` и добавить в него `BOT_TOKEN`
3. Запустить бота с помощью команды `go run main.go`