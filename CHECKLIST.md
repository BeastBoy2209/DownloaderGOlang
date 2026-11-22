# SIS Checkpoint 2 — Чеклист

Формат: `[требование] - [сделано/не сделано] - {путь к файлу:строка}`

Источник: `SIS_CP2_SPECIFICATION.md`. Тесты лежат в `internal/usecase/pool_test.go` (далее `pool_test.go`).

## 3. Обязательная функциональность

1. Сохранена доменная обработка из CP1 (разбор URL) - сделано - {internal/domain/models.go:35 (`Analyze`), :49 (`Process`)}
2. Создание и валидация задач - сделано - {internal/usecase/service.go:21, internal/domain/models.go:29}
3. Хранение задач и результатов в памяти - сделано - {internal/repository/memory.go:10-60}
4. Отправка задач на асинхронную обработку - сделано - {internal/usecase/pool.go:54 (`Submit`)}
5. Ограниченная очередь ожидающих задач - сделано - {internal/usecase/pool.go:40 (`make(chan int, capacity)`)}
6. Несколько воркеров берут задачи из очереди - сделано - {internal/usecase/pool.go:45-48, :87 (`worker`)}
7. Разные воркеры обрабатывают независимые задачи одновременно - сделано - {internal/usecase/pool.go:94-104; тест `TestConcurrentProcessing` pool_test.go:118}
8. Состояние задачи обновляется по жизненному циклу - сделано - {internal/domain/models.go:63 (`Enqueue`), :82 (`Start`), :91 (`Finish`)}
9. Результат сохраняется в нужной задаче - сделано - {internal/usecase/pool.go:103; тест `TestSubmitAndProcessMany` pool_test.go:75}
10. Задачи можно получать во время обработки других - сделано - {internal/repository/memory.go:29 (`Get` под `RLock`); тест `TestConcurrentAccess` pool_test.go:260}
11. Безопасный конкурентный доступ к общим данным - сделано - {internal/repository/memory.go:11, :21, :30, :40, :52}
12. Одна задача не обрабатывается двумя воркерами одновременно - сделано - {internal/repository/memory.go:51 (атомарный `Update`), internal/domain/models.go:82 (`Start`); тест `TestSameTaskSubmittedConcurrently` pool_test.go:304}
13. Обработка отправки задач в неверном состоянии - сделано - {internal/usecase/pool.go:61-63, internal/domain/models.go:63-67; тест `TestSubmitInvalidAndDuplicate` pool_test.go:179}
14. Обработка отправки при полной очереди - сделано - {internal/usecase/pool.go:64-71 (`select`/`default`, откат в `Pending`)}
15. Контролируемое завершение воркеров - сделано - {internal/usecase/pool.go:76-84 (`Stop`)}
16. После остановки новые задачи не принимаются - сделано - {internal/usecase/pool.go:57-59 (`ErrPoolStopped`)}
17. Принятые задачи не теряются при штатной остановке - сделано - {internal/usecase/pool.go:80 (`close`), :87-92 (воркеры дочитывают закрытый канал); тест `TestStopFinishesAcceptedTasks` pool_test.go:336}
18. `Submit` не ждёт завершения обработки - сделано - {internal/usecase/pool.go:54-74; тест `TestSubmitDoesNotWait` pool_test.go:99}

## 4. Пул воркеров

1. Минимум 2 воркера - сделано - {internal/usecase/pool.go:34-36 (`workers < 2` → `ErrInvalidPool`)}
2. Воркеры обрабатывают разные задачи параллельно - сделано - {pool_test.go:118 (`TestConcurrentProcessing`)}
3. Задачи приходят через канал-очередь - сделано - {internal/usecase/pool.go:40, :89 (`range p.queue`)}
4. Воркеры не завершаются после одной задачи - сделано - {internal/usecase/pool.go:89-91}
5. Число горутин-воркеров контролируется - сделано - {internal/usecase/pool.go:45-48; тест `TestWorkerLimit` pool_test.go:145}
6. Воркеры могут завершиться при остановке - сделано - {internal/usecase/pool.go:76-84}
7. Завершение скоординировано (`WaitGroup`) - сделано - {internal/usecase/pool.go:45 (`wg.Add`), :88 (`wg.Done`), :83 (`wg.Wait`)}
8. Нет `go process(task)` на каждую задачу - сделано - {internal/usecase/pool.go:47 (горутины только при старте пула)}

