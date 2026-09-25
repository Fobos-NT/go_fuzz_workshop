# Тренировочный стенд: фаззинг `gjson` со встроенным фаззером и использованием кастомного protobuf мутатора.

Стенд знакомит с coverage-guided фаззингом Go-кода на примере функции `gjson.Parse`.

- команды сборки и запуска выполняются из корня
- команды фаззинга выполняются внутри контейнера из /go/src/gjson-1.18.0

## 1. Сборка образа

```bash
docker build --tag=gjson_workshop_img .
```

- `--tag` задаёт имя
- `.` передаёт текущий каталог как точку сборки

## 2. Запуск контейнера

```bash
docker run -it -v "$(pwd)/artifacts:/home/fuzz/artifacts:ro" --name=gjson_fuzz gjson_workshop_img
```

- `-it` запускает контейнер с терминалом
- `--name` задаёт имя контейнера
- `-v` подключает локальный каталог `artifacts` к `/home/fuzz/artifacts` внутри контейнера
- `:ro` монтирует каталог только для чтения

## 3. Подготовка fuzz-target

```bash

mkdir -p testdata/fuzz/FuzzParseJSONUsual
cp artifacts/corpus/* testdata/fuzz/FuzzParseJSONUsual/
```
`testdata/fuzz/FuzzParseJSON` — каталог seed корпуса. Фаззер берёт из него стартовые тесткейсы для `FuzzParseJSONUsual`

## 4. Описание fuzz-функции


```go
func FuzzParseJSON(f *testing.F) {
	f.Fuzz(func(t *testing.T, orig string) {
		Parse(orig)
	})
}
```

- `f *testing.F` — объект фаззера
- `f.Fuzz(...)` задаёт тестирующую функцию
- `orig string` — вход, который генерирует фаззер
- `Parse(orig)` — целевая функция

## 5. Запуск фаззинга

Основные показатели строки статистики:

| Показатель | Значение |
|---|---|
| `elapsed` | время работы фаззера |
| `execs` | число выполнений fuzz-target |
| `new interesting` | новые входы, которые дали покрытие |

```bash
go test -fuzz=FuzzParseJSONUsual
```

- `-run=FuzzParseJSONUsual` выбирает конкретный fuzz-тест

Остановка: `Ctrl+C`. Найденные интересные входы сохраняются в кэше Go

## 6. Сбор покрытия

Копируем найденные входы из кэша обратно в корпус и генерим HTML-отчёт:

```bash
cp $(go env GOCACHE)/fuzz/$(go list)/FuzzParseJSONUsual/* testdata/fuzz/FuzzParseJSONUsual
go test -coverprofile=coverage.out -run=FuzzParseJSONUsual -v
go tool cover -html=coverage.out -o ./coverage.html
```

- `go env GOCACHE` — каталог кэша Go
- `go list` — путь пакета
- `-coverprofile` пишет профиль покрытия в файл
- `-run=FuzzParseJSONUsual` прогоняет корпус без мутаци
- `go tool cover -html` генерит HTML-отчёт

На хосте:

```bash
docker cp gjson_fuzz:/go/src/gjson-1.18.0/coverage.html .
```

Отчет появится в текущкй папке

## 7. Фаззинг с использованием protobuf mutator

Structured-aware fuzzing с Protobuf mutator — это способ фаззить код не случайными данными, а данными, которые представляют собой четкую структуру, что важно в нашем случае с фаззингом парсера JSON. Мутатор генерит валидные/полувалидные данные, которые начнут парситься, а не сразу отбросятся парсером, что позволяет покрыть большее количество кода и краевых случаев.

## 8. Запуск structed-aware фаззинга

Генерация protobuf-кода:

```bash
protoc --go_out=. --go_opt=paths=source_relative artifacts/json.proto
```

Теперь наша фаззинг функция будет выглядеть так:

```go
f.Fuzz(func(t *testing.T, data []byte) {
		var message JSON

		if err := proto.Unmarshal(data, &message); err != nil {
			return
		}

		m := mutator.New(1, 4096)

		if err := m.MutateProto(&message); err != nil {
			return
		}

		json := serializeJSON(&message)

		gjson.Parse(json)
	})
```
1) получили случайные байты
2) попробовали превратить их в JSON-структуру (если не получилось > остановились)
3) случайно изменили получившуюся структуру
4) превратили её обратно в JSON
5) попробовали распарсить JSON

Запуск:

```bash
go test -fuzz=FuzzParseJSON -fuzztime=5m ./artifacts
```

Сбор покрытия:
```bash
cp /root/.cache/go-build/fuzz/github.com/tidwall/gjson/artifacts/FuzzParseJSON/* artifacts/testdata/fuzz/FuzzParseJSON/
go test -coverprofile=coverage.out -run=FuzzParseJSON -v ./artifacts
go tool cover -html=coverage.out -o ./coverage.html
```
На хосте:
```bash
docker cp gjson_fuzz:/go/src/gjson-1.18.0/coverage.html .
```