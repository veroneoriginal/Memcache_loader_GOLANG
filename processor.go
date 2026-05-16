# processor.go
// обработка одного файла (= processor.py)
package main

import (
	"bufio"
	"compress/gzip"
	"log"
	"os"
)

// processFile читает один .tsv.gz файл и раздаёт записи горутинам-писателям.
// Возвращает суммарное количество обработанных записей и ошибок.
func processFile(fn string, deviceMemc map[string]string, dryRun bool) (processed, errors int) {
	// создаём канал и запускаем горутину-писатель для каждого типа устройства
	channels := make(map[string]chan record)
	resultCh := make(chan result, len(deviceMemc))

	for devType, addr := range deviceMemc {
		ch := make(chan record, chanBuffer)
		channels[devType] = ch
		go writer(addr, ch, resultCh, dryRun)
	}

	// читаем файл и раскидываем по каналам
	parseErrors := 0

	f, err := os.Open(fn)
	if err != nil {
		log.Printf("Ошибка открытия файла %s: %v", fn, err)
	} else {
		gz, err := gzip.NewReader(f)
		if err != nil {
			log.Printf("Ошибка чтения gzip %s: %v", fn, err)
		} else {
			scanner := bufio.NewScanner(gz)
			for scanner.Scan() {
				line := scanner.Text()
				if line == "" {
					continue
				}
				ai, ok := parseLine(line)
				if !ok {
					parseErrors++
					continue
				}
				ch, ok := channels[ai.devType]
				if !ok {
					log.Printf("Неизвестный тип устройства: %s", ai.devType)
					parseErrors++
					continue
				}
				key, packed, err := serialize(ai)
				if err != nil {
					log.Printf("Ошибка сериализации: %v", err)
					parseErrors++
					continue
				}
				ch <- record{key: key, packed: packed}
			}
			gz.Close()
		}
		f.Close()
	}

	// закрываем каналы — сигнал горутинам завершить работу
	for _, ch := range channels {
		close(ch)
	}

	// собираем результаты от всех горутин-писателей
	for range deviceMemc {
		r := <-resultCh
		processed += r.processed
		errors += r.errors
	}
	errors += parseErrors

	return processed, errors
}
