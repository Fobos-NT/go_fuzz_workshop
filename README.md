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
cd /go/src/go_fuzz_workshop/artifacts/fuzz
mkdir -p /go/src/go_fuzz_workshop/artifacts/fuzz/testdata/fuzz/FuzzParseJSONUsual
cp /go/src/go_fuzz_workshop/artifacts/fuzz/corpus/* /go/src/go_fuzz_workshop/artifacts/fuzz/testdata/fuzz/FuzzParseJSONUsual/
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

Перед началом его использования требуется описать структуру тестовых данных - это делается в proto-файле (artifacts\json.proto). 
Proto-файл — это текстовый файл в формате Protocol Buffers ([Protobuf](https://protobuf.dev/)). Он используется для описания структуры данных и API.

```
message JSON {
  oneof value {
    bool bool_value = 1;
    double number_value = 2;
    string string_value = 3;
    Object object_value = 4;
    Array array_value = 5;
    NullValue null_value = 6;
  }
}

message Object {
  repeated Field fields = 1;
}

message Field {
  string key = 1;
  JSON value = 2;
}

message Array {
  repeated JSON values = 1;
}

message NullValue {}
```

Главные особенности:

• Языковая независимость: Описание пишется один раз, а специальный компилятор (protoc) автоматически генерирует из него готовый программный код (структуры, классы, методы сериализации) для разных языков программирования (C#, Java, Python, Go и др.)

• Бинарная сериализация: Данные упаковываются в компактный бинарный формат, который передается по сети намного быстрее и весит меньше

• Контракт для gRPC: Часто используется в связке с фреймворком gRPC, где в .proto файле прописываются не только форматы сообщений, но и методы удаленного вызова процедур.

• Обратная совместимость: Позволяет безопасно добавлять новые поля в структуру данных, не ломая старые версии приложений.

Сгенерируем код с помощью компилятора protoc:

```bash
cd /go/src/go_fuzz_workshop

protoc \
    --go_out=. \
    --go_opt=paths=source_relative \
    artifacts/json.proto
```

В сгенерированном файле появятся готовые структуры (классы/объекты) для сериализации/десериализации данных.

## 8. Запуск structed-aware фаззинга

Теперь наша фаззинг-функция будет выглядеть так:

```go
func FuzzParseJSON(f *testing.F) {
	f.Fuzz(func(t *testing.T, message *artifacts.JSON) {
		gjson.Parse(serializeJSON(message))
	})
}
```

Таким образом, наша фаззинг-обёртка работает по следующему алгоритму:
1) получили случайные байты от фаззера
2) попробовали десериализовать их из бинарного protobuf-формата в proto-структуру — логическое представление JSON-структуры, если не получилось — остановились
3) случайно изменили получившуюся proto-структуру
4) используя кастомный сериализатор, трансформировали proto представление JSON-объекта в строку с JSON
5) подали получившуюся строку с JSON на вход целевой функции `gjson.Parse`

Наш кастомный сериализатор из логического представления в формате proto в строку находится в файле фаззинг-обертки в корне проекта (fuzz_test.go)
Стоит обратить внимание на то, что аргумент, передаваемый целевой функции теперь имеет не стандартный тип, а именно protobuf-сообщение. Обычными средствами (go test) запустить такую обертку не получится.

Мы будем использовать [форк go-118-fuzz-build](https://github.com/Fobos-NT/go-118-fuzz-build) 

go-118-fuzz-build — это специальный инструмент для Go, который позволяет компилировать стандартные фаззинг-тесты Go (появившиеся в версии 1.18+) в формат движка libFuzzer

Сборка:

```bash
cd /go/src/go_fuzz_workshop

go-118-fuzz-build \
    -proto \
    -proto_format binary \
    -func FuzzParseJSON \
    -o gjson_fuzz.a \
    go_fuzz_workshop

clang++ \
    -fsanitize=fuzzer,address \
    -o gjson_fuzz \
    gjson_fuzz.a

./gjson_fuzz artifacts
```

Первая команда

Она берёт Go-проект/пакет go_fuzz_workshop и собирает из него статическую библиотеку

1) func FuzzParseJSON — указывает fuzzing-функцию, которую нужно использовать
2) gjson_fuzz.a — имя выходного файл
3) proto — используется protobuf формат
4) proto_format binary — бинарный формат
5) go_fuzz_workshop — исходный Go-пакет, который компилируется

Вторая команда

Используется Clang, чтобы собрать из библиотеки исполняемый файл

-fsanitize=fuzzer,address

Это включает два санитайзера:

fuzzer - подключает инструментацию для LLVM libFuzzer — движок, который будет мутировать входные данные и подавать их на вход
address - подключает AddressSanitizer (ASan). Он обнаруживает ошибки памяти

Третья команда

Запускает фаззинг с сохранением корпуса в папку artifacts


Сборка с поддержкой покрытия

```bash
cd  /go/src/go_fuzz_workshop

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

Пересборка требуется так как поддержка сбора покрытия - это дополнительная инструментация кода, которой "из коробки" нет. Пересборка с указанием флага `-sanitizer coverage` позволяет включить сбор покрытия и получить статистику по выполнению целевого кода.