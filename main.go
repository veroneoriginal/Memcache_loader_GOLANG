# main.go
// точка входа, флаги, оркестрация (= memc_load.py)
// memc_loader — конкурентный загрузчик логов трекера мобильных приложений в memcache.
//
// Архитектура (Go-версия):
//   - Главная горутина читает .tsv.gz файлы и парсит строки
//   - На каждый тип устройства запускается горутина-писатель (writer.go)
//   - Данные передаются через каналы (вместо Python-очередей)
//   - Несколько файлов обрабатываются параллельно через sync.WaitGroup
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"sync"
)

func dotRename(path string) {
	dir, fn := filepath.Split(path)
	newPath := filepath.Join(dir, "."+fn)
	if err := os.Rename(path, newPath); err != nil {
		log.Printf("Ошибка переименования %s: %v", path, err)
	}
}

func main() {
	pattern := flag.String("pattern", "/data/appsinstalled/*.tsv.gz", "Паттерн файлов")
	logFile := flag.String("log", "", "Файл логов (по умолчанию stdout)")
	dryRun  := flag.Bool("dry", false, "Dry-run: логировать без записи")
	workers := flag.Int("workers", 4, "Количество параллельных файлов")
	idfa    := flag.String("idfa", "127.0.0.1:33013", "Адрес memcache для idfa")
	gaid    := flag.String("gaid", "127.0.0.1:33014", "Адрес memcache для gaid")
	adid    := flag.String("adid", "127.0.0.1:33015", "Адрес memcache для adid")
	dvid    := flag.String("dvid", "127.0.0.1:33016", "Адрес memcache для dvid")
	flag.Parse()

	if *logFile != "" {
		f, err := os.OpenFile(*logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("Не удалось открыть файл логов: %v", err)
		}
		defer f.Close()
		log.SetOutput(f)
	}

	deviceMemc := map[string]string{
		"idfa": *idfa,
		"gaid": *gaid,
		"adid": *adid,
		"dvid": *dvid,
	}

	files, err := filepath.Glob(*pattern)
	if err != nil || len(files) == 0 {
		log.Println("Нет файлов для обработки")
		return
	}
	log.Printf("Найдено файлов: %d", len(files))

	// fileResult хранит итог обработки одного файла
	type fileResult struct {
		fn        string
		processed int
		errors    int
	}

	// семафор ограничивает число параллельных горутин (аналог multiprocessing.Pool)
	sem := make(chan struct{}, *workers)
	var wg sync.WaitGroup

	var mu sync.Mutex
	var allResults []fileResult

	for _, fn := range files {
		fn := fn
		sem <- struct{}{}
		wg.Add(1)

		go func() {
			defer wg.Done()
			defer func() { <-sem }()

			log.Printf("Обработка: %s", fn)
			processed, errors := processFile(fn, deviceMemc, *dryRun)

			mu.Lock()
			allResults = append(allResults, fileResult{fn, processed, errors})
			mu.Unlock()
		}()
	}

	wg.Wait()

	// переименовываем файлы после завершения всех горутин
	for _, r := range allResults {
		if r.processed == 0 {
			dotRename(r.fn)
			continue
		}
		errRate := float64(r.errors) / float64(r.processed)
		if errRate < normalErrRate {
			log.Printf("Приемлемая доля ошибок (%.4f). Успешная загрузка: %s", errRate, r.fn)
		} else {
			log.Printf("Высокая доля ошибок (%.4f > %.2f). Неудачная загрузка: %s",
				errRate, normalErrRate, r.fn)
		}
		dotRename(r.fn)
	}
}
