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
docker run -it --name gjson-fuzz gjson_workshop_img
```

- `-it` запускает контейнер с терминалом


## 3. Подготовка fuzz-target

```bash
cd artifacts/fuzz
mkdir -p testdata/fuzz/FuzzParseJSONUsual
cp corpus/* testdata/fuzz/FuzzParseJSONUsual/
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

- -fuzz запускает указанную фаззинг-функцию и генерирует входные данные

Остановка: `Ctrl+C`. Найденные интересные входы сохраняются в кэше Go

## 6. Сбор покрытия

Копируем найденные входы из кэша обратно в корпус и генерим HTML-отчёт:

```bash
cp $(go env GOCACHE)/fuzz/$(go list)/FuzzParseJSONUsual/* testdata/fuzz/FuzzParseJSONUsual
go test -coverprofile=coverage.out -coverpkg=github.com/tidwall/gjson -run=FuzzParseJSONUsual
go tool cover -html=coverage.out -o ./coverage.html
```

- `go env GOCACHE` — каталог кэша Go
- `go list` — путь пакета
- `-coverprofile` пишет профиль покрытия в файл
- `-run=FuzzParseJSONUsual` прогоняет корпус без мутаци
- `go tool cover -html` генерит HTML-отчёт

На хосте:

```bash
docker cp gjson-fuzz:/go/src/go_fuzz_workshop/artifacts/fuzz/coverage.html .
```

Отчет появится в текущкй папке

## 7. Фаззинг с использованием protobuf mutator

Structured-aware fuzzing с Protobuf mutator — это способ фаззить код не случайными данными, а данными, которые представляют собой четкую структуру, что важно в нашем случае с фаззингом парсера JSON. Мутатор генерит валидные/полувалидные данные, которые начнут парситься, а не сразу отбросятся парсером, что позволяет покрыть большее количество кода и краевых случаев.

## 8. Запуск structed-aware фаззинга


Теперь наша фаззинг функция будет выглядеть так:

```go
func FuzzParseJSON(f *testing.F) {
	f.Fuzz(func(t *testing.T, message *artifacts.JSON) {
		gjson.Parse(serializeJSON(message))
	})
}
```
1) получили случайные байты
2) попробовали десериализовать их из бинарного protobuf-формата в proto-структуру — логическое представление JSON-структуры, если не получилось — остановились
3) случайно изменили получившуюся proto-структуру
4) сериализовали её обратно в JSON
5) попробовали распарсить получившийся JSON

Запуск:

```bash
cd ../../
mkdir -p out coverage
OUT="$PWD/out" \
go-118-fuzz-build \
  -sanitizer coverage \
  -proto \
  -proto_format binary \
  -coverpkg 'go_fuzz_workshop/...,github.com/tidwall/gjson' \
  -o FuzzParseJSON.cover \
  -func FuzzParseJSON \
  .

  ./gjson_fuzz artifacts
```

Сбор покрытия:
```bash
FUZZ_CORPUS_DIR="$PWD/artifacts" \
./out/FuzzParseJSON.cover \
-test.run=TestFuzzCorpus \
-test.gocoverdir="$PWD/coverage"

go tool covdata textfmt \
  -i="$PWD/coverage" \
  -o="$PWD/coverage.out"


go tool cover \
	-html="$PWD/coverage.out" \
	-o="$PWD/coverage.html"
```
На хосте:
```bash
docker cp gjson-fuzz:/go/src/go_fuzz_workshop/artifacts/fuzz/coverage.html .
```