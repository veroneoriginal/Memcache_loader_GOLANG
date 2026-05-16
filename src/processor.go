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

	// вспомогательная функция закрытия каналов
	closeChannels := func() {
		for _, ch := range channels {
			close(ch)
		}
	}

	parseErrors := 0

	f, err := os.Open(fn)
	if err != nil {
		log.Printf("Ошибка открытия файла %s: %v", fn, err)
		closeChannels()
		// ждём завершения всех горутин
		for range deviceMemc {
			r := <-resultCh
			processed += r.processed
			errors += r.errors
		}
		return
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		log.Printf("Ошибка чтения gzip %s: %v", fn, err)
		closeChannels()
		for range deviceMemc {
			r := <-resultCh
			processed += r.processed
			errors += r.errors
		}
		return
	}
	defer gz.Close()

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

	// закрываем каналы явно — сигнал горутинам завершить работу
	closeChannels()

	// теперь собираем результаты — горутины уже завершились
	for range deviceMemc {
		r := <-resultCh
		processed += r.processed
		errors += r.errors
	}
	errors += parseErrors

	return processed, errors
}