## 5. Ограниченная очередь

1. Чётко определённая конечная ёмкость - сделано - {internal/usecase/pool.go:33-40, cmd/app/main.go:26}
2. Задачи сначала попадают в очередь - сделано - {internal/usecase/pool.go:64-66}
3. Воркеры берут задачи из очереди - сделано - {internal/usecase/pool.go:89}
4. Корректная обработка полной очереди (отклонение с ошибкой) - сделано - {internal/usecase/pool.go:67-70; README.md:33 «Full queue»; тест `TestQueueFull` pool_test.go:211}
5. Ёмкость не растёт бесконечно - сделано - {internal/usecase/pool.go:40 (фиксированный буфер)}
6. Нет неограниченного числа горутин в обход лимита - сделано - {internal/usecase/pool.go:54-74 (`Submit` не создаёт горутин)}
7. Отклонённая задача остаётся в корректном состоянии (`Pending`) - сделано - {internal/usecase/pool.go:69, internal/domain/models.go:72 (`Dequeue`)}

## 6. Жизненный цикл задачи

1. Состояния `Pending`, `Queued`, `Processing`, `Completed` (+ `Failed`) - сделано - {internal/domain/models.go:7-13}
2. Состояние отражает реальный процесс - сделано - {internal/domain/models.go:63-101}
3. Изменения состояния синхронизированы - сделано - {internal/repository/memory.go:51-64 (`Update`)}
4. Несколько воркеров не обрабатывают одну задачу - сделано - {internal/domain/models.go:82-88 (`Start` только из `Queued`)}
5. Завершённые задачи сохраняют результат - сделано - {internal/domain/models.go:100-101 (`Finish`)}
6. Недопустимые переходы возвращают ошибку - сделано - {internal/domain/errors.go:8, internal/domain/models.go:65, :74, :84, :93}
7. Повторная отправка `Processing` / `Completed` - ошибка - сделано - {pool_test.go:179 (`TestSubmitInvalidAndDuplicate`)}
8. Сбой обработки не помечает задачу как `Completed` (`Failed`) - сделано - {internal/domain/models.go:95-98; тест `TestFailedProcessing` pool_test.go:245}
9. Жизненный цикл описан в README - сделано - {README.md:38 «Task lifecycle»}

## 7. Синхронизация и общее состояние

1. Общие изменяемые данные защищены - сделано - {internal/repository/memory.go:8-13}
2. Конкурентные чтения/записи map синхронизированы - сделано - {internal/repository/memory.go:21, :30, :40, :52}
3. Обновления состояния без гонок - сделано - {internal/repository/memory.go:51-64; `go test -race ./...`}
4. Результаты не перезаписываются чужими воркерами - сделано - {internal/usecase/pool.go:97-103 (обновление по `id`); тест pool_test.go:75}
5. Конкурентное получение корректно - сделано - {internal/repository/memory.go:29-37 (`RLock`, копия по значению)}
6. Атомарные операции не разбиты на read + write - сделано - {internal/repository/memory.go:51-64 (`Update`)}
7. Нет взаимных блокировок - сделано - {internal/usecase/pool.go:54-74 (неблокирующий `Submit`), :76-84; тест `TestSubmitRacesWithStop` pool_test.go:401}
8. Нет блокировки на время тяжёлой обработки - сделано - {internal/usecase/pool.go:100-102 (`analyze` вне лока)}

## 8. Жизненный цикл воркеров и остановка

1. Запуск воркеров - сделано - {internal/usecase/pool.go:33-50; тест `TestPoolStartup` pool_test.go:62}
2. Воркеры принимают и обрабатывают задачи - сделано - {internal/usecase/pool.go:87-104}
3. Остановка приёма новых задач - сделано - {internal/usecase/pool.go:77-80}
4. Доработка уже принятых задач - сделано - {internal/usecase/pool.go:89 (`range` по закрытому буферу)}
5. Ожидание завершения воркеров - сделано - {internal/usecase/pool.go:83 (`wg.Wait`)}
6. Нет паники при отправке в закрытый канал (гонка `Submit` и `Stop`) - сделано - {internal/usecase/pool.go:55-56 (`RLock`), :77 (`Lock`); тест `TestSubmitRacesWithStop` pool_test.go:401}
7. Повторный `Stop` безопасен - сделано - {internal/usecase/pool.go:78 (`if !p.closed`); тест `TestStopTwice` pool_test.go:382}
8. Владелец канала понятен (создаёт и закрывает только `Pool`) - сделано - {internal/usecase/pool.go:17-21 (комментарий), :40, :80}

