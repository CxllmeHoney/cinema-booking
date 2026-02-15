# Cinema Ticket Booking System

ระบบจำลองการจองตั๋วหนังออนไลน์ที่เน้นการจัดการ **High Concurrency** เพื่อป้องกันปัญหาการจองซ้อน (Double Booking) ด้วย Redis Distributed Lock

![Status](https://img.shields.io/badge/Status-Completed-success)
![Go](https://img.shields.io/badge/Backend-Go-blue?logo=go&logoColor=white)
![Vue](https://img.shields.io/badge/Frontend-Vue.js-green?logo=vue.js&logoColor=white)
![Redis](https://img.shields.io/badge/Cache-Redis-red?logo=redis&logoColor=white)
![MongoDB](https://img.shields.io/badge/DB-MongoDB-47A248?logo=mongodb&logoColor=white)
![Docker](https://img.shields.io/badge/Infrastructure-Docker-2496ED?logo=docker&logoColor=white)
![WebSocket](https://img.shields.io/badge/Realtime-WebSocket-orange)
![Firebase](https://img.shields.io/badge/Auth-Firebase-FFCA28?logo=firebase&logoColor=black)

## System Architecture

ระบบถูกออกแบบเป็น Containerized Service เชื่อมต่อกันดังนี้:

```
User((User)) -->|HTTP/WebSocket| Frontend[Vue.js Frontend];
Frontend -->|REST API| Backend[Go Backend];
Backend -->|Read/Write| Mongo[(MongoDB)];
Backend -->|Distributed Lock| Redis[(Redis Cache)];
Backend -->|Pub/Sub Events| Redis;
```
## Tech Stack
Backend: Go (Gin Framework)

Frontend: Vue 3 (Composition API + Vite)

Database: MongoDB (Booking Data & Audit Logs)

Cache & Lock: Redis (Distributed Lock & Pub/Sub)

Real-time: WebSocket (Gorilla WebSocket)

Infrastructure: Docker & Docker Compose

## How to Run
ระบบทั้งหมดรวมอยู่ใน Docker Compose เรียบร้อยแล้ว สามารถรันด้วยคำสั่ง:
```
Bash
docker-compose up --build
```
Access Application:

Frontend: http://localhost:5173

Backend API: http://localhost:8080

## Admin Access
ระบบแยกสิทธิ์ User และ Admin โดยใช้ Environment Variable ในการกำหนดตัวตน Admin เพื่อความสะดวกในการทดสอบ

ขั้นตอนการตั้งค่า Admin:

1.เปิดไฟล์ docker-compose.yml

2.ไปที่ service backend > environment

3.แก้ไขค่า ADMIN_EMAIL ให้ตรงกับ Gmail ที่ต้องการใช้ทดสอบ
```
YAML
environment: - ADMIN_EMAIL=your.email@gmail.com
```
4.รันคำสั่ง docker-compose up --build ใหม่อีกครั้ง

5.เมื่อล็อกอินที่หน้า Frontend ด้วยอีเมลดังกล่าว เมนู Admin Dashboard จะปรากฏขึ้น

## Booking Logic & Concurrency Control
กระบวนการจองที่นั่งถูกออกแบบให้รองรับ Race Condition ดังนี้:

1.Select Seat: ผู้ใช้เลือกที่นั่งที่มีสถานะ AVAILABLE

2.Acquire Lock: Backend จะพยายามสร้าง Lock ใน Redis ด้วยคำสั่ง SETNX (Set if Not Exists)

- Key: lock:seat:{seat_id}

- Value: user_id

- TTL: 5 นาที (ป้องกัน Deadlock กรณี Service ตายหรือ User หาย)

3.Validation:

- ✅ Success: ถ้า SETNX สำเร็จ -> เปลี่ยนสถานะเป็น LOCKED และ Broadcast ผ่าน WebSocket

- ❌ Fail: ถ้า SETNX ล้มเหลว -> แสดงว่ามีคนอื่นกำลังทำรายการอยู่ -> Reject Request

4.Confirm / Timeout:

- Confirm: หากชำระเงินทันเวลา -> บันทึกลง MongoDB -> ลบ Lock -> Broadcast สถานะ BOOKED

- Timeout: หากครบ 5 นาที -> Redis ลบ Key อัตโนมัติ -> ระบบคืนสถานะเป็น AVAILABLE

## Redis Lock Strategy
ระบบเลือกใช้ Optimistic Locking ผ่านคำสั่ง SETNX ของ Redis เนื่องจาก:

- Atomicity: รับประกันว่าจะมี Request เดียวเท่านั้นที่เขียน Key สำเร็จในช่วงเวลาเดียวกัน ป้องกัน Race Condition ได้สมบูรณ์

- TTL Safety: การกำหนด Time-to-Live ช่วยป้องกัน Deadlock ในระบบ Distributed System ได้อย่างมีประสิทธิภาพ

## Message Queue Use Case
ระบบใช้ Redis Pub/Sub เพื่อทำ Asynchronous Audit Logging:

- Pattern: Fire-and-Forget

- Publisher: API Server ส่ง message ไปยัง Channel booking_events เมื่อเกิดเหตุการณ์สำคัญ (BOOKING_SUCCESS, BOOKING_TIMEOUT)

- Subscriber: Background Worker (Go Routine) ดักฟัง Channel และบันทึก Log ลง MongoDB audit_logs

- Benefit: ลด Latency ของ API หลัก โดยไม่ต้องรอ Database Write Operation ใน Flow การจอง

## Assumptions & Trade-offs
```
เพื่อให้ส่งมอบงานได้ทันภายในระยะเวลาที่กำหนด จึงมีการตัดสินใจทางเทคนิคTrade-offs ดังนี้:

Authentication:
Implementation: Frontend ใช้ Firebase Auth แต่ Backend ตรวจสอบสิทธิ์ผ่าน Header X-User-Email (โดยเทียบกับ Env Var)

Trade-off: เพื่อความรวดเร็วในการ Implement Demo (Production ควร Verify ID Token กับ Firebase Admin SDK)

Payment Gateway:
Implementation: จำลองการชำระเงิน ด้วยการกด Confirm
เหตุผล : เพื่อโฟกัสที่ Concurrency Logic และ System Design เป็นหลัก
```

## Project Structure
``````
├── backend/            # Source code ฝั่ง Go
│   ├── main.go         # Entry point & Business Logic
│   ├── Dockerfile
│   └── go.mod
│   └── go.sum
├── frontend/           # Source code ฝั่ง Vue.js
│   ├── src/
│   │   └── App.vue     # UI Logic
│   └── Dockerfile
└── docker-compose.yml  # Orchestration Config
``````