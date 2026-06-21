# 🚀 DiceDB Distributed System

# 👥 Thành viên nhóm
* Ngô Thị Minh Phương - 23012156
* Nguyễn Thị Thu Giang - 23010871

---

## 📌 Giới thiệu dự án

**DiceDB Distributed System** là dự án xây dựng và nghiên cứu một hệ thống cơ sở dữ liệu phân tán dựa trên DiceDB - một hệ thống cơ sở dữ liệu trong bộ nhớ (In-memory Database) với khả năng xử lý tốc độ cao.

Mục tiêu của dự án là tìm hiểu kiến trúc hệ thống phân tán, cơ chế giao tiếp giữa các node, quản lý trạng thái hệ thống và xây dựng giao diện Dashboard để giám sát hoạt động của hệ thống.

Dự án được thực hiện trong khuôn khổ môn học **Phân tích và Thiết kế Phần mềm**.

---

# 🎯 Mục tiêu dự án

* Tìm hiểu kiến trúc và nguyên lý hoạt động của hệ thống phân tán.
* Nghiên cứu mô hình Database dạng In-memory.
* Xây dựng môi trường chạy DiceDB.
* Quản lý nhiều node trong hệ thống.
* Theo dõi trạng thái hoạt động của các node.
* Hiển thị thông tin hệ thống thông qua giao diện Dashboard.
* Mô phỏng các tình huống lỗi và phục hồi trong hệ thống phân tán.

---

# 🏗️ Kiến trúc hệ thống

Hệ thống gồm các thành phần chính:

```
                    User
                      |
                      |
              Dice Dashboard
                      |
                      |
              API / Backend
                      |
        --------------------------------
        |              |               |
      Node 1         Node 2          Node 3
        |              |               |
        --------------------------------
              Distributed Storage
```

### Các thành phần:

### 🔹 DiceDB Server

* Xử lý dữ liệu trong bộ nhớ.
* Cung cấp khả năng lưu trữ và truy vấn dữ liệu nhanh.
* Hỗ trợ mô hình hoạt động phân tán.

### 🔹 Distributed Nodes

* Các node trong hệ thống.
* Trao đổi trạng thái và dữ liệu.
* Đảm bảo hệ thống có khả năng mở rộng.

### 🔹 Dashboard

Giao diện quản lý hệ thống cho phép:

* Theo dõi trạng thái node.
* Kiểm tra heartbeat.
* Quan sát hoạt động hệ thống.
* Quản lý và kiểm tra dữ liệu.

---

# ✨ Chức năng chính

## 1. Quản lý hệ thống

✔ Hiển thị thông tin cluster
✔ Theo dõi trạng thái các node
✔ Kiểm tra tình trạng hoạt động của hệ thống

## 2. Monitoring

Dashboard hỗ trợ:

* Heartbeat monitoring
* Node status
* System logs
* Replication status

## 3. Distributed Database

Hệ thống hỗ trợ:

* Lưu trữ dữ liệu dạng Key-Value
* Truy vấn dữ liệu nhanh
* Quản lý dữ liệu phân tán

## 4. Mô phỏng lỗi

Có thể kiểm tra:

* Node failure
* Node recovery
* Khả năng duy trì hoạt động hệ thống

---

# 🛠️ Công nghệ sử dụng

## Backend

* Golang
* DiceDB
* REST API

## Frontend Dashboard

* HTML
* CSS
* JavaScript

## DevOps

* Docker
* Docker Compose

## Version Control

* Git
* GitHub

---

# 📂 Cấu trúc thư mục

```
DiceDB-Distributed-System
│
├── cmd
│
├── config
│
├── internal
│
├── dice-dashboard
│   ├── handlers
│   ├── templates
│   ├── static
│   └── main.go
│
├── Dockerfile
│
├── go.mod
│
└── README.md
```

---

# ⚙️ Hướng dẫn chạy dự án

## 1. Clone repository

```bash
git clone <repository-url>

cd DiceDB-Distributed-System
```

---

# 2. Chạy DiceDB bằng Docker

```bash
docker run -d \
--name dicedb \
-p 7379:7379 \
dicedb/dicedb
```

Kiểm tra container:

```bash
docker ps
```

---

# 3. Chạy Dashboard

Di chuyển vào thư mục:

```bash
cd dice-dashboard
```

Cài đặt dependencies:

```bash
go mod tidy
```

Chạy hệ thống:

```bash
go run cmd/main.go
```

---

# 4. Truy cập giao diện

Mở trình duyệt:

```
http://localhost:8080
```

---

# 🎥 Demo Sản Phẩm

Video demo hệ thống:

👉 [Xem Video Demo](https://drive.google.com/drive/folders/1opp3j_PYhNYhlv258h5p-5itZmBeEd8t?usp=sharing]

---


# 📚 Kiến thức đạt được

Qua dự án này nhóm đã tìm hiểu:

* Kiến trúc hệ thống phân tán.
* Cơ chế giao tiếp giữa các node.
* Quản lý trạng thái trong distributed system.
* Cách triển khai ứng dụng bằng Docker.
* Thiết kế và xây dựng Dashboard giám sát hệ thống.

---


# 📄 License

Dự án được phát triển phục vụ mục đích học tập và nghiên cứu.
