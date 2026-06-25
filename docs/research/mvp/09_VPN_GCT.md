# VPN research material, part:  09

Dive deep into this research and do comprehenisve refining! VPS hosted service has to be VPN gateway to network(s) connected to it. It MUST SUPPORT multiple joined networks which will be exposed through it to end users accessing these networks. So basically it establishes two ways VPN connection! We MUST have comprehensive client apps on both sides,  end-user clients and for network side. Apps for configuring, customizations and administration, and access apps (like Mullvad VPN). All power features Mullvad VPN service is offering MUST BE available through this system. Especially connections obfuscation, especially Quic! Technology stack: backend and API: Go, Gin Gonic, Postgres, Redis, Podman, and everything related. Client sides, all apps - shared reusable codebases, same tech stack chosen with maximal reusability of codebase, small app sizes and small memory consumption, blazing flashing fast and stable execution. We MUST cutting edge solution for this. Suggestions: Flutter, Go based solutions, Rust based solutions, Kotlin Multiplatform. We MUST have all apps on following platforms: Mobile (ios, android, aurora - auroraos.ru, HarmonyOS), Desktop (Windows, Linux, macOS), Web (all web browser and fully responsive). Everything MUST event driven, real time reaponsive! The whole system!!! Rock solid, fast, stable, game changer solution!!!

## Research

Вы предоставили исчерпывающее исследование по созданию VPN-шлюза на базе VPS, которое служит отличной базой. Однако ваш запрос требует не просто развертывания, а проектирования и реализации **полноценной, кроссплатформенной экосистемы**, сопоставимой по возможностям с Mullvad, с упором на QUIC, обфускацию и унифицированный технологический стек.

Ниже представлен проект архитектуры и план реализации, который объединяет ваши требования в единое решение.

### 1. Архитектура: Обратный туннель и Multi-Network Gateway

Ваша идея с обратным туннелем (outbound connection) — это фундамент. Мы развиваем её для поддержки нескольких сетей.

**Схема:**
1.  **VPS (Gateway):** Центральный узел. На нём работают серверные компоненты (Hysteria2/WireGuard), API-шлюз (Gin), база данных (Postgres) и брокер сообщений (Redis).
2.  **Internal Host (Gateway Node):** Устройство внутри вашей локальной сети (или облака). Оно инициирует соединение с VPS и выступает в роли моста для всей своей подсети.
3.  **Remote Clients:** Конечные пользователи (мобильные, десктопные), подключающиеся к VPS.

**Реализация Multi-Network:**
Чтобы «видеть» несколько сетей через один шлюз, `Internal Host` должен быть настроен как полноценный маршрутизатор:
*   **IP Forwarding:** Включается на уровне ядра (`net.ipv4.ip_forward=1`).
*   **NAT (MASQUERADE):** На `Internal Host` настраивается правило `iptables -t nat -A POSTROUTING -o <tunnel_interface> -j MASQUERADE`.
*   **Маршрутизация:** На VPS необходимо прописать маршруты ко всем сетям, которые находятся за `Internal Host`. Это делается через WireGuard (`AllowedIPs = <network1>, <network2>, ...`) или через статические маршруты в Linux.

### 2. Технологический стек: Go как единый язык

Требование использовать Go для бэкенда и API (Gin Gonic) — верное решение. Для достижения максимальной переиспользуемости кода и минимизации размера приложений, Go должен стать **единственным языком** для всей системы.

*   **Backend & API:** Go + Gin Gonic + Postgres + Redis.
    *   Gin обеспечит высокую производительность API для управления пользователями, ключами, мониторингом.
    *   Postgres — надежное хранилище для конфигураций и логов.
    *   Redis — для кэширования сессий, реализации kill-switch на стороне сервера или брокера событий.
