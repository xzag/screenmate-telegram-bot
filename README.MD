# ScreenMate Telegram Bot

Telegram-бот для просмотра и управления кондиционерами через древний ScreenMate / TAC Vista WebStation.

Бот авторизуется в ScreenMate, выбирает нужную комнату по `room_id`, парсит текущие состояния кондиционеров и позволяет переключать их через Telegram.

## Возможности

* Просмотр статусов кондиционеров по всем комнатам.
* Отдельный экран комнаты.
* Обновление только выбранной комнаты.
* Переключение кондиционера через кнопку.
* Защита от двойного toggle:

    * кнопка содержит состояние кондиционера на момент отрисовки;
    * если состояние уже изменилось другим пользователем, повторный toggle не выполняется.
* Кэширование ScreenMate-сессий по комнатам.
* Автоматический relogin при протухшей сессии.
* Последовательная работа внутри одной комнаты через mutex, чтобы не ломать ASP.NET ViewState.

## Команды

### `/start`

Открывает список комнат.

После выбора комнаты показывается экран управления:

```text
❄️ Кондиционирование

🏢 Кабинет разработки

🟢 у окна — включен
🔴 над столом — выключен

Обновлено: 12:34:56
```

Кнопки:

```text
[🟢 у окна · выключить]
[🔴 над столом · включить]
[🔄 Обновить комнату]
[⬅️ К комнатам]
```

### `/status`

Показывает полный список всех комнат и кондиционеров без кнопок управления.

## Конфигурация

Реальный конфиг хранится в `config.yaml`.

Файл `config.yaml` нельзя коммитить в репозиторий.

В репозитории должен быть только `config.example.yaml`.

### `config.example.yaml`

```yaml
telegram:
  token: "put-telegram-token-here"
  allowed_users:
    - 123456789

screenmate:
  base_url: "http://screenmate.example.local/screenmate"
  max_parallel_rooms: 3

rooms:
  - key: "dev"
    name: "Кабинет разработки"
    username: "screenmate-user"
    password: "screenmate-password"
    room_id: "12345"
    conditioners:
      2: "у окна"
      3: "над столами"
      4: "у входа"

  - key: "meeting"
    name: "Переговорка"
    username: "screenmate-user-2"
    password: "screenmate-password-2"
    room_id: "67890"
    conditioners:
      2: "левый"
      3: "правый"
```

### Поля конфига

#### `telegram.token`

Токен Telegram-бота от BotFather.

#### `telegram.allowed_users`

Список Telegram user ID, которым разрешен доступ к боту.

#### `screenmate.base_url`

Базовый URL ScreenMate.

Пример:

```text
http://screenmate.local/screenmate
```

#### `screenmate.max_parallel_rooms`

Сколько комнат можно опрашивать параллельно при `/status`.

Рекомендуемое значение: `3`.

#### `rooms[].key`

Технический ключ комнаты. Используется внутри callback-ов.

Лучше использовать короткие значения:

```yaml
key: "dev"
key: "meeting"
key: "office1"
```

#### `rooms[].name`

Человекочитаемое название комнаты.

#### `rooms[].username`

ScreenMate-пользователь для комнаты.

#### `rooms[].password`

Пароль ScreenMate-пользователя.

#### `rooms[].room_id`

Room ID в ScreenMate.

#### `rooms[].conditioners`

Список кондиционеров в комнате.

Ключ — номер кондиционера в ScreenMate.

Значение — человекочитаемый комментарий для Telegram UI.

## Локальный запуск

Скопировать пример конфига:

```bash
cp config.example.yaml config.yaml
```

Заполнить реальные значения.

Запустить:

```bash
go run ./cmd/screenmate-bot -config ./config.yaml
```

## Сборка

```bash
go build -o screenmate-bot ./cmd/screenmate-bot
```

## Сборка под Linux

```bash
GOOS=linux GOARCH=amd64 go build -o screenmate-bot ./cmd/screenmate-bot
```

## Деплой на сервер

Создать пользователя:

```bash
sudo useradd --system \
  --home /opt/screenmate-bot \
  --shell /usr/sbin/nologin \
  screenmate
```

Создать каталог:

```bash
sudo mkdir -p /opt/screenmate-bot
sudo chown screenmate:screenmate /opt/screenmate-bot
```

Загрузить бинарник:

```bash
scp screenmate-bot root@SERVER_IP:/opt/screenmate-bot/
```

Установить права:

```bash
sudo chown screenmate:screenmate /opt/screenmate-bot/screenmate-bot
sudo chmod +x /opt/screenmate-bot/screenmate-bot
```

## systemd unit

Права на конфиг:

```bash
sudo mkdir -p /etc/screenmate-bot
sudo nano /etc/screenmate-bot/config.yaml
sudo chown root:root /etc/screenmate-bot/config.yaml
sudo chmod 600 /etc/screenmate-bot/config.yaml
```

