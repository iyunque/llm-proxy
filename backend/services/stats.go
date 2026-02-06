package services

import (
	"ai-api-platform/backend/models"
	"ai-api-platform/backend/utils"
	"sync"
	"time"

	"gorm.io/gorm"
)

var (
	statsMutex  sync.Mutex
	memoryStats map[uint]*models.APIStats
)

func InitStats() {
	memoryStats = make(map[uint]*models.APIStats)
	loadStatsFromDB()

	// 启动定时同步协程
	interval := utils.GlobalConfig.Stats.SyncInterval
	if interval <= 0 {
		interval = 60
	}
	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		for range ticker.C {
			SyncStatsToDB()
		}
	}()
}

func loadStatsFromDB() {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	today := time.Now().Format("2006-01-02")
	var stats []models.APIStats
	models.DB.Where("date = ?", today).Find(&stats)

	for i := range stats {
		memoryStats[stats[i].APIEndpointID] = &stats[i]
	}
}

func AddStats(endpointID uint, inputTokens, outputTokens, cacheHitTokens int64) {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	today := time.Now().Format("2006-01-02")
	stat, ok := memoryStats[endpointID]
	if !ok || stat.Date != today {
		// 如果内存中没有，或者日期变了，尝试从数据库获取或新建
		var dbStat models.APIStats
		err := models.DB.Where("api_endpoint_id = ? AND date = ?", endpointID, today).First(&dbStat).Error
		if err == gorm.ErrRecordNotFound {
			dbStat = models.APIStats{
				APIEndpointID: endpointID,
				Date:          today,
			}
		}
		stat = &dbStat
		memoryStats[endpointID] = stat
	}

	stat.CallCount++
	stat.InputTokens += inputTokens
	stat.OutputTokens += outputTokens
	stat.CacheHitTokens += cacheHitTokens
	stat.LastUpdated = time.Now()
}

func SyncStatsToDB() {
	statsMutex.Lock()
	defer statsMutex.Unlock()

	for _, stat := range memoryStats {
		if stat.ID == 0 {
			models.DB.Create(stat)
		} else {
			models.DB.Save(stat)
		}
	}
}
