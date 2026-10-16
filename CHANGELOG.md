# CHANGELOG — куратор курса

## v31 — копия из v30

Копия из `AnimalGuide_Task30` (импорты `…/internal/X` → `aitr/curator/internal/X`), строки кода / тестов:
`llm` 581/502 (+`llmtest`), `tokens` 269/167, `tools` 425/257 (`tool.go`, `envelope.go`), `agent` 970/608,
`features` 595/224 (`features.go` + свой `catalog.go`), `embed` 783/472 (+`embedtest`), `words` 108/70,
`corpus` 202/64, `db` 172/258, `kb` 2006/896, `cite` 1303/542 (из `rag/`), `mcp` 379/353, `consent` 88/37
(из `lifecycle.go`), `embedder/server.py` 216. Итого ≈ 7 900 строк кода и ≈ 4 450 тестов.

Правки в скопированном коде (кроме импортов):

1. `corpus/corpus.go` — убраны `Species`, `Source*`, `Load/Save/WriteLicense`, `Fetcher.Wikipedia` (нет `store.go`, `wikipedia.go`); оставлены `Doc`, `Section`, `Entry`, `Manifest`, `Text`, `Find`, `Normalize`.
2. `kb` — `eval.go` обрезан до типов вопросов и `Covers` (без `CompareOptions/Report`, нет `compare.go`); из тестов вырезаны прогоны по настоящему корпусу (`realDocs`, `TestRealQuestions`, `TestRealCorpusChunks`, `TestFixedSectionMajority`, `TestMidSentenceAndUnion`), `corpus.SourceWikipedia` → `"wiki"`.
3. `tools` — тесты Википедии/GBIF/`Fetcher`/`LocalTools` и пакет `toolstest` удалены; `InjectedExtract` скопирован константой в `tools_test.go` и `agent/runner_test.go`.
4. `features/catalog.go` — свой: `help.rag`, `help.tools`, `review.rag`, `support.context`, `files.gates`, `close.verify` (все `Default: true`, цены — оценки) + `sources.envelope`, `sources.scan` из v30 (их читает `agent.Runner`); `features_test.go` переписан под новые имена (`TestCatalogBlockOrderFollowsStability`, `TestMCPIsTransport` убраны).
5. `mcp/server.go` — убраны `Fetcher`, `WikiBase/GBIFBase`, `Sources`, `HTTPRequests`; `ServerName` = `course`, версия 31.0.0, инструкция и заголовок — про курс; добавлен конверт `tools.Envelope` по `Spec.Untrusted` в `addTool`; `server_test.go` — свой (транспорт в памяти).
6. `cite/cite.go` — без `retrieve`: `Gate(hits, info, minScore)` по порогу индекса; вырезаны `RAGCite`, `SpeciesMismatch`, `speciesMismatch`, `mismatchProblem`, `Names`, `Past`/`withPast`, судья `Support/Judge`; `defaultClarify` — про задание, а не животное.
7. `cite/search.go` — `SearchTool(s, index, k, obs *Issued)` напрямую над `kb.Searcher`, `issued` → `Issued`, без `PipelineTool`; `ToolName` = `docs_search`, `DefaultIndex` = `structure`, описание — про документы курса.
8. `cite/numbers.go` — вырез `parse/tokenize/wordNumbers` из `rag/rule.go` и `clip` из `rag/judge.go` (нужны `numbersMissing`).
9. `consent/consent.go` — `decisionWords`, `Decided`, `negatedBefore`, `fold`; виды решений — свои константы; в `Approve` добавлены «применя», «публику».
9. `tools/envelope.go` — `TrustNote`: «сведения о животных» → «сведения о проекте», «статьи или базы» → «файлов, документации или вывода команд».
10. `agent/agent.go`, `agent/runner.go` — в `Event` добавлено поле `Result` (полный ответ инструмента, в JSON не уходит): проверка «факт из инструмента» сверяется с полным результатом, а `Detail` остаётся обрезанным для журнала.

### Новое в v31 — `/help`: ассистент, понимающий каталог курса

- **`internal/course`** — каталог курса как проект: корень (по `CLAUDE.md` + `scripts/aitr.ps1`), задания из `assignment.json`, реестр и расхождения с ним, последний контрольный прогон из `report.json` edgerec.
- **`internal/gitx`** — git через `os/exec`: ветка, статус, журнал, diff с пределом, изменённые файлы; `Guard` — ограда путей (корень, без `.git`, `.env*`, `*.local`, `INDEX.md`, реестра).
- **`internal/docs`** — markdown каталога → документ с разделами по заголовкам (виды: card, note, handbook, doc, readme, skill, task, template); «схемы» `assignments.md` и `aitr.md` генерирует код; индекс помнит коммит и хеш документов, `Stale` говорит, устарел ли он. Планы и решения по неделям (`docs/предложение-*`, `docs/решение-*`) не индексируются — они описывают контрольные вопросы самого куратора.
- **`internal/ps`** — запуск `aitr` из PowerShell 5.1 (`-NoProfile`, UTF-8), `Fake` для тестов.
- **`internal/project`** — 13 инструментов MCP-сервера `course` (`git_*`, `repo_*`, `read_file`, `assignments_*`, `assignment_get`, `demo_report`, `aitr_doctor`, `docs_search`) и срезы по ролям; тот же срез идёт агенту и в `.mcp.json` корня для Claude Code.
- **`internal/help`** — агент `/help`: `docs_search` кодом до первого запроса, финишёр `help_answer` — проверка цитат (`cite.CheckWith`) плюс факты о состоянии: ветка, статус, путь, число в ответе обязаны быть подстрокой результата инструмента с указанным `call_id` этого хода; после двух отказов — ответ с пометкой «не проверено». Механизмы `help.rag`, `help.tools`.
- **`internal/web`**, `web/index.html` — окно куратора: лента, источники с раскрываемыми цитатами, факты с `call_id`, журнал вызовов по SSE, вкладки дней 32–35 заглушками.
- **`internal/bench`** — стенд: дорожки, отличающиеся одним механизмом (`CheckLanes` из v30), И-14 на `eval/help.json` (24 вопроса), эталон состояния считает код.
- **`cmd/curator`** — `index`, `calibrate` (порог «не знаю» по dev/out-вопросам, при пересечении шкал — перебор с наибольшим перевесом), `help`, `serve`, `mcp`, `bench`, `snapshot` (снимок модуля в корень репозитория задания, `module aitr/NN-slug`).
- Цена хода `/help`: 1–5 запросов к `deepseek-v4-flash`, ≈ $0.003–0.005; индекс 54 документов — 786 чанков, плотный индекс e5 ≈ 4 мин на CPU, BM25 — 0,5 с.