Создать файл:

```bash
sudo nano /etc/systemd/system/screenmate-bot.service
```

Содержимое:

```ini
[Unit]
Description=ScreenMate Telegram Bot
After=network.target

[Service]
WorkingDirectory=/opt/screenmate-bot
ExecStart=/opt/screenmate-bot/screenmate-bot -config /etc/screenmate-bot/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Перезапустить systemd:

```bash
sudo systemctl daemon-reload
```

Включить автозапуск:

```bash
sudo systemctl enable screenmate-bot
```

Запустить:

```bash
sudo systemctl start screenmate-bot
```

Проверить статус:

```bash
sudo systemctl status screenmate-bot
```

Логи:

```bash
journalctl -u screenmate-bot -f
```

## Обновление на сервере

Собрать новый бинарник:

```bash
GOOS=linux GOARCH=amd64 go build -o screenmate-bot ./cmd/screenmate-bot
```

Скопировать на сервер:

```bash
scp screenmate-bot user@server:/tmp/screenmate-bot
```

На сервере:

```bash
sudo systemctl stop screenmate-bot
sudo cp /tmp/screenmate-bot /opt/screenmate-bot/screenmate-bot
sudo chmod +x /opt/screenmate-bot/screenmate-bot
sudo systemctl start screenmate-bot
sudo systemctl status screenmate-bot
```

## Безопасность

В репозиторий нельзя коммитить:

```text
config.yaml
.env
debug_*.html
```

Рекомендуемый `.gitignore`:

```gitignore
config.yaml
*.local.yaml
.env
debug_*.html
screenmate-bot
```

## Как работает ScreenMate-клиент

ScreenMate использует ASP.NET WebForms.

Бот повторяет браузерный сценарий:

```text
GET  LoginPage.aspx
POST LoginPage.aspx

POST room lookup form

POST room control page через __EVENTTARGET
```

Для работы с WebForms бот сохраняет:

```text
cookies
__VIEWSTATE
__VIEWSTATEGENERATOR
__EVENTVALIDATION
roomId
form action
```

Важно: `roomId` должен отправляться не только при выборе комнаты, но и при последующих postback-ах.

## Сессии

Для каждой комнаты хранится отдельная ScreenMate-сессия:

```text
room key -> screenmate client
```

Внутри комнаты все операции выполняются последовательно через mutex.

Это нужно, потому что ASP.NET ViewState mutable, и параллельные postback-и в одну комнату могут сломать состояние страницы.

## Защита от двойного toggle

Кнопка кондиционера содержит состояние, которое пользователь видел при отрисовке:

```text
t:<roomKey>:<acNumber>:<expectedPower>
```

Пример:

```text
t:dev:2:1
```

Это значит:

```text
Пользователь видел, что кондиционер 2 включен.
Перед toggle бот проверит текущее состояние.
Если он всё еще включен — выполнит toggle.
Если он уже выключен — ничего не сделает и просто обновит UI.
```

Так два пользователя не смогут случайно отменить действие друг друга.

## Troubleshooting

### Бот пишет `Нет доступа`

Проверь Telegram user ID в `telegram.allowed_users`.

### Комната не открывается

Проверь:

```yaml
room_id
username
password
```

Также стоит проверить логи:

```bash
journalctl -u screenmate-bot -f
```

### Не находятся кондиционеры

Проверь, что в ScreenMate для комнаты реально есть элементы вида:

```text
Кондиционер N (ВКЛ/ВЫКЛ)
```

И что в HTML есть ссылки:

```html
dataList_toggle_...
```

### Toggle не работает

Проверь, что при postback отправляется поле:

```text
roomId
```

ScreenMate требует его даже при клике по toggle.

### Ошибка `context deadline exceeded`

ScreenMate не ответил вовремя.

Бот сбросит сессию комнаты. Следующая команда должна создать новую сессию и перелогиниться.

Если ошибка повторяется постоянно, проверь доступность ScreenMate с сервера:

```bash
curl -I http://screenmate.example.local/screenmate/LoginPage.aspx
```

## Разработка

Основные пакеты:

```text
cmd/screenmate-bot
  main.go

internal/config
  загрузка config.yaml

internal/screenmate
  низкоуровневый клиент к ASP.NET WebForms

internal/service
  комнаты, сессии, бизнес-логика

internal/bot
  Telegram UI, callbacks, formatting
```

Поток `/start`:

```text
Telegram /start
  -> список комнат
  -> выбор комнаты
  -> service.RoomStatus(roomKey)
  -> screenmate.Client.Status()
  -> edit Telegram message
```

Поток toggle:

```text
Telegram button
  -> callback t:<roomKey>:<acNumber>:<expectedPower>
  -> service.TogglePowerIfState()
  -> screenmate refresh
  -> проверка expectedPower
  -> postback toggle
  -> обновление только выбранной комнаты
```