*   **Клиентские приложения:** Go + Fyne (для Desktop) / go-flutter-desktop (для Desktop/Web) / Wails (для Desktop).
    *   **Почему не Flutter/Kotlin Multiplatform?** Хотя они решают задачу кроссплатформенности, они вводят дополнительную сложность: необходимость поддерживать две кодовые базы (Go-бэкенд и Dart/Kotlin-клиент), синхронизацию версий, отладку взаимодействия между ними. Использование Go для всего позволяет писать бизнес-логику один раз.
    *   **Fyne / go-flutter-desktop:** Позволяют компилировать нативные приложения для Windows, macOS, Linux из одного кода на Go. Размер бинарника будет небольшим, потребление памяти — минимальным.
    *   **Web:** Можно использовать Wails или компилировать Go в WASM для запуска в браузере. Это обеспечит единую кодовую базу и для веб-версии.
    *   **Mobile (iOS/Android):** Здесь Go пока не может компилироваться напрямую в нативный UI так же легко, как во Fyne. Однако можно использовать подход с **общей библиотекой логики** на Go и тонкими обертками на Swift/Kotlin/Java для UI. Это компромисс, но он сохраняет 80% логики в одном месте.

### 3. Выбор протокола: Hysteria2 vs WireGuard

*   **Hysteria2:** **Основной выбор.** Это единственный готовый к продакшену протокол, который реализует QUIC и встроенную обфускацию (Salamander). Он идеально подходит под требование «быть как Mullvad». Он отлично работает на нестабильных сетях.
*   **WireGuard:** **Резервный/Альтернативный.** Используется там, где критична максимальная совместимость со встроенными клиентами ОС или где не требуется глубокая обфускация. Для WireGuard можно использовать сторонние решения вроде udp2raw или AmneziaWG для маскировки трафика.

**Рекомендация:** Архитектура должна поддерживать оба протокола. Серверное приложение на Go будет уметь поднимать как Hysteria2, так и WireGuard интерфейсы одновременно по запросу клиента.

### 4. Реализация функций уровня Mullvad

Все эти функции должны быть реализованы на стороне сервера (Go) и управляться через API:

*   **Kill-Switch:** Реализуется на уровне фаервола сервера и/или клиента. При падении туннеля API-сервер дает команду фаерволу (`iptables`/`nftables`) блокировать весь трафик клиента, кроме управляющего API.
*   **Split-Tunneling:** Настраивается через `AllowedIPs` в конфиге туннеля. Клиентское приложение позволяет пользователю выбрать, какие приложения/подсети пускать в VPN, а какие — нет.
*   **Multi-Hop:** Реализуется цепочкой туннелей на VPS. Трафик от клиента приходит на один интерфейс VPS и маршрутизируется на другой интерфейс (второй VPS или выход в интернет). Это сложная, но решаемая задача на уровне ядра Linux.
*   **Obfuscation:** Встроено в Hysteria2. Для WireGuard — решается внешними инструментами.

### 5. Управление событиями и Real-Time

Требование «всё должно быть событийно-ориентированным» реализуется через:
1.  **Redis Pub/Sub:** Бэкенд-сервисы будут подписываться на каналы Redis. Например, сервис мониторинга публикует событие о падении туннеля, а сервис kill-switch его получает и мгновенно применяет правила фаервола.
2.  **WebSockets / Server-Sent Events (SSE):** Для клиентских приложений (особенно Web и Desktop). Клиент подписывается на события от сервера: статус подключения, скорость, ошибки. Это обеспечивает отзывчивость UI в реальном времени без постоянного опроса API.

### Итоговый план действий по реализации

1.  **Ядро системы (Backend):**
    *   Написать на Go сервис управления туннелями (`vpn-manager`). Он будет отвечать за запуск/остановку Hysteria2/WireGuard процессов, управление конфигурациями пользователей через API Gin.
    *   Реализовать API для аутентификации, создания/удаления туннелей, получения статуса.
    *   Интегрировать Postgres для хранения данных о пользователях и их сессиях.

2.  **Событийная архитектура:**
    *   Настроить Redis Pub/Sub для коммуникации между сервисами внутри бэкенда (например, `vpn-manager` -> `firewall-manager`).

