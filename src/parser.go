package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"google.golang.org/protobuf/proto"

	pb "memc_loader/src/appsinstalled"
)

type appsInstalled struct {
	devType string
	devID   string
	lat     float64
	lon     float64
	apps    []uint32
}

func parseLine(line string) (*appsInstalled, bool) {
	line = strings.TrimSpace(line)
	parts := strings.Split(line, "\t")
	if len(parts) < 5 {
		return nil, false
	}
	devType, devID, rawLat, rawLon, rawApps := parts[0], parts[1], parts[2], parts[3], parts[4]
	if devType == "" || devID == "" {
		return nil, false
	}
	lat, err := strconv.ParseFloat(rawLat, 64)
	if err != nil {
		log.Printf("Некорректная широта: %s", line)
		return nil, false
	}
	lon, err := strconv.ParseFloat(rawLon, 64)
	if err != nil {
		log.Printf("Некорректная долгота: %s", line)
		return nil, false
	}
	var apps []uint32
	for _, a := range strings.Split(rawApps, ",") {
		a = strings.TrimSpace(a)
		n, err := strconv.ParseUint(a, 10, 32)
		if err != nil {
			log.Printf("Пропускаем нечисловой app id %q в строке: %s", a, line)
			continue
		}
		apps = append(apps, uint32(n))
	}
	return &appsInstalled{devType: devType, devID: devID, lat: lat, lon: lon, apps: apps}, true
}

func serialize(ai *appsInstalled) (string, []byte, error) {
	ua := &pb.UserApps{
		Lat:  proto.Float64(ai.lat),
		Lon:  proto.Float64(ai.lon),
		Apps: ai.apps,
	}
	packed, err := proto.Marshal(ua)
	if err != nil {
		return "", nil, err
	}
	key := fmt.Sprintf("%s:%s", ai.devType, ai.devID)
	return key, packed, nil
}
