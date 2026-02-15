package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var ctx = context.Background()
var rdb *redis.Client
var mongoCollection *mongo.Collection
var auditCollection *mongo.Collection

// Helper อ่าน Env
func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}

var upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
var clients = make(map[*websocket.Conn]bool)
var clientsMutex sync.Mutex

// Structs
type Booking struct {
	SeatID    string    `bson:"seat_id" json:"seat_id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	Status    string    `bson:"status"  json:"status"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

type ConfirmRequest struct {
	SeatID string `json:"seat_id"`
	UserID string `json:"user_id"`
}

type AuditLog struct {
	Event     string    `bson:"event" json:"event"`
	SeatID    string    `bson:"seat_id" json:"seat_id"`
	UserID    string    `bson:"user_id" json:"user_id"`
	Timestamp time.Time `bson:"timestamp" json:"timestamp"`
	Message   string    `bson:"message" json:"message"`
}

func main() {
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	mongoURI := getEnv("MONGO_URI", "mongodb://localhost:27017")

	// 1. Redis Connect
	rdb = redis.NewClient(&redis.Options{Addr: redisAddr})

	// 2. Mongo Connect
	clientOptions := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	mongoCollection = client.Database("cinema").Collection("bookings")
	auditCollection = client.Database("cinema").Collection("audit_logs")

	// 3. Start MQ Subscriber
	go startMQSubscriber()

	// 4. Gin Setup
	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "X-User-ID", "X-User-Email"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
	}))

	// Public Routes
	r.GET("/seats", getSeats)
	r.GET("/ws", handleWebSocket)

	// User Routes
	userRoutes := r.Group("/")
	userRoutes.Use(AuthMiddleware("USER"))
	{
		userRoutes.POST("/lock", lockSeat)
		userRoutes.POST("/confirm", confirmBooking)
	}

	// Admin Routes
	adminRoutes := r.Group("/admin")
	adminRoutes.Use(AuthMiddleware("ADMIN"))
	{
		adminRoutes.GET("/dashboard", getAdminDashboard)
	}

	fmt.Println("🚀 Server running on :8080")
	r.Run(":8080")
}

// --- Middleware & Handlers ---

func AuthMiddleware(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID := c.GetHeader("X-User-ID")
		userEmail := c.GetHeader("X-User-Email")

		if userID == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		role := "USER"

		adminEmailEnv := getEnv("ADMIN_EMAIL", "")
		if adminEmailEnv != "" && userEmail == adminEmailEnv {
			role = "ADMIN"
		}
		if userEmail == "peemawat8685@gmail.com" {
			role = "ADMIN"
		}
		if requiredRole == "ADMIN" && role != "ADMIN" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: สำหรับ Admin เท่านั้น"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("userRole", role)
		c.Next()
	}
}

func getSeats(c *gin.Context) {
	bookings := []Booking{}
	cursor, err := mongoCollection.Find(ctx, bson.M{})
	if err == nil {
		cursor.All(ctx, &bookings)
	}

	keys, err := rdb.Keys(ctx, "lock:seat:*").Result()
	if err == nil {
		for _, key := range keys {
			seatID := key[10:]
			isBooked := false
			for _, b := range bookings {
				if b.SeatID == seatID {
					isBooked = true
					break
				}
			}
			if !isBooked {
				bookings = append(bookings, Booking{SeatID: seatID, Status: "locked"})
			}
		}
	}
	c.JSON(200, bookings)
}

