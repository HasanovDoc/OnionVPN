# Onion VPN Utility

[English](#english) | [Русский](#русский)

---

## Русский

**Onion VPN Utility** — это легковесный инструмент автоматизации сетевой маршрутизации с открытым исходным кодом, разработанный на связке Go (Wails) и Vue 3 (TailwindCSS). Утилита предназначена для маршрутизации выборочного трафика через распределенную сеть Tor, обеспечивая гибкое управление приватностью на уровне отдельных доменов.

Проект разработан исключительно в образовательных целях, а также для демонстрации возможностей интеграции модулей маршрутизации.

### Основные возможности

* **Выборочный роутинг (Split Tunneling):** Направление трафика только для определенных пользователем доменных имен и их поддоменов (используется сопоставление по ключевым словам).
* **Автоматическая обработка `.onion`:** Любые скрытые сервисы в зоне `.onion` перенаправляются в сеть Tor по умолчанию.
* **Два режима работы на Windows:**
    1.  **Модуль TUN (sing-box):** Создание виртуального сетевого адаптера для полноценного туннелирования на уровне системы.
    2.  **Режим SysProxy:** Легковесный режим через динамическую генерацию скрипта автоматической настройки прокси (`proxy.pac`) без необходимости сетевой виртуализации.
* **Поддержка Pluggable Transports:** Встроенная поддержка обфускации трафика через мосты (obfs4, snowflake, webtunnel).
* **Интерактивный интерфейс:** Динамический поиск, быстрое добавление/удаление правил и удобное переключение активности доменов прямо «на лету».

### Настройка Pluggable Transports (Мосты)

Для работы утилиты в изолированных сетях или при тестировании соединений в условиях ограниченной сетевой связности может потребоваться использование транспортов обфускации (мостов). 

Получить официальные параметры конфигурации мостов можно через стандартные каналы распределенной сети:
1. Официальный роботизированный email-сервис: отправить письмо с текстом `get bridges` на адрес bridges@torproject.org (запрос необходимо отправлять с почтовых сервисов Gmail или Riseup).
2. Официальный Telegram-бот: @GetBridgesBot

### Системные требования

* **ОС:** Windows 10 / 11 (64-bit)
* **Права:** Для работы модуля TUN (`wintun.dll`) требуются права администратора.

### Отказ от ответственности (Disclaimer)

Данное программное обеспечение предоставляется «как есть» (as is), без каких-либо явных или подразумеваемых гарантий. Автор не несет ответственности за возможные сбои в работе системы, потерю данных или характер использования утилиты конечными пользователями. Разработчик не поддерживает и не поощряет использование данного софта для осуществления противоправной деятельности.

---

## English

**Onion VPN Utility** is an open-source, lightweight network routing automation tool built using Go (Wails framework) and Vue 3 (TailwindCSS). This utility orchestrates selective domain-based traffic forwarding through the Tor network, providing deep customizability for application-level and system-wide privacy management.

This project has been developed strictly for educational purposes and internal network management demonstrations.

### Features

* **Selective Domain Routing (Split Tunneling):** Routes traffic only for user-specified domain keywords and their matching subdomains.
* **Native `.onion` Resolution:** Automatically captures and forces all `.onion` hidden services through the Tor network.
* **Dual-Engine Interception on Windows:**
    1.  **TUN Module (via sing-box):** Spawns a virtual network interface driver for robust, lower-level system-wide tunneling.
    2.  **SysProxy Mode:** A non-intrusive approach utilizing automated Local PAC (`proxy.pac`) deployment via the Windows Registry.
* **Pluggable Transports Support:** Built-in integration for bridges (obfs4, snowflake, webtunnel).
* **Advanced Web UI:** Fast domain mutation (inline append/remove through live lookups) and state toggles.

### Requirements

* **OS:** Windows 10 / 11 (64-bit)
* **Privileges:** Administrator privileges are mandatory when using the TUN adapter engine (`wintun.dll`).

### Disclaimer

This software is provided "as is", without warranty of any kind, express or implied. The author shall not be held liable for any claims, damages, or system misconfigurations arising from the usage of this tool. The developer does not promote, condone, or encourage the utilization of this utility for any illicit activities.

---

## Лицензия / License

Distributed under the MIT License. See `LICENSE` for more information.