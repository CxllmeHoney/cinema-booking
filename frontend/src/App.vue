<script setup>
import { ref, onMounted } from 'vue'
import { initializeApp } from "firebase/app";
import { getAuth, signInWithPopup, GoogleAuthProvider, signOut } from "firebase/auth";

const firebaseConfig = {
  apiKey: "AIzaSyCjdcifmwfImx4XEXQlySuHuzEFda56D8o",
  authDomain: "cinema-booking-a46d3.firebaseapp.com",
  projectId: "cinema-booking-a46d3",
  storageBucket: "cinema-booking-a46d3.firebasestorage.app",
  messagingSenderId: "683612178493",
  appId: "1:683612178493:web:025026d3e671d392620254"
};

const app = initializeApp(firebaseConfig);
const auth = getAuth(app);
const provider = new GoogleAuthProvider();

const currentUser = ref(null);
const currentView = ref('booking');
const adminLogs = ref([]);
const filterUserID = ref('');
const filterEvent = ref('');

const seats = ref([
  { id: 'A1', status: 'available' }, { id: 'A2', status: 'available' }, { id: 'A3', status: 'available' },
  { id: 'B1', status: 'available' }, { id: 'B2', status: 'available' }, { id: 'B3', status: 'available' },
])

onMounted(async () => {
  const ws = new WebSocket("ws://localhost:8080/ws");
  ws.onmessage = (event) => {
    const data = JSON.parse(event.data);
    const targetSeat = seats.value.find(s => s.id === data.seat_id);
    if (targetSeat) targetSeat.status = data.status;
  };
  fetchSeats();
});

const fetchSeats = async () => {
  try {
    const res = await fetch('http://localhost:8080/seats');
    const allBookings = await res.json();
    allBookings.forEach(booking => {
      const seat = seats.value.find(s => s.id === booking.seat_id);
      if (seat) seat.status = booking.status;
    });
  } catch (e) { console.error(e); }
}

const loginWithGoogle = async () => {
  try {
    const result = await signInWithPopup(auth, provider);
    currentUser.value = result.user;
  } catch (error) { console.error("Login Failed:", error); }
}

const logout = async () => {
  await signOut(auth);
  currentUser.value = null;
  currentView.value = 'booking';
}

async function selectSeat(seat) {
  if (!currentUser.value) return alert("กรุณาล็อกอินก่อนจองที่นั่งครับ!");
  if (seat.status !== 'available') return;

  const userHeaders = {
    'Content-Type': 'application/json',
    'X-User-ID': currentUser.value.uid,
    'X-User-Email': currentUser.value.email
  };

  try {
    const lockResponse = await fetch('http://localhost:8080/lock', {
      method: 'POST', headers: userHeaders,
      body: JSON.stringify({ seat_id: seat.id, user_id: currentUser.value.uid })
    });

    if (!lockResponse.ok) return alert("❌ " + (await lockResponse.json()).message);

    const isConfirmPay = window.confirm(`ล็อกที่นั่ง ${seat.id} สำเร็จ! ต้องการชำระเงินเลยหรือไม่?`);
    if (isConfirmPay) {
      const confirmResponse = await fetch('http://localhost:8080/confirm', {
        method: 'POST', headers: userHeaders,
        body: JSON.stringify({ seat_id: seat.id, user_id: currentUser.value.uid })
      });
      alert(confirmResponse.ok ? "✅ จองสำเร็จ!" : "❌ " + (await confirmResponse.json()).message);
      fetchSeats();
    }
  } catch (error) { alert("ติดต่อ Server ไม่ได้"); }
}

const loadAdminDashboard = async () => {
  currentView.value = 'admin';
  try {
    const params = new URLSearchParams();
    if (filterUserID.value) params.append('user_id', filterUserID.value);
    if (filterEvent.value) params.append('event', filterEvent.value);

    const res = await fetch(`http://localhost:8080/admin/dashboard?${params.toString()}`, {
      headers: {
        'X-User-ID': currentUser.value.uid,
        'X-User-Email': currentUser.value.email
      }
    });

    if (res.ok) {
      const data = await res.json();
      adminLogs.value = data.audit_logs;
    } else {
      alert("❌ คุณไม่มีสิทธิ์เข้าถึงหน้านี้ (เฉพาะ Admin)");
      currentView.value = 'booking';
    }
  } catch (e) { console.error(e); }
}
</script>

