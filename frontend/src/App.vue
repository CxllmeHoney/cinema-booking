<script setup>
import { ref, onMounted, computed } from 'vue'
import { initializeApp } from "firebase/app";
import { getAuth, signInWithPopup, GoogleAuthProvider, signOut, onAuthStateChanged } from "firebase/auth";

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

const seats = ref([]);

const generateSeats = () => {
  const newSeats = [];
  
  const rows24 = ['K', 'J', 'I', 'H', 'G', 'F', 'E', 'D'];
  rows24.forEach(row => {
    for(let i = 1; i <= 24; i++) {
      newSeats.push({ id: `${row}${i}`, status: 'available', row: row, type: 'normal' });
    }
  });

  const rows20 = ['C', 'B', 'A'];
  rows20.forEach(row => {
    for(let i = 1; i <= 20; i++) {
      newSeats.push({ id: `${row}${i}`, status: 'available', row: row, type: 'normal' });
    }
  });

  for(let i = 1; i <= 10; i++) {
    newSeats.push({ id: `VP${i}`, status: 'available', row: 'VP', type: 'vip' });
  }

  seats.value = newSeats;
};

generateSeats();

const seatRows = computed(() => {
  const rowLabels = ['K', 'J', 'I', 'H', 'G', 'F', 'E', 'D', 'C', 'B', 'A', 'VP'];
  return rowLabels.map(label => ({
    label: label,
    seats: seats.value.filter(s => s.row === label)
  }));
});

