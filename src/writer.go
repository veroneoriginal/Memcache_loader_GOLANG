// горутина-писатель в memcache (= writer.py)
package main

import (
	"log"

	"github.com/bradfitz/gomemcache/memcache"
)

// record — сериализованная запись, готовая к записи в memcache.
type record struct {
	key    string
	packed []byte
}

// result — итог работы одной горутины-писателя.
type result struct {
	processed int
	errors    int
}

// writer читает записи из канала и пишет их в memcache батчами.
// Когда канал закрыт — дописывает остаток и отправляет итог в resultCh.
func writer(addr string, ch <-chan record, resultCh chan<- result, dryRun bool) {
	var processed, errors int

	var mc *memcache.Client
	if !dryRun {
		mc = memcache.New(addr)
	}

	batch := make(map[string][]byte, batchSize)

	flush := func() {
		if len(batch) == 0 {
			return
		}
		if dryRun {
			processed += len(batch)
			batch = make(map[string][]byte, batchSize)
			return
		}
		for key, packed := range batch {
			err := mc.Set(&memcache.Item{Key: key, Value: packed})
			if err != nil {
				log.Printf("Ошибка записи в memc %s key=%s: %v", addr, key, err)
				errors++
			} else {
				processed++
			}
		}
		batch = make(map[string][]byte, batchSize)
	}

	for rec := range ch {
		if dryRun {
			log.Printf("[dry] %s -> %s", addr, rec.key)
		}
		batch[rec.key] = rec.packed
		if len(batch) >= batchSize {
			flush()
		}
	}
	flush() // дописываем остатки

	resultCh <- result{processed: processed, errors: errors}
}
