package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"testing"
)

const baseURL = "http://localhost:8080"

// 📝 Test Case 1: ทดสอบว่าดึงข้อมูลที่นั่งได้ปกติหรือไม่ (API /seats)
func TestGetSeats(t *testing.T) {
	resp, err := http.Get(baseURL + "/seats")
	if err != nil {
		t.Fatalf("❌ ติดต่อ Server ไม่ได้: %v (รัน Docker ไว้หรือยัง?)", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("❌ คาดหวัง Status 200 แต่ได้ %v", resp.StatusCode)
	} else {
		t.Log("✅ ดึงข้อมูลที่นั่งสำเร็จ (Status 200)")
	}
}

// 📝 Test Case 2: ทดสอบจำลองผู้ใช้ 10 คน แย่งกันกดล็อก "ที่นั่งเดียวกัน" ในเสี้ยววินาที
func TestConcurrentLocking(t *testing.T) {
	seatID := "A1" // สมมติว่าทุกคนแย่งกันจองที่นั่ง A1
	successCount := 0
	conflictCount := 0

	var wg sync.WaitGroup
	var mu sync.Mutex

	t.Logf("🚀 เริ่มจำลองผู้ใช้ 10 คน แย่งกันจองที่นั่ง %s พร้อมกัน...", seatID)

	// จำลองยิง Request พร้อมกัน 10 รอบ (Goroutines)
	for i := 1; i <= 10; i++ {
		wg.Add(1)
		go func(userID string) {
			defer wg.Done()

			// สร้างข้อมูล Request
			reqBody, _ := json.Marshal(map[string]string{
				"seat_id": seatID,
				"user_id": userID,
			})

			req, _ := http.NewRequest("POST", baseURL+"/lock", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-User-ID", userID)
			req.Header.Set("X-User-Email", "tester@test.com")

			// ยิง API
			client := &http.Client{}
			resp, err := client.Do(req)

			if err == nil {
				mu.Lock()
				if resp.StatusCode == http.StatusOK {
					successCount++
					t.Logf("🟢 %s ล็อกที่นั่งสำเร็จ!", userID)
				} else if resp.StatusCode == http.StatusConflict {
					conflictCount++
					t.Logf("🔴 %s ล็อกไม่ทัน (ติด Conflict)", userID)
				}
				mu.Unlock()
				resp.Body.Close()
			}
		}(fmt.Sprintf("user_test_%d", i))
	}

	wg.Wait() // รอให้ทุกคนกดยิง API จนเสร็จ

	// ตรวจสอบผลลัพธ์
	t.Logf("📊 สรุปผล: สำเร็จ %d คน, ล็อกไม่ทัน %d คน", successCount, conflictCount)

	// หัวใจสำคัญ: ระบบที่ดี ต้องมีคนจองสำเร็จได้แค่ "คนเดียว" เท่านั้น
	if successCount > 1 {
		t.Errorf("❌ ระบบพัง! มีคนจองที่นั่งเดียวกันได้มากกว่า 1 คน (%d คน)", successCount)
	} else if successCount == 0 {
		t.Errorf("❌ ไม่มีใครจองสำเร็จเลย อาจจะติด Lock ค้างจากเทสต์รอบก่อน")
	} else {
		t.Log("✅ ระบบ Redis ป้องกันการแย่งจอง (Race Condition) ได้สมบูรณ์แบบ!")
	}
}