3.  **Клиентское приложение:**
    *   Разработать десктопное приложение на Fyne/go-flutter-desktop. Оно будет иметь единый код для Win/Mac/Lin.
    *   Реализовать логику подключения к API бэкенда, генерации ключей, управления туннелем через системные вызовы OS.
    *   Встроить WebView или использовать нативный UI для отображения статусов, полученных через SSE/WebSocket от бэкенда.

4.  **Мобильные клиенты:**
    *   Создать нативные приложения под iOS/Android.
    *   Вынести всю сложную логику (криптография, парсинг ответов API) в общую библиотеку на Go (скомпилированную в виде .aar/.framework) или использовать gomobile.
    *   UI писать нативно на Swift/Kotlin, но вся "начинка" будет общей.

5.  **Веб-клиент:**
    *   Реализовать SPA-приложение на Go+WASM или просто SPA на JS/TS, которое общается с тем же API бэкенда.

Это решение является "game changer", так как оно использует передовые технологии (QUIC/Hysteria2), обеспечивает полный контроль над данными (self-hosted), гарантирует высочайшую производительность за счет унификации стека на Go и реализует современную событийно-ориентированную архитектуру.

Отлично. Раз мы движемся по пути создания монолитной, но невероятно мощной и кроссплатформенной системы, давайте детализируем следующий этап: **проектирование и реализация серверного ядра (Backend) и API**.

Это фундамент, на котором будет держаться вся остальная логика: управление туннелями, пользователями, мониторинг и, что критически важно, **Kill-Switch**.

### 1. Проектирование серверного ядра (Go + Gin)

Мы создаем не просто API, а управляющий демон. Назовем его `vpn-controller`.

**Основные компоненты:**
1.  **Gin Web Server:** Обрабатывает REST API-запросы от клиентов (регистрация, аутентификация, получение конфигов).
2.  **Менеджер процессов (`ProcessManager`):** Модуль на Go, который отвечает за запуск, остановку и мониторинг процессов `hysteria` и `wg-quick`. Он должен уметь:
    *   Запускать процесс с переданным конфигом.
    *   Перезапускать его при падении.
    *   Убивать процесс по команде.
3.  **Менеджер фаервола (`FirewallManager`):** Модуль, который через `exec.Command` управляет `iptables` или `nftables`. Это ключевой элемент для реализации Kill-Switch.
4.  **Менеджер конфигураций (`ConfigManager`):** Генерирует файлы конфигурации для Hysteria2 и WireGuard на лету на основе данных из БД.

#### Архитектура взаимодействия:
*   **Клиент -> API (Gin):** Запрашивает создание туннеля.
*   **API -> `ProcessManager`:** Запускает процесс туннеля.
*   **API -> `FirewallManager`:** Применяет правила Kill-Switch для этого пользователя/туннеля.
*   **Мониторинг (Health Check):** Отдельный endpoint `/health` или фоновый процесс, который проверяет статус `hysteria`/`wg-quick` и, в случае падения, дает команду `FirewallManager` заблокировать трафик.

### 2. Реализация Kill-Switch на стороне сервера

Это самый надежный способ гарантировать отсутствие утечек. Логика такова:

1.  **При подключении клиента:**
    *   Сервер создает уникальный маркер (например, `mark=1234`) для трафика этого пользователя.
    *   `FirewallManager` применяет правило:
        ```bash
        # Разрешаем трафик с нашим маркером
        iptables -A OUTPUT -m mark --mark 1234 -j ACCEPT
        # Блокируем весь остальной трафик для этого пользователя/сети
        iptables -A OUTPUT -j REJECT
        ```
2.  **При активном туннеле:**
    *   Весь трафик от клиента, проходящий через VPS-интерфейс, помечается этим маркером (`-j MARK --set-mark 1234`).
    *   Благодаря правилу выше, этот трафик выходит в интернет.
3.  **При падении туннеля:**
    *   Трафик перестает маркироваться.
    *   Правило `REJECT` блокирует всё. Утечка невозможна.
