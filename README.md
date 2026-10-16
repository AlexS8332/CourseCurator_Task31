# 31 — Куратор курса: `/help` — ассистент, понимающий каталог курса

Задание 31 курса по AI (неделя 8, день 1). **Куратор** — ассистент
разработчика для каталога курса `AITrenning`: он отвечает на вопросы об
устройстве курса и прошлых заданиях **только по документации** (карточки,
конспекты, пособия, README, скилл `aitr`), а о текущем состоянии — ветка,
статусы заданий, последний контрольный прогон — **только из инструментов**
(git, `assignment.json`, `report.json`, `aitr doctor`), которые подключены к
проекту через MCP-сервер `course`.

Этот репозиторий — снимок модуля `tools/curator` корня курса на момент
сдачи задания (тег корня `v31`); снимок делает команда `curator snapshot 31`.
Код в заданиях 32–35 продолжит этот же модуль.

## Что сделано по условию

| Условие | Что есть |
|---|---|
| RAG по документации проекта: README, docs, схемы/API | `curator index` режет по заголовкам 54 markdown-документа каталога (31 карточка, конспекты, 2 пособия, ТЗ, README, SKILL, условия недель, шаблон карточки) и две «схемы», которые генерирует код: `data/schema/assignments.md` (поля `assignment.json` по 31 файлу) и `aitr.md` (функции `scripts/aitr.ps1` и таблица команд). Эмбеддер e5 (локальный сайдкар), без него — BM25. Порог «не знаю» калибруется по dev- и out-вопросам (`curator calibrate`) |
| MCP: хотя бы git branch, по желанию файлы и diff | MCP-сервер `course` (stdio и HTTP): `git_branch`, `git_status`, `git_log`, `git_diff`, `git_changed_files` — для корня и любого задания; `repo_tree`, `read_file`, `repo_search` с оградой (`.git`, `.env*`, `*.local`, `INDEX.md`, реестр); `assignments_list` (с расхождениями против реестра), `assignment_get`, `demo_report`, `aitr_doctor`, `docs_search`. Тот же сервер подключается к Claude Code через `.mcp.json` корня |
| `/help` отвечает о структуре проекта | Команда `/help` в окне куратора и `curator help "вопрос"` в консоли. Ответ — завершающий инструмент `help_answer`, который проверяет код: каждый источник — `chunk_id` из выдачи этого хода, каждая цитата — дословная подстрока фрагмента, каждый факт о состоянии (ветка, статус, путь, число) — подстрока результата инструмента с указанным `call_id`. Нет оснований — `unknown` и уточняющий вопрос |

## Что нужно

- Go 1.25+, `git` в PATH; `gh` не нужен.
- Ключ DeepSeek в переменной `DEEPSEEK_API_KEY` (файл `.env` в корне курса или
  `.env.local` рядом) — или локальная модель: `LOCAL_LLM_BASE_URL`,
  `LOCAL_LLM_MODEL` и флаг `-llm local`.
- Для плотного поиска — `uv` и сайдкар `embedder/server.py` (модель
  `intfloat/multilingual-e5-base`, ≈1,1 ГБ, CPU). Без него индекс и поиск
  работают на BM25.
- Каталог курса: куратор ищет корень по `CLAUDE.md` + `scripts/aitr.ps1`
  вверх от текущего каталога или по флагу `-root`.

## Быстрый старт

```powershell
go build -o curator.exe ./cmd/curator

uv run embedder/server.py                    # по желанию, в отдельном окне
.\curator.exe index -root D:\Projects\AITrenning          # data/course.db + data/schema/*.md
.\curator.exe calibrate -root D:\Projects\AITrenning -write  # порог «не знаю» (нужен эмбеддер)

.\curator.exe help "в каком задании появились права этапа и чем они держатся?"
.\curator.exe help "на какой ветке корень и какие задания не done?"
.\curator.exe serve -addr 127.0.0.1:8781     # окно: http://127.0.0.1:8781
.\curator.exe mcp                            # MCP-сервер course по stdio
.\curator.exe mcp -http 127.0.0.1:8782       # то же по HTTP
.\curator.exe bench -trial 14 -out examples/help   # испытание И-14
```

Данные (`data/`) — вне git; индекс помнит коммит и хеш документов, по которым
собран, и `index -check` говорит, устарел ли он.

## Как устроено

```
cmd/curator/        index | calibrate | help | serve | mcp | bench | snapshot
internal/
  course/   каталог курса: корень, задания (assignment.json), реестр и расхождения, последний прогон (report.json)
  gitx/     git через os/exec: ветка, статус, журнал, diff, изменённые файлы; ограда путей
  docs/     markdown → документ с разделами; генераторы схем; индекс с коммитом и хешем документов
  ps/       запуск aitr из PowerShell 5.1 (-NoProfile, UTF-8), подстановка для тестов
  project/  инструменты MCP-сервера course и срезы по ролям
  help/     агент /help: Preload docs_search, help_answer с проверкой цитат и фактов состояния
  web/      окно куратора: страница, /api/ask по SSE, /api/state
  bench/    стенд: дорожки, отличающиеся одним механизмом; И-14
  kb, cite, embed, agent, llm, tools, mcp, features, tokens, words, db, corpus, consent — ядро из AnimalGuide (v30), см. CHANGELOG.md
web/index.html      страница окна (без CDN, CSP, тёмная тема)
eval/help.json      24 контрольных вопроса: 10 test, 6 dev, 4 state, 4 out
testdata/course/    мини-курс для тестов без сети
embedder/           сайдкар эмбеддингов (Python, uv)
```

Два механизма с выключателями (`-features`): `help.rag` (поиск по
документации) и `help.tools` (инструменты состояния). Стенд И-14 сравнивает
дорожки, отличающиеся ровно одним из них.

## Испытание И-14

`eval/help.json`: 10 вопросов `test` об устройстве курса (с дословными
`evidence`-цитатами из файлов — тест `TestEvidenceExists` проверяет, что они
есть в каталоге), 6 `dev` для калибровки порога, 4 `state` (эталон считает код
в момент прогона: ветка — `git`, статусы — `assignment.json`, число заданий —
реестр, последний прогон — `report.json`) и 4 `out` вне курса.

Пороги: источники и дословные цитаты 10/10; `must` ≥ 9/10; state — 4/4 из
инструмента и равны эталону; out → `unknown` ≥ 3/4; ложных `unknown` ≤ 1;
«не проверено» 0; запросов на ход ≤ 4. Контроль: без `help.rag` точность
падает больше шума; без `help.tools` ветка не из инструмента. Отчёт живого
прогона — `examples/help/`.

## Тесты

```powershell
go vet ./... ; go test ./...
```

Сети и Python тестам не нужно: модель — `llmtest.Fake`, эмбеддер — `hash`,
git — настоящий во временных каталогах, PowerShell — `ps.Fake`, каталог —
`testdata/course`.