## 9. Автотесты

1. Запуск пула - сделано - {pool_test.go:62 (`TestPoolStartup`)}
2. Асинхронная отправка - сделано - {pool_test.go:99 (`TestSubmitDoesNotWait`)}
3. Обработка нескольких задач - сделано - {pool_test.go:75 (`TestSubmitAndProcessMany`)}
4. Реальное перекрытие двух задач (без `sleep`) - сделано - {pool_test.go:118 (`TestConcurrentProcessing`)}
5. Корректные результаты отдельных задач - сделано - {pool_test.go:75}
6. Корректные состояния после обработки - сделано - {pool_test.go:75, :145, :245}
7. Неверные и дублирующиеся отправки - сделано - {pool_test.go:179 (`TestSubmitInvalidAndDuplicate`), :304}
8. Полная очередь - сделано - {pool_test.go:211 (`TestQueueFull`)}
9. Безопасный конкурентный доступ - сделано - {pool_test.go:260 (`TestConcurrentAccess`), internal/repository/memory_test.go (`TestUpdateAtomic`, `TestConcurrentCreateUniqueIDs`)}
10. Завершение воркеров после остановки - сделано - {pool_test.go:336 (`TestStopFinishesAcceptedTasks`), :388 (`TestWorkersTerminate`)}
11. Отклонение отправки после остановки - сделано - {pool_test.go:369 (`TestSubmitAfterStop`)}
12. Тесты CP1 по-прежнему проходят - сделано - {internal/usecase/service_test.go, internal/domain/models_test.go, internal/repository/memory_test.go}
13. Тесты не зависят от порядка завершения задач - сделано - {pool_test.go:75 (проверка по ID, а не по порядку)}
14. Координация тестов через каналы и `WaitGroup`, а не `sleep` - сделано - {pool_test.go:45 (`blockingAnalyzer`), :55 (`gateRelease`)}
15. `go test -race ./...` без гонок - сделано - {проверено командой, QA-прогон `-count=20`}

## 10. Технические требования

1. Используется Go-модуль из CP1 - сделано - {go.mod:1}
2. Сохранена доменная логика - сделано - {internal/domain/models.go}
3. Воркер-пул + ограниченная очередь на каналах - сделано - {internal/usecase/pool.go}
4. Ожидаемые ошибки через обычную обработку ошибок - сделано - {internal/domain/errors.go:5-12}
5. Без `panic` для ожидаемых ошибок - сделано - {весь код; `grep panic` пуст}
6. Без `sleep` для имитации конкурентности - сделано - {internal/usecase/pool.go (нет `time.Sleep`)}
7. Код отформатирован (`gofmt -l .` пуст) - сделано - {проверено командой}
8. `go vet ./...` проходит - сделано - {проверено командой}
9. `go test ./...` проходит - сделано - {проверено командой}
10. `go test -race ./...` проходит - сделано - {проверено командой}

## 11. README

1. Назначение приложения и логика обработки - сделано - {README.md:1-9}
2. Как создаются и отправляются задачи - сделано - {README.md:10-20 «Running»}
3. Как работает пул воркеров - сделано - {README.md:22-36 «Concurrency design»}
4. Сколько воркеров - сделано - {README.md:30}
5. Ёмкость очереди - сделано - {README.md:32}
6. Что происходит при полной очереди - сделано - {README.md:33}
7. Как меняются состояния - сделано - {README.md:38-50 «Task lifecycle»}
8. Как синхронизируются общие данные - сделано - {README.md:52-64 «Synchronization»}
9. Как останавливаются воркеры - сделано - {README.md:65-70 «Stopping»}
10. Как запустить приложение и тесты - сделано - {README.md:10-20, :80-87 «Verification»}
11. Простая диаграмма очереди, воркеров и хранилища - сделано - {README.md:24-28}

## 12. Сдача и защита

1. Репозиторий из CP1, ветка `checkpoint2` - сделано - {git: ветка `checkpoint2`, коммит `c4a2d2a`}
2. Ссылка на репозиторий и коммит/тег в Microsoft Teams - не сделано - {нужно запушить ветку и отправить ссылку вручную}
3. Индивидуальная защита (~2 минуты, обязательна) - не сделано - {проходит студент лично}