func lockSeat(c *gin.Context) {
	var req Booking
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Data"})
		return
	}

	req.UserID = c.MustGet("userID").(string)
	lockKey := fmt.Sprintf("lock:seat:%s", req.SeatID)

	success, err := rdb.SetNX(ctx, lockKey, req.UserID, 5*time.Minute+10*time.Second).Result()
	if err != nil {
		c.JSON(500, gin.H{"error": "Redis connection failed"})
		return
	}

	if !success {
		c.JSON(409, gin.H{"status": "FAIL", "message": "ที่นั่งนี้กำลังถูกทำรายการโดยผู้อื่น"})
		return
	}

	broadcastUpdate(req.SeatID, "locked")

	go func(seatID, userID, key string) {
		time.Sleep(5 * time.Minute)

		val, _ := rdb.Get(ctx, key).Result()
		if val == userID {
			rdb.Del(ctx, key)
			broadcastUpdate(seatID, "available")

			logMsg, _ := json.Marshal(AuditLog{
				Event: "BOOKING_TIMEOUT", SeatID: seatID, UserID: userID,
				Timestamp: time.Now(), Message: "หมดเวลา 5 นาที คืนที่นั่งเข้าสู่ระบบ",
			})
			rdb.Publish(ctx, "booking_events", logMsg)
		}
	}(req.SeatID, req.UserID, lockKey)

	c.JSON(200, gin.H{"status": "SUCCESS", "message": "ล็อกที่นั่งสำเร็จ กรุณาชำระเงินใน 5 นาที"})
}

func confirmBooking(c *gin.Context) {
	var req ConfirmRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Data"})
		return
	}

	req.UserID = c.MustGet("userID").(string)
	lockKey := fmt.Sprintf("lock:seat:%s", req.SeatID)

	val, err := rdb.Get(ctx, lockKey).Result()
	if err != nil || val != req.UserID {
		c.JSON(400, gin.H{"status": "FAIL", "message": "Session หมดอายุ หรือไม่ใช่เจ้าของที่นั่ง"})
		return
	}

	booking := Booking{SeatID: req.SeatID, UserID: req.UserID, Status: "booked", CreatedAt: time.Now()}
	_, err = mongoCollection.InsertOne(ctx, booking)
	if err != nil {
		c.JSON(500, gin.H{"error": "Save to MongoDB failed"})
		return
	}

	rdb.Del(ctx, lockKey)
	broadcastUpdate(req.SeatID, "booked")

	logMsg, _ := json.Marshal(AuditLog{
		Event: "BOOKING_SUCCESS", SeatID: req.SeatID, UserID: req.UserID,
		Timestamp: time.Now(), Message: "ชำระเงินและจองที่นั่งสำเร็จ",
	})
	rdb.Publish(ctx, "booking_events", logMsg)

	c.JSON(200, gin.H{"status": "SUCCESS", "message": "จองและชำระเงินสำเร็จ!"})
}

func getAdminDashboard(c *gin.Context) {
	userID := c.Query("user_id")
	event := c.Query("event")

	filter := bson.M{}
	if userID != "" {
		filter["user_id"] = userID
	}
	if event != "" {
		filter["event"] = event
	}

	logs := []AuditLog{}
	cursor, err := auditCollection.Find(ctx, filter)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error fetching logs"})
		return
	}
	cursor.All(ctx, &logs)

	c.JSON(200, gin.H{
		"message":    "Admin Dashboard",
		"audit_logs": logs,
	})
}

func startMQSubscriber() {
	pubsub := rdb.Subscribe(ctx, "booking_events")
	defer pubsub.Close()
	ch := pubsub.Channel()

	fmt.Println("🎧 MQ Subscriber is listening...")
	for msg := range ch {
		var logEntry AuditLog
		if err := json.Unmarshal([]byte(msg.Payload), &logEntry); err == nil {
			auditCollection.InsertOne(ctx, logEntry)
			fmt.Printf("📝 [Log]: %s - Seat %s\n", logEntry.Event, logEntry.SeatID)
		}
	}
}

func handleWebSocket(c *gin.Context) {
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	defer ws.Close()

	clientsMutex.Lock()
	clients[ws] = true
	clientsMutex.Unlock()

	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			clientsMutex.Lock()
			delete(clients, ws)
			clientsMutex.Unlock()
			break
		}
	}
}

func broadcastUpdate(seatID string, status string) {
	clientsMutex.Lock()
	defer clientsMutex.Unlock()
	message := map[string]string{"seat_id": seatID, "status": status}
	for client := range clients {
		client.WriteJSON(message)
	}
}