onMounted(async () => {
  onAuthStateChanged(auth, (user) => {
    if (user) {
      currentUser.value = user; // ถ้ามีประวัติล็อกอิน ให้จำค่า User ไว้เลย
    } else {
      currentUser.value = null; // ถ้าไม่มีก็เคลียร์ค่าทิ้ง
    }
  });

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
  <div class="cinema-wrapper">
    <header class="top-nav">
      <div class="nav-left">
        <h1 class="main-title"><span class="icon">🎬</span>Cinema-Booking</h1>
      </div>

      <div class="nav-right">
        <div v-if="!currentUser">
          <button @click="loginWithGoogle" class="btn btn-login">
            <svg viewBox="0 0 24 24" width="20" height="20" xmlns="http://www.w3.org/2000/svg"><path d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z" fill="#4285F4"/><path d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z" fill="#34A853"/><path d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z" fill="#FBBC05"/><path d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z" fill="#EA4335"/></svg>
            ล็อกอินด้วย Google
          </button>
        </div>
        
        <div v-else class="user-menu">
          <button @click="currentView = 'booking'" class="btn btn-menu" :class="{ active: currentView === 'booking' }">🎟️ หน้าจองตั๋ว</button>
          <button @click="loadAdminDashboard" class="btn btn-menu admin-btn" :class="{ active: currentView === 'admin' }">📊 Admin Dashboard</button>
          
          <div class="user-profile">
            <img :src="currentUser.photoURL || 'https://via.placeholder.com/40'" class="user-avatar" />
            <span class="user-name">{{ currentUser.displayName }}</span>
          </div>
          <button @click="logout" class="btn btn-logout">ออกจากระบบ</button>
        </div>
      </div>
    </header>

    <main class="main-content">
      <div v-if="currentView === 'booking'" class="booking-section fade-in">
        <div class="screen-container">
          <div class="screen">SCREEN</div>
          <div class="screen-glow"></div>
        </div>
        
        <div class="cinema-layout">
          <div v-for="rowGroup in seatRows" :key="rowGroup.label" class="seat-row-wrapper">
            <div class="row-label">{{ rowGroup.label }}</div>

            <div class="seats-row" :class="{'is-vp': rowGroup.label === 'VP'}">
              <button
                v-for="seat in rowGroup.seats"
                :key="seat.id"
                class="seat-btn"
                :class="[seat.status, seat.type]"
                @click="selectSeat(seat)"
                :title="`ที่นั่ง ${seat.id}`"
              >
                <svg v-if="seat.status === 'available'" class="icon-svg chair" viewBox="0 0 24 24">
                  <path fill="currentColor" d="M4 18v3h2v-3h12v3h2v-6H4v3zm15-8h-1V6c0-1.1-.9-2-2-2H8c-1.1 0-2 .9-2 2v4H5c-1.1 0-2 .9-2 2v4h22v-4c0-1.1-.9-2-2-2z"/>
                </svg>

                <svg v-else-if="seat.status === 'locked'" class="icon-svg locked" viewBox="0 0 24 24">
                  <path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm0-14c-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4-1.79-4-4-4zm0 6c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm0 3c-2.67 0-8 1.34-8 4v1h16v-1c0-2.66-5.33-4-8-4zm-6 3.12c.98-.67 3.5-1.12 6-1.12s5.02.45 6 1.12H6z"/>
                </svg>

                <svg v-else class="icon-svg booked" viewBox="0 0 24 24">
                   <path fill="currentColor" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm0-14c-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4-1.79-4-4-4zm0 6c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm0 3c-2.67 0-8 1.34-8 4v1h16v-1c0-2.66-5.33-4-8-4zm-6 3.12c.98-.67 3.5-1.12 6-1.12s5.02.45 6 1.12H6z"/>
                </svg>
              </button>
            </div>

            <div class="row-label">{{ rowGroup.label }}</div>
          </div>
        </div>

        <div class="legend">
          <div class="legend-item">
            <svg class="legend-icon" viewBox="0 0 24 24"><path fill="#b91c1c" d="M4 18v3h2v-3h12v3h2v-6H4v3zm15-8h-1V6c0-1.1-.9-2-2-2H8c-1.1 0-2 .9-2 2v4H5c-1.1 0-2 .9-2 2v4h22v-4c0-1.1-.9-2-2-2z"/></svg> ว่าง (Available)
          </div>
          <div class="legend-item">
            <svg class="legend-icon" viewBox="0 0 24 24"><path fill="#be185d" d="M4 18v3h2v-3h12v3h2v-6H4v3zm15-8h-1V6c0-1.1-.9-2-2-2H8c-1.1 0-2 .9-2 2v4H5c-1.1 0-2 .9-2 2v4h22v-4c0-1.1-.9-2-2-2z"/></svg> VIP (Available)
          </div>
          <div class="legend-item">
            <svg class="legend-icon" viewBox="0 0 24 24"><path fill="#f59e0b" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm0-14c-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4-1.79-4-4-4zm0 6c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm0 3c-2.67 0-8 1.34-8 4v1h16v-1c0-2.66-5.33-4-8-4zm-6 3.12c.98-.67 3.5-1.12 6-1.12s5.02.45 6 1.12H6z"/></svg> กำลังจอง (Locked)
          </div>
          <div class="legend-item">
            <svg class="legend-icon" viewBox="0 0 24 24"><path fill="#64748b" d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm0-14c-2.21 0-4 1.79-4 4s1.79 4 4 4 4-1.79 4-4-1.79-4-4-4zm0 6c-1.1 0-2-.9-2-2s.9-2 2-2 2 .9 2 2-.9 2-2 2zm0 3c-2.67 0-8 1.34-8 4v1h16v-1c0-2.66-5.33-4-8-4zm-6 3.12c.98-.67 3.5-1.12 6-1.12s5.02.45 6 1.12H6z"/></svg> จองแล้ว (Booked)
          </div>
        </div>
      </div>

      <div v-if="currentView === 'admin'" class="admin-panel fade-in">
        <h2><span class="icon">📊</span> ประวัติการทำรายการ (Audit Logs)</h2>
        
        <div class="filter-box">
          <div class="input-group">
            <span class="input-icon">🔎</span>
            <input v-model="filterUserID" placeholder="ค้นหาด้วย User ID..." class="filter-input" />
          </div>
          <div class="input-group">
            <span class="input-icon">📁</span>
            <select v-model="filterEvent" class="filter-input">
              <option value="">-- ทุกเหตุการณ์ --</option>
              <option value="BOOKING_SUCCESS">จองสำเร็จ (Success)</option>
              <option value="BOOKING_TIMEOUT">หมดเวลา (Timeout)</option>
              <option value="locked">ล็อกที่นั่ง (Locked)</option>
            </select>
          </div>
          <button @click="loadAdminDashboard" class="btn btn-search">ค้นหา</button>
          <button @click="filterUserID = ''; filterEvent = ''; loadAdminDashboard()" class="btn btn-clear">ล้างค่า</button>
        </div>

        <div class="table-container">
          <table class="log-table">
            <thead>
              <tr>
                <th>วัน-เวลา</th>
                <th>เหตุการณ์</th>
                <th>ที่นั่ง</th>
                <th>รายละเอียด</th>
                <th>User ID</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="log in adminLogs" :key="log.timestamp">
                <td class="time-col">{{ new Date(log.timestamp).toLocaleString() }}</td>
                <td><span class="badge" :class="log.event">{{ log.event }}</span></td>
                <td class="seat-col">{{ log.seat_id }}</td>
                <td>{{ log.message }}</td>
                <td class="user-id-col">{{ log.user_id }}</td>
              </tr>
              <tr v-if="adminLogs.length === 0">
                <td colspan="5" class="empty-state">ไม่พบข้อมูลในระบบ</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </main>
  </div>
</template>

<style>
@import url('https://fonts.googleapis.com/css2?family=Kanit:wght@300;400;500;600&display=swap');

html, body {
  margin: 0;
  padding: 0;
  background-color: #000000; 
}
#app {
  max-width: 100% !important;
  width: 100%;
  margin: 0 !important;
  padding: 0 !important;
}
</style>

<style scoped>
.cinema-wrapper {
  background-color: #000000;
  color: #f8fafc;
  min-height: 100vh;
  width: 100%;
  font-family: 'Kanit', sans-serif;
  display: flex;
  flex-direction: column;
}

/* --- Header / Navigation Bar --- */
.top-nav {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px 40px;
  background: rgba(15, 23, 42, 0.9);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
  position: sticky;
  top: 0;
  z-index: 100;
}

.main-title { font-size: 1.5rem; font-weight: 600; margin: 0; color: #fff; }
.user-menu { display: flex; align-items: center; gap: 15px; }
.user-profile { display: flex; align-items: center; gap: 10px; padding: 0 15px; border-left: 1px solid #334155; border-right: 1px solid #334155; }
.user-avatar { width: 35px; height: 35px; border-radius: 50%; border: 2px solid #38bdf8; }
.user-name { font-size: 0.95rem; font-weight: 500; color: #fff; }

/* --- Buttons --- */
.btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 8px;
  cursor: pointer; border: none; padding: 8px 16px; border-radius: 8px;
  font-family: 'Kanit', sans-serif; font-weight: 500; font-size: 0.95rem; transition: 0.2s;
}
.btn:hover { transform: translateY(-2px); }
.btn-login { background-color: #ffffff; color: #333; }
.btn-menu { background-color: transparent; color: #94a3b8; }
.btn-menu:hover { color: #fff; }
.btn-menu.active { background-color: rgba(56, 189, 248, 0.1); color: #38bdf8; font-weight: 600; }
.admin-btn.active { background-color: rgba(139, 92, 246, 0.1); color: #a78bfa; }
.btn-logout { background-color: transparent; color: #ef4444; border: 1px solid #ef4444; }

/* --- Main Content --- */
.main-content {
  flex: 1;
  padding: 40px 20px;
  width: 100%;
  margin: 0 auto;
}

.screen-container { position: relative; margin: 20px auto 50px; text-align: center;}
.screen {
  background: linear-gradient(to bottom, #cbd5e1, #64748b);
  color: #0f172a; padding: 10px; margin: 0 auto;
  width: 80%; max-width: 800px; border-radius: 10px; font-weight: 600; letter-spacing: 5px;
  transform: perspective(300px) rotateX(-5deg);
  box-shadow: 0 20px 50px rgba(255, 255, 255, 0.1); z-index: 2; position: relative;
}
.screen-glow {
  position: absolute; top: 10px; left: 10%; width: 80%; height: 40px;
  background: #cbd5e1; filter: blur(40px); opacity: 0.1; z-index: 1;
}

.cinema-layout {
  display: flex;
  flex-direction: column;
  gap: 12px;
  overflow-x: auto;
  padding-bottom: 20px;
  align-items: center;
}

.seat-row-wrapper {
  display: flex;
  align-items: center;
  gap: 15px; 
}

.row-label {
  color: #cbd5e1;
  font-family: sans-serif;
  font-size: 14px;
  font-weight: 600;
  width: 25px;
  text-align: center;
}

.seats-row {
  display: flex;
  gap: 8px; 
}

.seats-row.is-vp {
  gap: 0;
  margin-top: 15px; 
}

/* --- Seat Button & Icons --- */
.seat-btn {
  background: transparent;
  border: none;
  padding: 0;
  margin: 0;
  cursor: pointer;
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.2s ease;
}

.seat-btn:hover:not(.booked) {
  transform: scale(1.2);
}

.icon-svg {
  width: 100%;
  height: 100%;
}

.seat-btn.available.normal .icon-svg.chair { color: #b91c1c; } 
.seat-btn.available.vip .icon-svg.chair { color: #be185d; } 

.seat-btn.locked .icon-svg.locked { 
  color: #f59e0b;
  animation: pulse 1s infinite alternate; 
}

.seat-btn.booked .icon-svg.booked { 
  color: #64748b; 
  cursor: not-allowed;
}

.seat-btn.vip {
  margin-right: 8px;
}
.seat-btn.vip:nth-child(2n) {
  margin-right: 40px; 
}
.seat-btn.vip:last-child {
  margin-right: 0;
}

@keyframes pulse {
  0% { transform: scale(1); opacity: 0.8; }
  100% { transform: scale(1.15); opacity: 1; text-shadow: 0 0 10px #f59e0b; }
}

.legend {
  display: flex; justify-content: center; gap: 25px; flex-wrap: wrap;
  margin-top: 40px; padding: 15px; border-radius: 12px;
}
.legend-item { display: flex; align-items: center; font-size: 0.95rem; color: #cbd5e1; gap: 8px; }
.legend-icon { width: 24px; height: 24px; }

/* --- Admin Panel --- */
.admin-panel { background: #0f172a; padding: 30px; border-radius: 16px; box-shadow: 0 10px 30px rgba(0,0,0,0.5); max-width: 900px; margin: 0 auto;}
.admin-panel h2 { margin-top: 0; margin-bottom: 20px; border-bottom: 1px solid #1e293b; padding-bottom: 15px; }
.filter-box { display: flex; flex-wrap: wrap; gap: 15px; background: #1e293b; padding: 15px; border-radius: 12px; margin-bottom: 25px; align-items: center; }
.input-group { display: flex; align-items: center; background: #0f172a; border: 1px solid #334155; border-radius: 8px; overflow: hidden; flex: 1; min-width: 200px; }
.input-icon { padding: 10px; background: #334155; font-size: 0.9rem; }
.filter-input { border: none; background: transparent; color: white; padding: 10px; width: 100%; outline: none; font-family: 'Kanit', sans-serif; }
.filter-input option { background: #0f172a; }
.btn-search { background-color: #38bdf8; color: #0f172a; font-weight: 600; }
.btn-clear { background-color: transparent; border: 1px solid #64748b; color: #cbd5e1; }
.log-table { width: 100%; border-collapse: collapse; background: #1e293b; border-radius: 12px; overflow: hidden; }
.log-table th, .log-table td { padding: 15px; border-bottom: 1px solid #0f172a; font-size: 0.95rem; }
.log-table th { background-color: #334155; color: #94a3b8; font-weight: 500; text-transform: uppercase; font-size: 0.85rem; letter-spacing: 1px; text-align: left;}
.seat-col { font-weight: bold; color: #38bdf8; }
.time-col { color: #94a3b8; font-size: 0.85rem; }
.user-id-col { font-family: monospace; color: #64748b; font-size: 0.85rem; }
.empty-state { text-align: center; color: #64748b; padding: 30px !important; }
.badge { padding: 5px 10px; border-radius: 6px; font-size: 0.8rem; font-weight: 600; display: inline-block; }
.badge.BOOKING_SUCCESS { background-color: rgba(16, 185, 129, 0.2); color: #10b981; border: 1px solid #10b981; }
.badge.BOOKING_TIMEOUT { background-color: rgba(245, 158, 11, 0.2); color: #f59e0b; border: 1px solid #f59e0b; }
.badge.locked { background-color: rgba(56, 189, 248, 0.2); color: #38bdf8; border: 1px solid #38bdf8; }
.fade-in { animation: fadeIn 0.4s ease-in-out; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(10px); } to { opacity: 1; transform: translateY(0); } }
</style>