<template>
  <div class="cinema-container">
    <h1>🎬 จองตั๋วหนัง (Demo)</h1>

    <div class="auth-box">
      <div v-if="!currentUser">
        <button @click="loginWithGoogle" class="btn-login">🔑 ล็อกอินด้วย Google</button>
      </div>
      <div v-else>
        <p>ยินดีต้อนรับ, <b>{{ currentUser.displayName }}</b></p>
        <button @click="currentView = 'booking'" class="btn-menu">🎟️ หน้าจองตั๋ว</button>
        <button @click="loadAdminDashboard" class="btn-menu admin-btn">📊 Admin Dashboard</button>
        <button @click="logout" class="btn-logout">ออกจากระบบ</button>
      </div>
    </div>

    <div v-if="currentView === 'booking'">
      <div class="screen">จอภาพยนตร์</div>
      <div class="seat-grid">
        <button v-for="seat in seats" :key="seat.id" class="seat" :class="seat.status" @click="selectSeat(seat)">
          {{ seat.id }}
        </button>
      </div>
      <div class="legend">
        <span class="box available"></span> ว่าง
        <span class="box locked"></span> กำลังจอง (Lock)
        <span class="box booked"></span> จองแล้ว
      </div>
    </div>

    <div v-if="currentView === 'admin'" class="admin-panel">
      <h2>📊 ประวัติการทำรายการ (Audit Logs)</h2>
      <div class="filter-box">
        <input v-model="filterUserID" placeholder="🔎 ค้นหา User ID..." class="filter-input" />
        <select v-model="filterEvent" class="filter-input">
          <option value="">-- ทุกเหตุการณ์ --</option>
          <option value="BOOKING_SUCCESS">จองสำเร็จ (Success)</option>
          <option value="BOOKING_TIMEOUT">หมดเวลา (Timeout)</option>
          <option value="locked">ล็อกที่นั่ง (Locked)</option>
        </select>
        <button @click="loadAdminDashboard" class="btn-menu admin-btn">🔍 ค้นหา</button>
        <button @click="filterUserID = ''; filterEvent = ''; loadAdminDashboard()" class="btn-menu">❌ ล้างค่า</button>
      </div>
      <table class="log-table">
        <thead>
          <tr>
            <th>เวลา</th>
            <th>เหตุการณ์</th>
            <th>ที่นั่ง</th>
            <th>รายละเอียด</th>
            <th>User ID</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="log in adminLogs" :key="log.timestamp">
            <td>{{ new Date(log.timestamp).toLocaleString() }}</td>
            <td><span class="badge" :class="log.event">{{ log.event }}</span></td>
            <td>{{ log.seat_id }}</td>
            <td>{{ log.message }}</td>
            <td style="font-size: 0.8em; color: #666;">{{ log.user_id }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<style scoped>
.cinema-container {
  text-align: center;
  font-family: sans-serif;
  margin-top: 30px;
}

.screen {
  background: #333;
  color: white;
  padding: 10px;
  margin: 0 auto 30px;
  width: 200px;
  border-radius: 5px;
}

.auth-box {
  margin-bottom: 30px;
  padding: 15px;
  background: #f8f9fa;
  border-radius: 8px;
  display: inline-block;
}

button {
  cursor: pointer;
  border: none;
  padding: 8px 15px;
  border-radius: 5px;
  font-weight: bold;
  margin: 0 5px;
}

.btn-login {
  background-color: #4285F4;
  color: white;
}

.btn-menu {
  background-color: #e0e0e0;
  color: #333;
}

.admin-btn {
  background-color: #6f42c1;
  color: white;
}

.btn-logout {
  background-color: #dc3545;
  color: white;
}

.seat-grid {
  display: grid;
  grid-template-columns: repeat(3, 60px);
  gap: 10px;
  justify-content: center;
}

.seat {
  width: 60px;
  height: 60px;
  border: none;
  border-radius: 8px;
  cursor: pointer;
  font-weight: bold;
}

.seat.available {
  background-color: #ddd;
}

.seat.locked {
  background-color: #ffc107;
  cursor: not-allowed;
}

.seat.booked {
  background-color: #dc3545;
  color: white;
  cursor: not-allowed;
}

.legend {
  margin-top: 20px;
  display: flex;
  justify-content: center;
  gap: 15px;
}

.box {
  width: 20px;
  height: 20px;
  display: inline-block;
  vertical-align: middle;
  margin-right: 5px;
}

.box.available {
  background: #ddd;
}

.box.locked {
  background: #ffc107;
}

.box.booked {
  background: #dc3545;
}

.admin-panel {
  max-width: 900px;
  margin: 0 auto;
  text-align: left;
}

.log-table {
  width: 100%;
  border-collapse: collapse;
  margin-top: 15px;
  background: white;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.log-table th,
.log-table td {
  padding: 12px;
  border: 1px solid #ddd;
}

.log-table th {
  background-color: #f4f4f4;
  text-align: left;
}

.badge {
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.85em;
  font-weight: bold;
}

.badge.BOOKING_SUCCESS {
  background-color: #28a745;
  color: white;
}

.badge.BOOKING_TIMEOUT {
  background-color: #ffc107;
  color: black;
}

.filter-box {
  background: #f8f9fa;
  padding: 15px;
  border-radius: 8px;
  margin-bottom: 20px;
  display: flex;
  gap: 10px;
  align-items: center;
  border: 1px solid #ddd;
}

.filter-input {
  padding: 8px;
  border: 1px solid #ccc;
  border-radius: 4px;
}
</style>