4.  **При отключении клиента:**
    *   Сервер удаляет все правила с маркером `1234`, освобождая ресурсы.

### 3. Проектирование API (Gin)

API будет отвечать за управление всем жизненным циклом туннеля.

**Основные эндпоинты:**

*   **Аутентификация:**
    *   `POST /auth/login`: Аутентификация пользователя, получение JWT-токена.
*   **Управление туннелями:**
    *   `POST /tunnels`: Создать новый туннель для пользователя. В ответе — статус и ссылка на конфиг.
    *   `GET /tunnels/{id}/status`: Получить статус (активен/неактивен), статистику (tx/rx байт).
    *   `DELETE /tunnels/{id}`: Остановить туннель и снять фаервол-правила Kill-Switch.
*   **Генерация конфигураций:**
    *   `GET /tunnels/{id}/config/hysteria`: Отдает готовый YAML-конфиг для Hysteria2-клиента.
    *   `GET /tunnels/{id}/config/wireguard`: Отдает готовый `.conf` файл для WireGuard-клиента.
*   **События (SSE):**
    *   `GET /tunnels/{id}/events`: Клиент подключается к этому эндпоинту и получает поток событий в реальном времени: "connected", "disconnected", "tx=10MB", "rx=5MB".

### 4. Структура базы данных (Postgres)

Для управления пользователями и туннелями нам нужна простая, но надежная схема.

```sql
-- Таблица пользователей
CREATE TABLE users (
    id BIGSERIAL PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Таблица активных туннелей
CREATE TABLE tunnels (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT REFERENCES users(id) ON DELETE CASCADE,
    protocol VARCHAR(20) NOT NULL CHECK (protocol IN ('hysteria', 'wireguard')), -- какой протокол используется
    server_port INTEGER NOT NULL, -- порт на сервере (для информации)
    client_ip INET, -- последний известный IP клиента (для Kill-Switch)
    firewall_mark INTEGER UNIQUE NOT NULL, -- уникальный mark для iptables
    created_at TIMESTAMP DEFAULT NOW(),
    expires_at TIMESTAMP -- для сессионных ключей
);
```

### 5. Примерный код на Go (псевдокод логики)

Вот как может выглядеть упрощенная логика создания туннеля в `vpn-controller`.

```go
// TunnelService.go

func (s *TunnelService) CreateTunnel(userID int64, protocol string) (*Tunnel, error) {
    // 1. Создаем запись в БД
    tunnel := &Tunnel{
        UserID:      userID,
        Protocol:    protocol,
        FirewallMark: generateUniqueMark(), // генерируем уникальное число
        // ... другие поля
    }
    if err := s.db.Create(tunnel).Error; err != nil {
        return nil, err
    }

    // 2. Генерируем конфиг на диске или в памяти
    configPath := fmt.Sprintf("/etc/vpn/tunnels/%d.yaml", tunnel.ID)
    if err := s.configManager.Generate(tunnel, configPath); err != nil {
        return nil, err
    }

    // 3. Применяем правила Kill-Switch через FirewallManager
    // Блокируем исходящий трафик для этого mark'а
    if err := s.firewallManager.ApplyKillSwitch(tunnel.FirewallMark); err != nil {
        return nil, err
    }

    // 4. Запускаем процесс туннеля через ProcessManager
    // Передаем ему путь к конфигу и mark, чтобы он мог маркировать трафик
    cmdArgs := []string{"-c", configPath}
    
    // Для Hysteria мы можем маркировать трафик так:
    // hysteria server -c config.yaml --fwmark <mark>
    
    if err := s.processManager.Start(tunnel.ID, "hysteria", cmdArgs...); err != nil {
        // Если не удалось запустить - откатываем фаервол
        s.firewallManager.RemoveKillSwitch(tunnel.FirewallMark)
        return nil, err
    }

    return tunnel, nil
}
```

