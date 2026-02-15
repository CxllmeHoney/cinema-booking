# frontend

This template should help get you started developing with Vue 3 in Vite.

## Recommended IDE Setup

[VS Code](https://code.visualstudio.com/) + [Vue (Official)](https://marketplace.visualstudio.com/items?itemName=Vue.volar) (and disable Vetur).

## Recommended Browser Setup

- Chromium-based browsers (Chrome, Edge, Brave, etc.):
  - [Vue.js devtools](https://chromewebstore.google.com/detail/vuejs-devtools/nhdogjmejiglipccpnnnanhbledajbpd)
  - [Turn on Custom Object Formatter in Chrome DevTools](http://bit.ly/object-formatters)
- Firefox:
  - [Vue.js devtools](https://addons.mozilla.org/en-US/firefox/addon/vue-js-devtools/)
  - [Turn on Custom Object Formatter in Firefox DevTools](https://fxdx.dev/firefox-devtools-custom-object-formatters/)

## Customize configuration

See [Vite Configuration Reference](https://vite.dev/config/).

## Project Setup

```sh
npm install
```

### Compile and Hot-Reload for Development

```sh
npm run dev
```

### Compile and Minify for Production

```sh
npm run build
```


# 🎬 Cinema Ticket Booking System (Concurrency & Real-time)

ระบบจองตั๋วหนังออนไลน์ (Demo) ที่ออกแบบมาเพื่อรองรับการทำงานพร้อมกัน (Concurrency) ป้องกันปัญหาการจองซ้อน (Double Booking) และแสดงผลแบบ Real-time

![Status](https://img.shields.io/badge/Status-Completed-success)
![Tech](https://img.shields.io/badge/Go-Vue.js-Redis-MongoDB-blue)

## 🏗️ System Architecture

ระบบถูกออกแบบเป็น Containerized Service ทำงานร่วมกันดังนี้:

```mermaid
graph TD;
    User((User)) -->|HTTP/WebSocket| Frontend[Vue.js Frontend];
    Frontend -->|REST API| Backend[Go Backend];
    Backend -->|Read/Write| Mongo[(MongoDB)];
    Backend -->|Distributed Lock| Redis[(Redis Cache)];
    Backend -->|Pub/Sub Events| Redis;
🛠️ Tech Stack
Backend: Go (Gin Framework)

Frontend: Vue 3 (Composition API + Vite)

Database: MongoDB (เก็บข้อมูล Booking และ Audit Logs)

Cache & Lock: Redis (ใช้ทำ Distributed Lock และ Message Queue)

Real-time: WebSocket (Gorilla WebSocket)

Infrastructure: Docker & Docker Compose

🚀 How to Run (วิธีรันระบบ)
ระบบทั้งหมดถูกตั้งค่าไว้ใน Docker Compose แล้ว สามารถรันได้ด้วยคำสั่งเดียว:

1. Start System
Bash
docker-compose up --build
2. Access Application
Frontend (User Interface): http://localhost:5173

Backend API: http://localhost:8080

👮‍♂️ Admin Access (วิธีการเข้าใช้งาน Admin Dashboard)
ระบบมีการแยกสิทธิ์ User และ Admin โดยใช้ Environment Variable ในการกำหนดตัวตน Admin เพื่อความปลอดภัยและสะดวกในการทดสอบ

ขั้นตอนการตั้งค่า Admin:

เปิดไฟล์ docker-compose.yml

ไปที่ service backend -> environment

แก้ไขค่า ADMIN_EMAIL ให้เป็น Gmail ที่คุณจะใช้ล็อกอินทดสอบ

YAML
environment:
  - ADMIN_EMAIL=your.email@gmail.com
รันคำสั่ง docker-compose up --build อีกครั้ง

ล็อกอินที่ Frontend ด้วยอีเมลดังกล่าว เมนู Admin Dashboard จะปรากฏขึ้น

🔄 Booking Flow (Logic การทำงาน)
User Select Seat: ผู้ใช้เลือกที่นั่ง (สถานะ AVAILABLE)

Acquire Lock: Backend พยายามสร้าง Lock ใน Redis ด้วยคำสั่ง SETNX (Set if Not Exists)

Key: lock:seat:{seat_id}

Value: user_id

TTL: 5 นาที (ป้องกัน Deadlock)

Validation:

✅ Success: ถ้าได้ Lock -> เปลี่ยนสถานะเป็น LOCKED และ Broadcast บอกทุกคนผ่าน WebSocket

❌ Fail: ถ้า SETNX return false -> แสดงว่ามีคนอื่นกำลังจองอยู่ -> แจ้งเตือนผู้ใช้

Confirm / Timeout:

Confirm: หากชำระเงินทันเวลา -> บันทึกลง MongoDB -> ลบ Lock -> Broadcast สถานะ BOOKED

Timeout: หากครบ 5 นาทีแล้วไม่จ่ายเงิน -> Redis ลบ Key อัตโนมัติ -> Backend ตรวจสอบและ Broadcast สถานะ AVAILABLE

🔐 Redis Lock Strategy
ใช้กลยุทธ์ Optimistic Locking ผ่าน Redis SETNX เพื่อจัดการ Concurrency:

Why SETNX? เป็น Atomic Operation รับประกันว่าจะมี Request เดียวเท่านั้นที่เขียน Key สำเร็จในช่วงเวลานั้นๆ ป้องกัน Race Condition ได้สมบูรณ์

Why TTL (5 min)? เพื่อป้องกันกรณี Service ตาย หรือ User ปิดหน้าจอกะทันหัน ทำให้ Lock ค้างตลอดกาล (Deadlock)

📨 Message Queue Use Case
ใช้ Redis Pub/Sub เพื่อทำ Asynchronous Audit Logging:

Publisher: เมื่อเกิดเหตุการณ์สำคัญ (เช่น BOOKING_SUCCESS, BOOKING_TIMEOUT) API จะ Publish message ไปยัง Channel booking_events

Subscriber: มี Go Routine (Worker) แยกทำงานเบื้องหลัง คอยดักฟัง Channel นี้และบันทึก Log ลง MongoDB audit_logs

Benefit: ช่วยลด Response Time ของ API หลัก เพราะไม่ต้องรอ Database Write เสร็จในจังหวะที่ตอบกลับ User

⚖️ Assumptions & Trade-offs
เพื่อให้ส่งมอบงานได้ทันภายในระยะเวลาที่กำหนด จึงมีการตัดสินใจทางเทคนิค (Trade-offs) ดังนี้:

Authentication:

Implementation: Frontend ใช้ Firebase Auth แต่ Backend ตรวจสอบสิทธิ์ผ่าน Header X-User-Email (โดยเทียบกับ Env Var)

Trade-off: เพื่อความรวดเร็วในการ Implement Demo (Production ควร Verify ID Token กับ Firebase Admin SDK)

Payment Gateway:

Implementation: จำลองการชำระเงิน (Mock) ด้วยการกด Confirm

Reason: เพื่อโฟกัสที่ Concurrency Logic และ System Design เป็นหลัก

📂 Project Structure
Plaintext
.
├── backend/            # Source code ฝั่ง Go
│   ├── main.go         # Entry point & Business Logic
│   ├── Dockerfile
│   └── go.mod
├── frontend/           # Source code ฝั่ง Vue.js
│   ├── src/
│   │   └── App.vue     # UI Logic
│   └── Dockerfile
└── docker-compose.yml  # Orchestration Config