### Что делаем дальше?
Мы спроектировали бэкенд. Следующий логический шаг — это **разработка клиентского приложения**, которое будет единым для всех платформ.

Как мы уже решили, для максимальной переиспользуемости кода и минимизации размера приложений, мы будем использовать **Go**. Для десктопных платформ идеально подойдет **Fyne** или **Wails**, так как они позволяют писать UI на Go и компилировать его в нативные приложения для Windows, macOS и Linux с общим кодом.

Готовы перейти к проектированию клиентской части? Мы обсудим архитектуру приложения, как оно будет общаться с нашим новым API и как реализовать SSE для получения событий в реальном времени.

Отлично, движемся дальше. Мы спроектировали мощный и надежный бэкенд. Теперь перейдем к **клиентской части** — той самой, с которой будет взаимодействовать конечный пользователь.

Ваша цель — кроссплатформенность, единый код и высокая производительность. Идеальным инструментом для этого в экосистеме Go является фреймворк **Fyne**. Он позволяет писать UI-код один раз и компилировать его в нативные приложения для **Windows, macOS, Linux**, а также в веб-приложение (WASM).

### 1. Архитектура клиентского приложения (Desktop/Web)

Мы создаем приложение, которое будет не просто оболочкой для конфигов, а полноценным менеджером VPN-соединений.

**Основные модули приложения:**
1.  **API Client (API-ядро):** Модуль, отвечающий за всю коммуникацию с нашим бэкендом (`vpn-controller`). Он будет работать с JWT-токенами, отправлять запросы на создание туннелей, получать статистику и, что самое важное, **устанавливать SSE-соединение** для получения событий в реальном времени.
2.  **VPN Engine:** Модуль, который управляет системным процессом VPN-клиента (`hysteria` или `wg-quick`). Он будет запускать процесс с полученным от API конфигом и следить за его статусом.
3.  **UI Layer (Presentation):** Слой, написанный на Fyne. Он отображает данные, полученные от `API Client`, и отправляет пользовательские действия (нажать кнопку "Подключиться") в `API Client`.

#### Поток данных в приложении:
*   **Пользователь** нажимает кнопку "Подключиться" в **UI**.
*   **UI** вызывает метод `Connect()` в **API Client**.
*   **API Client** отправляет запрос на `/tunnels` на сервер.
*   Сервер создает туннель и возвращает ID и ссылку на конфиг.
*   **API Client** скачивает конфиг и передает его в модуль **VPN Engine** для запуска процесса.
*   Одновременно **API Client** открывает SSE-соединение с `/tunnels/{id}/events`.
*   Любое событие (подключение, отключение, передача данных) приходит по SSE.
*   **API Client** обрабатывает событие и обновляет состояние (например, `status = "Connected"`).
*   Состояние передается в **UI**, который мгновенно обновляет индикатор статуса и графики скорости.

### 2. Реализация Real-Time событий (Server-Sent Events - SSE)

Это ключевой элемент отзывчивости. Бэкенд на Gin должен уметь отдавать поток событий.

**На стороне сервера (Gin):**
```go
// Обработчик SSE
func TunnelEventsHandler(c *gin.Context) {
    tunnelID := c.Param("id")

    // Устанавливаем заголовки для SSE
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("Connection", "keep-alive")
    c.Writer.Header().Set("Transfer-Encoding", "chunked")
    c.Writer.Header().Set("X-Accel-Buffering", "no") // Для Nginx

    // Создаем канал для отправки событий этому клиенту
    eventChan := make(chan string)
    // Регистрируем клиента (в реальном приложении это будет в map[sseClientID]chan)

    // Функция для отправки сообщения
    sendEvent := func(event string) {
        fmt.Fprintf(c.Writer, "data: %s\n\n", event)
        c.Writer.(http.Flusher).Flush() // Обязательно сбрасываем буфер
    }

    // Отправляем начальное событие
    sendEvent(`{"status": "connected_to_events"}`)

    // Ждем событий или закрытия соединения
    for {
        select {
        case event := <-eventChan:
            sendEvent(event)
        case <-c.Request.Context().Done():
            // Клиент отключился
            return
        }
    }
}
```

**На стороне клиента (Go):**
```go
// В модуле API Client
func (c *APIClient) SubscribeToEvents(tunnelID string) error {
    req, _ := http.NewRequest("GET", fmt.Sprintf("%s/tunnels/%d/events", c.BaseURL, tunnelID), nil)
    req.Header.Set("Authorization", "Bearer "+c.Token)

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    scanner := bufio.NewScanner(resp.Body)
    for scanner.Scan() {
        line := scanner.Text()
        if strings.HasPrefix(line, "data: ") {
            eventData := strings.TrimPrefix(line, "data: ")
            // Парсим eventData (JSON) и отправляем в основной канал событий приложения
            c.handleEvent(eventData)
        }
    }
    return scanner.Err()
}
```
Эта система обеспечит мгновенную реакцию UI на любые изменения статуса VPN.

### 3. Проектирование UI на Fyne

Fyne использует декларативный подход. Мы описываем состояние виджетов, а фреймворк сам решает, как их отрисовать.

**Основные экраны приложения:**
1.  **Экран авторизации:** Простая форма логин/пароль.
2.  **Главный экран (Dashboard):**
    *   Кнопка глобального подключения/отключения.
    *   Индикатор статуса (цветовой круг: серый/зеленый/красный).
    *   Графики скорости передачи (TX/RX) в реальном времени.
    *   Список доступных локаций/серверов (если у вас их несколько).
3.  **Экран настроек:**
    *   Включение/выключение Kill-Switch.
    *   Настройка Split-Tunneling (выбор приложений или IP-адресов).
    *   Выбор протокола (Hysteria2 / WireGuard).
4.  **Экран логов:** Отображение системных логов `hysteria`/`wg-quick`.

**Пример кода главного экрана на Fyne:**
```go
// В функции main() или при создании окна

// 1. Создаем структуру для хранения состояния приложения
appState := &AppState{
    IsConnected: false,
    TXSpeed:     0,
    RXSpeed:     0,
}

// 2. Создаем виджеты, привязанные к состоянию
statusCircle := canvas.NewCircle(color.Gray{Y: 200}) // Серый по умолчанию

txLabel := widget.NewLabel("TX: 0 KB/s")
rxLabel := widget.NewLabel("RX: 0 KB/s")
connectBtn := widget.NewButton("Connect", func() {
    // Логика подключения через API Client
})

// 3. Функция для обновления UI из другого потока (обязательно!)
func updateUI() {
    if appState.IsConnected {
        statusCircle.FillColor = color.Green()
        connectBtn.SetText("Disconnect")
    } else {
        statusCircle.FillColor = color.Gray{Y: 200}
        connectBtn.SetText("Connect")
    }
    txLabel.SetText(fmt.Sprintf("TX: %d KB/s", appState.TXSpeed))
    rxLabel.SetText(fmt.Sprintf("RX: %d KB/s", appState.RXSpeed))
    
    // Это магическая функция Fyne для безопасного обновления UI из горутин
    fyne.CurrentApp().SendNotification(&fyne.Notification{
        Title:   "VPN Status",
        Content: fmt.Sprintf("Status changed to %v", appState.IsConnected),
    })
}

// 4. Собираем контейнер и отображаем
content := container.NewVBox(
    statusCircle,
    connectBtn,
    txLabel,
    rxLabel,
)
myWindow.SetContent(content)
myWindow.ShowAndRun()
```
В реальном коде `updateUI()` будет вызываться из обработчика событий SSE.

### Что делаем дальше?
Мы спроектировали клиентское приложение. Теперь у нас есть полная картина всей системы.
Следующий шаг — переход от архитектуры к **реализации**. Я могу помочь вам составить детальный план разработки с конкретными задачами или даже написать код для одного из ключевых модулей (например, `API Client` на Go или базовое окно авторизации на Fyne).

Куда двигаемся? К написанию кода?
